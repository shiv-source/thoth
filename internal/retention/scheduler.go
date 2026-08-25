// Package retention periodically purges conversations older than the
// configured chat-history retention window, so the conversation tables cannot
// grow without bound. The window lives in the settings table and is re-read on
// every sweep, so a change in the Settings UI applies within one tick without
// a restart.
package retention

import (
	"context"
	"log/slog"
	"time"

	"github.com/shiv-source/thoth/internal/settings"
	"github.com/shiv-source/thoth/internal/store"
)

// defaultTick is the sweep cadence. Deleting a conversation up to one tick
// late is harmless, so an hour keeps the scan cost trivial.
const defaultTick = 1 * time.Hour

// Scheduler deletes conversations whose created_at is older than the retention
// window. A window of zero or fewer days disables auto-delete. It lives in
// serve, shares the store's delete path with the API, and stops on ctx
// cancellation so no goroutine outlives the server.
type Scheduler struct {
	st   *store.Store
	stg  *settings.Repo
	log  *slog.Logger
	tick time.Duration
	done chan struct{} // closed on Stop
}

// NewScheduler builds the retention scheduler. log defaults to a discard
// handler when nil.
func NewScheduler(st *store.Store, stg *settings.Repo, log *slog.Logger) *Scheduler {
	if log == nil {
		log = slog.New(slog.DiscardHandler)
	}
	return &Scheduler{st: st, stg: stg, log: log, tick: defaultTick, done: make(chan struct{})}
}

// Start runs the scheduler until ctx is cancelled (server shutdown) or Stop.
// One sweep runs immediately so stale conversations are purged on boot, then
// the scheduler ticks until told otherwise.
func (s *Scheduler) Start(ctx context.Context) {
	s.sweep(ctx)
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

// Stop halts the scheduler after the current sweep.
func (s *Scheduler) Stop() { close(s.done) }

// sweep reads the retention window and deletes every conversation older than
// it. A read or delete failure is logged and skipped — the next tick retries.
func (s *Scheduler) sweep(ctx context.Context) {
	if s.st == nil || s.stg == nil {
		return
	}
	days, err := s.stg.ConversationRetentionDays()
	if err != nil {
		s.log.Warn("retention: read setting", "err", err)
		return
	}
	if days <= 0 {
		return
	}
	cutoff := time.Now().UTC().Add(-time.Duration(days) * 24 * time.Hour)
	n, err := s.st.DeleteConversationsBefore(cutoff)
	if err != nil {
		s.log.Warn("retention: purge", "err", err)
		return
	}
	if n > 0 {
		s.log.Info("retention: purged old conversations", "days", days, "deleted", n)
	}
}
