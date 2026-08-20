package claude

import (
	"bufio"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"os/exec"
	"sync"
	"time"
)

// defaultIdleTimeout evicts a pooled process after 10 minutes without a
// turn, bounding memory to active conversations plus a short idle window.
const defaultIdleTimeout = 10 * time.Minute

// defaultMaxProcs caps the number of pooled processes. Each is a full Claude
// CLI (hundreds of MB RSS), so a hard ceiling keeps memory bounded when a
// user touches several conversations in quick succession.
const defaultMaxProcs = 4

// PersistentClient implements Client over a pool of long-lived CLI
// processes, one per conversation (session id). Spawning is lazy: the first
// turn of a conversation pays the CLI boot; later turns reuse the process
// via stream-json control messages on stdin. A cancelled turn kills its
// process (there is no per-turn interrupt in the plain CLI) and the next
// turn respawns, resuming the session from disk.
type PersistentClient struct {
	*CLIClient
	// IdleTimeout evicts an idle process (default 10 min; small values in
	// tests). Zero disables idle eviction.
	IdleTimeout time.Duration
	// MaxProcs caps how many processes the pool holds (default 4). When a
	// spawn would exceed it, the least-recently-used idle process is evicted
	// first; busy processes are never killed to make room, so the cap can be
	// exceeded briefly while several turns run at once. Zero disables the cap.
	MaxProcs int

	mu        sync.Mutex
	procs     map[string]*proc
	dump      *os.File // shared debug stream, opened at first spawn
	dumpMu    sync.Mutex
	dumpBytes int
	closed    bool
}

// proc is one long-lived CLI process plus its dispatcher goroutine. The
// hub guarantees at most one in-flight turn per session, so busy is a
// boolean, not a counter.
type proc struct {
	mu        sync.Mutex // guards busy, evict, evicting, seq, w, stdin, idleTimer, lastUsed
	cmd       *exec.Cmd
	stdout    io.ReadCloser // read end; closing it unblocks the dispatcher
	stdin     *bufio.Writer
	busy      bool
	evict     bool // eviction requested while busy → applied at turn end
	evicting  bool // eviction in progress: never handed to a new turn
	seq       int  // turn counter: dispatcher detects turn boundaries by it
	w         EventWriter
	lastUsed  time.Time // recency for cap eviction (LRU); zero = freshly spawned
	idleTimer *time.Timer
	tail      *stderrTail
	turnDone  chan error // dispatcher → in-flight Start (buffered 1)
}

// errBusy marks a session whose pooled process is mid-turn. The hub never
// overlaps same-session turns, so Start treats it as transient (a cancelled
// turn's cleanup is still in flight) and retries briefly.
var errBusy = errors.New("claude: session busy")

// poolEntry pairs a session id with its process for lock-free batch work.
type poolEntry struct {
	sid string
	p   *proc
}

// NewPersistent builds a pooled client. Same options as New — the embedded
// CLIClient supplies the bin, cwd, permission mode, model, and debug stream.
func NewPersistent(bin, dir string, opts ...Option) *PersistentClient {
	return &PersistentClient{
		CLIClient:   New(bin, dir, opts...),
		IdleTimeout: defaultIdleTimeout,
		MaxProcs:    defaultMaxProcs,
		procs:       map[string]*proc{},
	}
}

// poolSize reports how many processes the pool holds (test aid).
func (c *PersistentClient) poolSize() int {
	c.mu.Lock()
	defer c.mu.Unlock()
	return len(c.procs)
}

// Start implements Client: the prompt travels over the process's stdin as a
// stream-json control message and the turn ends at the CLI's result line.
// A busy session is transient (a cancelled turn's cleanup is in flight), so
// Start retries briefly before reporting it.
func (c *PersistentClient) Start(ctx context.Context, sessionID, prompt string, w EventWriter, opts ...StartOption) error {
	var cfg startConfig
	for _, opt := range opts {
		opt(&cfg)
	}
	var busyErr error
	for attempt := 0; attempt < 5; attempt++ {
		if attempt > 0 {
			select {
			case <-ctx.Done():
				return ctx.Err()
			case <-time.After(10 * time.Millisecond):
			}
		}
		err := c.start(ctx, sessionID, prompt, w, &cfg)
		if !errors.Is(err, errBusy) {
			return err
		}
		busyErr = err
	}
	return busyErr
}

// stdinErrOrExit resolves a stdin write/flush failure: if the CLI already
// exited, its exit error carries the stderr tail (e.g. "already in use") and
// is the real failure — the broken pipe is just the symptom. Wait briefly for
// the dispatcher's verdict before falling back to the pipe error.
func (c *PersistentClient) stdinErrOrExit(p *proc, pipeErr error) error {
	select {
	case turnErr := <-p.turnDone:
		return turnErr
	case <-time.After(2 * time.Second):
		return fmt.Errorf("claude: stdin: %w", pipeErr)
	}
}

// start runs one turn on the pooled process for sessionID.
func (c *PersistentClient) start(ctx context.Context, sessionID, prompt string, w EventWriter, cfg *startConfig) error {
	p, err := c.getOrSpawn(sessionID, cfg)
	if err != nil {
		return err
	}

	p.mu.Lock()
	if p.busy || p.evicting {
		// A busy session is transient (a cancelled turn's cleanup is still in
		// flight). An evicting process is mid-kill — a cancel or the cap
		// eviction selected it between getOrSpawn's check and this claim — so
		// it must never be handed to a turn. Both fail fast so Start retries
		// and spawns fresh.
		p.mu.Unlock()
		return fmt.Errorf("%w %s", errBusy, sessionID)
	}
	p.busy = true
	p.lastUsed = time.Now()
	p.seq++
	p.w = w
	if p.idleTimer != nil {
		p.idleTimer.Stop()
		p.idleTimer = nil
	}
	// The control message is marshaled, never hand-built: prompts carry
	// newlines, quotes, and emoji.
	line, err := json.Marshal(map[string]any{
		"type": "user", "message": map[string]string{"role": "user", "content": prompt},
	})
	if err != nil {
		p.busy = false
		p.mu.Unlock()
		return fmt.Errorf("claude: marshal prompt: %w", err)
	}
	if _, err := p.stdin.Write(append(line, '\n')); err != nil {
		p.mu.Unlock()
		c.evict(sessionID, p, true)
		return c.stdinErrOrExit(p, err)
	}
	if err := p.stdin.Flush(); err != nil {
		p.mu.Unlock()
		c.evict(sessionID, p, true)
		return c.stdinErrOrExit(p, err)
	}
	p.mu.Unlock()

	select {
	case <-ctx.Done():
		// No per-turn interrupt exists in the plain CLI: cancel kills the
		// process and the next turn respawns. The cancelled turn still owns
		// the process (busy stayed set), so it clears busy and marks the
		// process evicting before the kill — later claims spawn fresh.
		p.mu.Lock()
		p.busy = false
		p.evicting = true
		p.mu.Unlock()
		c.evict(sessionID, p, true)
		return ctx.Err()
	case err := <-p.turnDone:
		if err != nil {
			c.evict(sessionID, p, true)
			return err
		}
		p.mu.Lock()
		p.busy = false
		p.w = nil
		evict := p.evict
		p.evict = false
		if !evict && c.IdleTimeout > 0 {
			p.idleTimer = time.AfterFunc(c.IdleTimeout, func() { c.evict(sessionID, p, false) })
		}
		p.mu.Unlock()
		if evict {
			c.evict(sessionID, p, false)
		}
		return nil
	}
}

// getOrSpawn returns the process for sessionID, spawning it on first use.
// A spawn at the pool's MaxProcs cap evicts the least-recently-used idle
// process first (busy processes are never killed to make room).
func (c *PersistentClient) getOrSpawn(sessionID string, cfg *startConfig) (*proc, error) {
	c.mu.Lock()
	if c.closed {
		c.mu.Unlock()
		return nil, errors.New("claude: client closed")
	}
	if sessionID == "" {
		c.mu.Unlock()
		return nil, errors.New("claude: session id required")
	}
	if p, ok := c.procs[sessionID]; ok {
		p.mu.Lock()
		evicting := p.evicting
		p.mu.Unlock()
		if !evicting {
			c.mu.Unlock()
			return p, nil
		}
		delete(c.procs, sessionID) // dying: spawn fresh below
	}
	// Make room under the cap before spawning. The victim is marked evicting
	// so no concurrent claim can grab it between selection and the kill; a
	// failed spawn restores it (it was never killed).
	var victimSid string
	var victim *proc
	if c.MaxProcs > 0 && len(c.procs) >= c.MaxProcs {
		victimSid, victim = c.lruIdleLocked()
	}
	p, err := c.spawnLocked(sessionID, cfg)
	if err != nil {
		if victim != nil {
			victim.mu.Lock()
			victim.evicting = false
			victim.mu.Unlock()
		}
		c.mu.Unlock()
		return nil, err
	}
	c.procs[sessionID] = p
	c.mu.Unlock()

	if victim != nil {
		// Kill the evicted process after releasing c.mu: evict → unregister
		// re-acquires it, so holding the lock here would deadlock.
		c.evict(victimSid, victim, true)
	}
	return p, nil
}

// lruIdleLocked selects the least-recently-used idle process to evict when
// the pool is at its cap, marking it evicting so a concurrent claim cannot
// grab it. Caller holds c.mu; returns ("", nil) when every process is busy or
// already dying — the cap is then exceeded briefly until a turn ends.
func (c *PersistentClient) lruIdleLocked() (string, *proc) {
	var victimSid string
	var victim *proc
	var oldest time.Time
	for sid, p := range c.procs {
		p.mu.Lock()
		idle := !p.busy && !p.evicting
		last := p.lastUsed
		p.mu.Unlock()
		if !idle {
			continue
		}
		// The zero timestamp (a spawn that has not yet claimed a turn) counts
		// as the oldest candidate — it has provably never been used.
		if victim == nil ||
			(last.IsZero() && !oldest.IsZero()) ||
			(!last.IsZero() && !oldest.IsZero() && last.Before(oldest)) {
			victim, victimSid, oldest = p, sid, last
		}
	}
	if victim != nil {
		victim.mu.Lock()
		victim.evicting = true
		victim.mu.Unlock()
	}
	return victimSid, victim
}

// Warm eagerly spawns the pooled process for sessionID and leaves it idle, so
// the first turn on that session skips the CLI boot. It goes through the exact
// getOrSpawn → spawnLocked path a turn takes (same BLAST WALL args, cwd, and
// debug stream) with no prompt, and arms the same idle-eviction timer a
// finished turn would set, so an unused warm process is reaped after
// IdleTimeout. An error — closed client, empty session id, or a failed spawn —
// leaves the pool unchanged; the next Start on that session spawns normally.
func (c *PersistentClient) Warm(sessionID string) error {
	p, err := c.getOrSpawn(sessionID, &startConfig{})
	if err != nil {
		return err
	}
	// Only a turn's end arms the idle timer; a warm process has run no turn,
	// so arm it here or an unused process would linger until Close. A busy
	// process (a turn just claimed it) arms its own timer at turn end, so
	// arming one now would wrongly schedule an eviction for an active process.
	p.mu.Lock()
	p.lastUsed = time.Now()
	if p.idleTimer == nil && !p.busy && c.IdleTimeout > 0 {
		p.idleTimer = time.AfterFunc(c.IdleTimeout, func() { c.evict(sessionID, p, false) })
	}
	p.mu.Unlock()
	return nil
}

// spawnLocked starts the CLI and its dispatcher. Caller holds c.mu. The
// dispatcher starts before any stdin write so the stdout pipe buffer can
// never fill, no matter how much init output the CLI emits.
func (c *PersistentClient) spawnLocked(sessionID string, cfg *startConfig) (*proc, error) {
	cmd := exec.Command(c.Bin, c.persistentArgs(sessionID, cfg)...)
	cmd.Dir = c.dir()
	cmd.Env = c.env()
	setProcessGroup(cmd)

	stdout, err := cmd.StdoutPipe()
	if err != nil {
		return nil, fmt.Errorf("claude stdout pipe: %w", err)
	}
	stdin, err := cmd.StdinPipe()
	if err != nil {
		return nil, fmt.Errorf("claude stdin pipe: %w", err)
	}
	p := &proc{
		cmd:      cmd,
		stdout:   stdout,
		stdin:    bufio.NewWriterSize(stdin, 64<<10),
		tail:     &stderrTail{max: 4 << 10},
		turnDone: make(chan error, 1),
	}
	cmd.Stderr = p.tail
	if err := cmd.Start(); err != nil {
		_ = stdin.Close()
		return nil, fmt.Errorf("start claude: %w", err)
	}

	if c.debugPath != "" && c.dump == nil {
		if f, err := openDebugDump(c.debugPath); err == nil {
			c.dump = f
		}
	}
	go c.dispatch(sessionID, p)
	return p, nil
}

// dispatch reads the process stdout for its whole lifetime: lines for the
// in-flight turn go to its writer, the result line ends the turn, and EOF
// (crash, kill, or CLI exit) fails the in-flight turn and removes the
// process from the pool.
func (c *PersistentClient) dispatch(sessionID string, p *proc) {
	sc := bufio.NewScanner(p.stdout)
	sc.Buffer(make([]byte, 0, 1<<20), 1<<20)

	var streamErr error
	lastSeq := -1
	turnOver := false // drop stray output after the result line
	for sc.Scan() {
		line := sc.Bytes()
		c.dumpLine(line)
		p.mu.Lock()
		busy, w, seq := p.busy, p.w, p.seq
		p.mu.Unlock()
		if !busy {
			lastSeq = -1
			turnOver = false // idle output (init/system chatter): drop
			continue
		}
		if seq != lastSeq {
			lastSeq = seq
			turnOver = false // a new turn began: fresh stream state
		}
		if turnOver {
			continue
		}
		ev, err := ParseLine(line)
		if errors.Is(err, ErrIgnore) {
			continue
		}
		if err != nil {
			_ = w.Write(Event{Type: EventError, Detail: err.Error()})
			continue
		}
		// The parsed event decides turn end: EventDone and EventError come
		// only from the CLI's result line, and JSON parsing is key-order
		// independent — the real CLI emits is_error first, not type.
		endTurn := ev.Type == EventDone || ev.Type == EventError
		if err := w.Write(ev); err != nil {
			streamErr = err
			break
		}
		if endTurn {
			turnOver = true
			p.turnDone <- nil
		}
	}
	if streamErr == nil {
		streamErr = sc.Err()
	}

	waitErr := p.cmd.Wait()
	p.mu.Lock()
	busy := p.busy
	p.mu.Unlock()
	if busy {
		p.turnDone <- turnFailure(waitErr, streamErr, p.tail)
	}
	c.unregister(sessionID, p)
}

// turnFailure builds the turn error from the process exit and stream state,
// preserving the stderr tail so callers (the hub's "already in use" retry)
// can inspect it.
func turnFailure(waitErr, streamErr error, tail *stderrTail) error {
	if waitErr != nil {
		if s := tail.String(); s != "" {
			return fmt.Errorf("claude exited: %w (stderr: %s)", waitErr, s)
		}
		return fmt.Errorf("claude exited: %w", waitErr)
	}
	if streamErr != nil {
		return fmt.Errorf("claude stream: %w", streamErr)
	}
	return errors.New("claude exited before the turn finished")
}

// evict kills the process and removes it from the pool. A busy process is
// only marked (force=false) and evicted at its turn's end; force=true is
// for paths that already own the turn's termination (cancel, crash, close).
func (c *PersistentClient) evict(sessionID string, p *proc, force bool) {
	p.mu.Lock()
	if p.busy && !force {
		p.evict = true
		p.mu.Unlock()
		return
	}
	p.evicting = true
	p.mu.Unlock()

	if p.cmd.Process != nil {
		_ = killProcess(p.cmd.Process.Pid, p.cmd.Process)
	}
	// Closing the read end unblocks the dispatcher's Scan immediately on
	// every platform — on windows a grandchild can hold the pipe open, and
	// waiting for EOF would hang.
	_ = p.stdout.Close()
	c.unregister(sessionID, p)
}

// unregister removes p from the pool when it is still the registered
// process for sessionID.
func (c *PersistentClient) unregister(sessionID string, p *proc) {
	c.mu.Lock()
	defer c.mu.Unlock()
	if c.procs[sessionID] == p {
		delete(c.procs, sessionID)
	}
}

// Flush drops every pooled process: idle ones die now, a busy one finishes
// its turn and is evicted. Called when the wiki path changes (so the next
// spawn picks up the new cwd) and when the user leaves the chat page for a
// while (so idle CLI processes do not linger until their own timers).
func (c *PersistentClient) Flush() {
	c.mu.Lock()
	entries := make([]poolEntry, 0, len(c.procs))
	for sid, p := range c.procs {
		entries = append(entries, poolEntry{sid, p})
	}
	c.mu.Unlock()
	for _, e := range entries {
		c.evict(e.sid, e.p, false)
	}
}

// Close shuts the pool down: every process dies and later Starts error.
// The server's shutdown context cancels in-flight turns first; Close
// guarantees no CLI process outlives the server.
func (c *PersistentClient) Close() error {
	c.mu.Lock()
	if c.closed {
		c.mu.Unlock()
		return nil
	}
	c.closed = true
	entries := make([]poolEntry, 0, len(c.procs))
	for sid, p := range c.procs {
		entries = append(entries, poolEntry{sid, p})
	}
	dump := c.dump
	c.dump = nil
	c.mu.Unlock()

	for _, e := range entries {
		c.evict(e.sid, e.p, true)
	}
	if dump != nil {
		return dump.Close()
	}
	return nil
}

// dumpLine appends one raw stream line to the shared debug dump, rotating
// the file in place past debugDumpMaxBytes.
func (c *PersistentClient) dumpLine(line []byte) {
	c.dumpMu.Lock()
	defer c.dumpMu.Unlock()
	if c.dump == nil {
		return
	}
	_, _ = c.dump.Write(line)
	_, _ = c.dump.Write([]byte("\n"))
	c.dumpBytes += len(line) + 1
	if c.dumpBytes > debugDumpMaxBytes {
		_ = c.dump.Close()
		if f, err := openDebugDump(c.debugPath); err == nil {
			c.dump = f
		} else {
			c.dump = nil
		}
		c.dumpBytes = 0
	}
}
