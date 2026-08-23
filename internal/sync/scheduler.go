package sync

import (
	"context"
	"log/slog"
	"strconv"
	"sync"
	"time"

	"github.com/shiv-source/thoth/internal/store"
)

// SyncInterval is the default tick cadence of the auto-sync scheduler. It is
// fine to be coarse: each connection's due time is last_attempt_at + its own
// interval, checked on every tick, so a 30s cadence bounds how late a push
// can be.
const defaultSyncTick = 30 * time.Second

// Result is one completed scheduled sync, published on the event bus so the
// API layer can surface it as a notification.
type Result struct {
	ConnectionID int64
	Name         string
	OK           bool
	Error        string
}

// Scheduler periodically pushes enabled connections whose configured interval
// has elapsed since their last sync attempt. It is the "automatic sync"
// feature on top of the per-connection `enabled` switch and `interval` config
// field: a connection with enabled=1 and interval>0 is pushed on schedule; the
// first push is due immediately (a never-attempted connection is due now). It
// lives in serve, shares the Service.Push path with the API (so results land
// on the row + history exactly like a manual push), and reports outcomes to
// the caller via OnResult so the serve layer can publish notifications.
type Scheduler struct {
	svc    *Service
	root   string
	log    *slog.Logger
	tick   time.Duration
	max    int           // max concurrent pushes
	sem    chan struct{} // push concurrency cap
	done   chan struct{} // closed on Stop
	onPush func(Result)

	mu      sync.Mutex
	pushing map[int64]struct{} // connection IDs with an in-flight push
}

// NewScheduler builds the auto-sync scheduler. onPush receives each completed
// push's Result (nil skips reporting — the serve layer passes a publisher).
func NewScheduler(svc *Service, root string, log *slog.Logger, onPush func(Result)) *Scheduler {
	if log == nil {
		log = slog.New(slog.DiscardHandler)
	}
	return &Scheduler{
		svc:     svc,
		root:    root,
		log:     log,
		tick:    defaultSyncTick,
		max:     2,
		sem:     make(chan struct{}, 2),
		done:    make(chan struct{}),
		onPush:  onPush,
		pushing: make(map[int64]struct{}),
	}
}

// Start runs the scheduler until ctx is cancelled (server shutdown) or Stop.
// Every tick it scans connections and fires any enabled, due push; pushes run
// concurrently up to the cap, and a connection already in flight is skipped.
func (s *Scheduler) Start(ctx context.Context) {
	t := time.NewTicker(s.tick)
	defer t.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-s.done:
			return
		case <-t.C:
			s.sweep(ctx)
		}
	}
}

// Stop halts the scheduler after the current tick; in-flight pushes finish.
func (s *Scheduler) Stop() { close(s.done) }

// sweep evaluates every connection once: enabled + interval>0 + due (never
// attempted, or interval elapsed since the last attempt) → fire a push unless
// the connection is already pushing.
func (s *Scheduler) sweep(ctx context.Context) {
	if s.svc == nil || s.svc.Store == nil {
		return
	}
	conns, err := s.svc.Store.ListConnections()
	if err != nil {
		s.log.Warn("sync scheduler: list connections", "err", err)
		return
	}
	now := time.Now().UTC()
	for _, c := range conns {
		if !c.Enabled {
			continue
		}
		interval, ok := connectionInterval(c)
		if !ok {
			continue
		}
		if !s.due(c, interval, now) {
			continue
		}
		s.mu.Lock()
		_, inFlight := s.pushing[c.ID]
		s.mu.Unlock()
		if inFlight {
			s.log.Debug("sync scheduler: connection already in flight, skipping", "connection", c.ID)
			continue
		}
		select {
		case s.sem <- struct{}{}:
			s.mu.Lock()
			s.pushing[c.ID] = struct{}{}
			s.mu.Unlock()
			go s.push(ctx, c, interval)
		default:
			s.log.Debug("sync scheduler: concurrency cap reached, skipping", "connection", c.ID)
		}
	}
}

// due reports whether c is scheduled now: never attempted, or interval
// elapsed since the last attempt (success or failure). Scheduling off the
// attempt — not just the last success — gives a failing connection a cooldown
// between retries instead of re-firing on every tick.
func (s *Scheduler) due(c store.Connection, interval time.Duration, now time.Time) bool {
	if c.LastAttemptAt == "" {
		return true
	}
	last, err := time.Parse(time.RFC3339, c.LastAttemptAt)
	if err != nil {
		return true // unparseable timestamp — treat as never attempted
	}
	return !now.Before(last.Add(interval))
}

// push runs one scheduled push through the shared Service.Push path and
// reports the result. The connection's in-flight marker and the semaphore
// slot are released when it finishes.
func (s *Scheduler) push(ctx context.Context, c store.Connection, interval time.Duration) {
	defer func() {
		<-s.sem
		s.mu.Lock()
		delete(s.pushing, c.ID)
		s.mu.Unlock()
	}()
	err := s.svc.Push(ctx, c, s.root)
	result := Result{ConnectionID: c.ID, Name: c.Name, OK: err == nil}
	if err != nil {
		result.Error = err.Error()
	}
	if s.onPush != nil {
		s.onPush(result)
	}
	s.log.Info("sync scheduler: pushed", "connection", c.ID, "interval", interval.String(), "ok", result.OK)
}

// connectionInterval reads the connection's configured auto-sync interval
// (minutes) from its config. ok is false when the field is absent, unparseable,
// or zero — the connection is manual-only.
func connectionInterval(c store.Connection) (time.Duration, bool) {
	cfg, err := DecodeConfig(c.Config)
	if err != nil {
		return 0, false
	}
	raw := cfg["interval"]
	if raw == "" {
		return 0, false
	}
	n, err := strconv.Atoi(raw)
	if err != nil || n <= 0 {
		return 0, false
	}
	return time.Duration(n) * time.Minute, true
}
