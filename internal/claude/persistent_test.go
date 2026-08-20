package claude

import (
	"context"
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

// writePersistentFakeCLI writes a fake claude that appends each invocation's
// argv to argv.txt and, per stream-json user control line on stdin, replies
// with an assistant delta and a result line — the same contract a persistent
// process must satisfy. Variants below gate one-shot behaviors via marker
// files so the first spawn can misbehave while respawns behave.
func writePersistentFakeCLI(t *testing.T) string {
	t.Helper()
	return writeFakeCLIVariant(t, `#!/bin/sh
echo "$@" >> "$(dirname "$0")/argv.txt"
n=0
while IFS= read -r line; do
  case "$line" in
    *user*)
      n=$((n+1))
      echo '{"type":"assistant","message":{"content":[{"type":"text","text":"turn-'$n'"}]}}'
      echo '{"type":"result","subtype":"success","is_error":false,"result":"done"}'
      ;;
  esac
done
`)
}

func writeFakeCLIVariant(t *testing.T, script string) string {
	t.Helper()
	dir := t.TempDir()
	bin := filepath.Join(dir, "claude")
	if err := os.WriteFile(bin, []byte(script), 0o755); err != nil {
		t.Fatal(err)
	}
	return bin
}

// sleepOnceFake sleeps instead of answering on its FIRST spawn; respawns
// answer normally.
const sleepOnceFake = `#!/bin/sh
echo "$@" >> "$(dirname "$0")/argv.txt"
marker="$(dirname "$0")/sleep-done"
if [ ! -f "$marker" ]; then touch "$marker"; sleep 5; fi
n=0
while IFS= read -r line; do
  case "$line" in
    *user*)
      n=$((n+1))
      echo '{"type":"assistant","message":{"content":[{"type":"text","text":"turn-'$n'"}]}}'
      echo '{"type":"result","subtype":"success","is_error":false,"result":"done"}'
      ;;
  esac
done
`

// crashOnceFake exits 1 instead of answering on its FIRST spawn; respawns
// answer normally.
const crashOnceFake = `#!/bin/sh
echo "$@" >> "$(dirname "$0")/argv.txt"
marker="$(dirname "$0")/crashed"
if [ ! -f "$marker" ]; then touch "$marker"; exit 1; fi
n=0
while IFS= read -r line; do
  case "$line" in
    *user*)
      n=$((n+1))
      echo '{"type":"assistant","message":{"content":[{"type":"text","text":"turn-'$n'"}]}}'
      echo '{"type":"result","subtype":"success","is_error":false,"result":"done"}'
      ;;
  esac
done
`

const lockedFake = `#!/bin/sh
echo 'Session ID 1234 is already in use' >&2
exit 2
`

// slowOnceFake sleeps instead of answering on its FIRST spawn; respawns
// answer normally. Shorter than sleepOnceFake so mid-turn assertions finish
// quickly.
const slowOnceFake = `#!/bin/sh
marker="$(dirname "$0")/slow-done"
if [ ! -f "$marker" ]; then touch "$marker"; sleep 1; fi
n=0
while IFS= read -r line; do
  case "$line" in
    *user*)
      n=$((n+1))
      echo '{"type":"assistant","message":{"content":[{"type":"text","text":"turn-'$n'"}]}}'
      echo '{"type":"result","subtype":"success","is_error":false,"result":"done"}'
      ;;
  esac
done
`

// sessionIsolationFake routes each stdin line into a per-session log file
// keyed by the invocation's --session-id, proving a turn on one session never
// touches another session's process (each conversation owns its history).
const sessionIsolationFake = `#!/bin/sh
echo "$@" >> "$(dirname "$0")/argv.txt"
sid=""
prev=""
for a in "$@"; do
  if [ "$prev" = "--session-id" ]; then sid="$a"; fi
  prev="$a"
done
while IFS= read -r line; do
  echo "$line" >> "$(dirname "$0")/$sid.log"
  case "$line" in
    *user*)
      echo '{"type":"assistant","message":{"content":[{"type":"text","text":"ok"}]}}'
      echo '{"type":"result","subtype":"success","is_error":false,"result":"done"}'
      ;;
  esac
done
`

// realResultFake emits the result line with the real CLI's key order —
// is_error first, "type":"result" later in the object. Turn-end detection
// must not depend on JSON key order.
const realResultFake = `#!/bin/sh
n=0
while IFS= read -r line; do
  case "$line" in
    *user*)
      n=$((n+1))
      echo '{"type":"assistant","message":{"content":[{"type":"text","text":"turn-'$n'"}]}}'
      echo '{"is_error":false,"duration_api_ms":100,"num_turns":'$n',"subtype":"success","result":"done","type":"result"}'
      ;;
  esac
done
`

// initDrainFake emits a single ~200KB init line before answering, exercising
// the stdout pipe-buffer drain on spawn.
const initDrainFake = `#!/bin/sh
{ printf '{"type":"system","subtype":"init","data":"'; head -c 200000 /dev/zero | tr '\0' x; printf '"}\n'; }
n=0
while IFS= read -r line; do
  case "$line" in
    *user*)
      n=$((n+1))
      echo '{"type":"assistant","message":{"content":[{"type":"text","text":"turn-'$n'"}]}}'
      echo '{"type":"result","subtype":"success","is_error":false,"result":"done"}'
      ;;
  esac
done
`

func TestPersistentInjectsAPIKeyEnv(t *testing.T) {
	// The pooled spawn path must carry the same env as the one-shot Start:
	// the fake writes ANTHROPIC_API_KEY to a file and answers one turn.
	bin := writeFakeCLIVariant(t, `#!/bin/sh
printf '%s' "${ANTHROPIC_API_KEY-unset}" > "$(dirname "$0")/env.txt"
n=0
while IFS= read -r line; do
  case "$line" in
    *user*)
      n=$((n+1))
      echo '{"type":"assistant","message":{"content":[{"type":"text","text":"turn-'$n'"}]}}'
      echo '{"type":"result","subtype":"success","is_error":false,"result":"done"}'
      ;;
  esac
done
`)
	c := NewPersistent(bin, t.TempDir(), WithAPIKey("sk-pool-key"))
	c.IdleTimeout = time.Second
	t.Cleanup(func() { _ = c.Close() })

	if events := startTurn(t, c, "sess-env", "hi"); len(events) == 0 {
		t.Fatal("no events from turn")
	}
	raw, err := os.ReadFile(filepath.Join(filepath.Dir(bin), "env.txt"))
	if err != nil {
		t.Fatal(err)
	}
	if string(raw) != "sk-pool-key" {
		t.Fatalf("ANTHROPIC_API_KEY = %q, want sk-pool-key", raw)
	}
}

func argvOf(t *testing.T, bin string) string {
	t.Helper()
	raw, err := os.ReadFile(filepath.Join(filepath.Dir(bin), "argv.txt"))
	if err != nil {
		t.Fatal(err)
	}
	return string(raw)
}

func spawnCount(t *testing.T, bin string) int {
	return strings.Count(argvOf(t, bin), "--input-format")
}

// waitSpawnCount polls until the fake's argv file records at least want
// spawns. Warm spawns asynchronously (exec returns before the shell writes
// its argv line), so spawn assertions after a Warm need to wait.
func waitSpawnCount(t *testing.T, bin string, want int) {
	t.Helper()
	waitFor(t, 5*time.Second, func() bool {
		raw, err := os.ReadFile(filepath.Join(filepath.Dir(bin), "argv.txt"))
		return err == nil && strings.Count(string(raw), "--input-format") >= want
	})
}

// startTurn runs one turn collecting its events.
func startTurn(t *testing.T, c Client, sessionID, prompt string, opts ...StartOption) []Event {
	t.Helper()
	var got []Event
	err := c.Start(context.Background(), sessionID, prompt, WriterFunc(func(e Event) error {
		got = append(got, e)
		return nil
	}), opts...)
	if err != nil {
		t.Fatalf("Start: %v", err)
	}
	return got
}

// waitFor polls cond until it holds or the deadline passes.
func waitFor(t *testing.T, timeout time.Duration, cond func() bool) {
	t.Helper()
	deadline := time.Now().Add(timeout)
	for time.Now().Before(deadline) {
		if cond() {
			return
		}
		time.Sleep(10 * time.Millisecond)
	}
	t.Fatalf("condition not met within %v", timeout)
}

func TestPersistentServesTwoTurnsFromOneProcess(t *testing.T) {
	bin := writePersistentFakeCLI(t)
	pc := NewPersistent(bin, t.TempDir())
	t.Cleanup(func() { _ = pc.Close() })

	first := startTurn(t, pc, "sess-1", "question one")
	if len(first) == 0 || first[0].Text != "turn-1" {
		t.Fatalf("first turn deltas: %+v", first)
	}
	second := startTurn(t, pc, "sess-1", "question two")
	if len(second) == 0 || second[0].Text != "turn-2" {
		t.Fatalf("second turn deltas: %+v", second)
	}
	if n := spawnCount(t, bin); n != 1 {
		t.Fatalf("expected one process to serve both turns, got %d spawns", n)
	}
}

func TestPersistentSpawnsOneProcessPerSession(t *testing.T) {
	bin := writePersistentFakeCLI(t)
	pc := NewPersistent(bin, t.TempDir())
	t.Cleanup(func() { _ = pc.Close() })

	startTurn(t, pc, "sess-1", "one")
	startTurn(t, pc, "sess-2", "two")
	if n := spawnCount(t, bin); n != 2 {
		t.Fatalf("expected two processes for two sessions, got %d spawns", n)
	}
	argv := argvOf(t, bin)
	for _, want := range []string{"--session-id", "sess-1", "--session-id", "sess-2"} {
		if !strings.Contains(argv, want) {
			t.Fatalf("argv %q missing %q", argv, want)
		}
	}
}

func TestPersistentCancelKillsAndRespawns(t *testing.T) {
	bin := writeFakeCLIVariant(t, sleepOnceFake)
	pc := NewPersistent(bin, t.TempDir())
	t.Cleanup(func() { _ = pc.Close() })

	ctx, cancel := context.WithCancel(context.Background())
	done := make(chan error, 1)
	go func() {
		done <- pc.Start(ctx, "sess-1", "slow question", WriterFunc(func(Event) error { return nil }))
	}()
	time.Sleep(300 * time.Millisecond) // the turn is in flight
	cancel()
	select {
	case err := <-done:
		if !errors.Is(err, context.Canceled) {
			t.Fatalf("expected context.Canceled, got %v", err)
		}
	case <-time.After(10 * time.Second):
		t.Fatal("Start did not return after cancel")
	}
	waitFor(t, 10*time.Second, func() bool { return pc.poolSize() == 0 })

	// The next turn respawns (marker makes the fake answer this time). A
	// fresh pooled process is the deterministic respawn proof — the fake's
	// argv file can miss a killed spawn's line.
	events := startTurn(t, pc, "sess-1", "again")
	if len(events) == 0 || events[0].Text != "turn-1" {
		t.Fatalf("respawned turn deltas: %+v", events)
	}
	if pc.poolSize() != 1 {
		t.Fatalf("pool size after respawned turn = %d, want 1", pc.poolSize())
	}
}

// TestPersistentCancelRacesNextTurn hammers the cancel/claim overlap: a turn
// cancelled mid-flight must never strand the pool in a state where the next
// turn fails with a spurious busy error. Each iteration cancels a slow turn
// and immediately claims the session again.
func TestPersistentCancelRacesNextTurn(t *testing.T) {
	bin := writeFakeCLIVariant(t, sleepOnceFake)
	pc := NewPersistent(bin, t.TempDir())
	t.Cleanup(func() { _ = pc.Close() })

	for i := 0; i < 10; i++ {
		ctx, cancel := context.WithCancel(context.Background())
		done := make(chan error, 1)
		go func() {
			done <- pc.Start(ctx, "sess-1", "slow question", WriterFunc(func(Event) error { return nil }))
		}()
		time.Sleep(100 * time.Millisecond) // the turn is in flight
		cancel()
		// Claim the session again without waiting for the cancelled turn's
		// cleanup — the exact overlap that used to strand the pool.
		var got []Event
		err := pc.Start(context.Background(), "sess-1", "quick question", WriterFunc(func(e Event) error {
			got = append(got, e)
			return nil
		}))
		if err != nil {
			t.Fatalf("iteration %d: turn after cancel failed: %v", i, err)
		}
		select {
		case <-done:
		case <-time.After(10 * time.Second):
			t.Fatalf("iteration %d: cancelled Start did not return", i)
		}
		if len(got) == 0 {
			t.Fatalf("iteration %d: no deltas from the follow-up turn", i)
		}
	}
}

func TestPersistentIdleEviction(t *testing.T) {
	bin := writePersistentFakeCLI(t)
	pc := NewPersistent(bin, t.TempDir())
	pc.IdleTimeout = 50 * time.Millisecond
	t.Cleanup(func() { _ = pc.Close() })

	startTurn(t, pc, "sess-1", "one")
	if pc.poolSize() != 1 {
		t.Fatalf("pool size after turn = %d, want 1", pc.poolSize())
	}
	waitFor(t, 5*time.Second, func() bool { return pc.poolSize() == 0 })
}

func TestPersistentCrashMidTurn(t *testing.T) {
	bin := writeFakeCLIVariant(t, crashOnceFake)
	pc := NewPersistent(bin, t.TempDir())
	t.Cleanup(func() { _ = pc.Close() })

	err := pc.Start(context.Background(), "sess-1", "crash me", WriterFunc(func(Event) error { return nil }))
	if err == nil || !strings.Contains(err.Error(), "claude exited") {
		t.Fatalf("expected claude exited error, got %v", err)
	}
	waitFor(t, 5*time.Second, func() bool { return pc.poolSize() == 0 })

	events := startTurn(t, pc, "sess-1", "recover")
	if len(events) == 0 || events[0].Text != "turn-1" {
		t.Fatalf("recovered turn deltas: %+v", events)
	}
	if n := spawnCount(t, bin); n != 2 {
		t.Fatalf("expected a respawn after crash, got %d spawns", n)
	}
}

func TestPersistentSpawnLockedSession(t *testing.T) {
	bin := writeFakeCLIVariant(t, lockedFake)
	pc := NewPersistent(bin, t.TempDir())
	t.Cleanup(func() { _ = pc.Close() })

	err := pc.Start(context.Background(), "sess-1", "hello", WriterFunc(func(Event) error { return nil }))
	if err == nil || !strings.Contains(err.Error(), "already in use") {
		t.Fatalf("expected already-in-use error, got %v", err)
	}
}

func TestPersistentWarmSpawnsIdleProcess(t *testing.T) {
	bin := writePersistentFakeCLI(t)
	pc := NewPersistent(bin, t.TempDir())
	t.Cleanup(func() { _ = pc.Close() })

	if err := pc.Warm("sess-1"); err != nil {
		t.Fatalf("Warm: %v", err)
	}
	if pc.poolSize() != 1 {
		t.Fatalf("pool size after warm = %d, want 1", pc.poolSize())
	}
	waitSpawnCount(t, bin, 1)

	// A turn on the warm session produces zero new spawns: it rides the
	// process that boot already spawned.
	events := startTurn(t, pc, "sess-1", "question")
	if len(events) == 0 || events[0].Text != "turn-1" {
		t.Fatalf("warm-session turn deltas: %+v", events)
	}
	if n := spawnCount(t, bin); n != 1 {
		t.Fatalf("turn after warm must not respawn, got %d spawns", n)
	}
	if pc.poolSize() != 1 {
		t.Fatalf("pool size after turn = %d, want 1", pc.poolSize())
	}
}

func TestPersistentWarmIdempotent(t *testing.T) {
	bin := writePersistentFakeCLI(t)
	pc := NewPersistent(bin, t.TempDir())
	t.Cleanup(func() { _ = pc.Close() })

	if err := pc.Warm("sess-1"); err != nil {
		t.Fatalf("first Warm: %v", err)
	}
	if err := pc.Warm("sess-1"); err != nil {
		t.Fatalf("second Warm: %v", err)
	}
	if pc.poolSize() != 1 {
		t.Fatalf("pool size after two warms = %d, want 1", pc.poolSize())
	}
	waitSpawnCount(t, bin, 1)
	if n := spawnCount(t, bin); n != 1 {
		t.Fatalf("double warm must spawn once, got %d spawns", n)
	}
}

func TestPersistentWarmIdleEviction(t *testing.T) {
	bin := writePersistentFakeCLI(t)
	pc := NewPersistent(bin, t.TempDir())
	pc.IdleTimeout = 50 * time.Millisecond
	t.Cleanup(func() { _ = pc.Close() })

	if err := pc.Warm("sess-1"); err != nil {
		t.Fatalf("Warm: %v", err)
	}
	if pc.poolSize() != 1 {
		t.Fatalf("pool size after warm = %d, want 1", pc.poolSize())
	}
	// The idle-eviction timer a warm process arms must reap it, exactly like
	// a post-turn idle process.
	waitFor(t, 5*time.Second, func() bool { return pc.poolSize() == 0 })
}

func TestPersistentWarmErrors(t *testing.T) {
	pc := NewPersistent(writePersistentFakeCLI(t), t.TempDir())

	if err := pc.Warm(""); err == nil || !strings.Contains(err.Error(), "session id") {
		t.Fatalf("Warm(\"\"): want session id error, got %v", err)
	}
	startTurn(t, pc, "sess-1", "one")
	if err := pc.Close(); err != nil {
		t.Fatalf("Close: %v", err)
	}
	if err := pc.Warm("sess-1"); err == nil || !strings.Contains(err.Error(), "closed") {
		t.Fatalf("Warm after Close: want closed error, got %v", err)
	}
}

func TestPersistentWarmSpawnFailureLeavesPoolEmpty(t *testing.T) {
	pc := NewPersistent(filepath.Join(t.TempDir(), "missing-claude"), t.TempDir())
	t.Cleanup(func() { _ = pc.Close() })

	if err := pc.Warm("sess-1"); err == nil {
		t.Fatal("Warm with a missing binary must error")
	}
	if pc.poolSize() != 0 {
		t.Fatalf("pool size after failed warm = %d, want 0", pc.poolSize())
	}
	// A later turn spawns fresh and fails loudly rather than reusing a
	// half-spawned process.
	err := pc.Start(context.Background(), "sess-1", "hello", WriterFunc(func(Event) error { return nil }))
	if err == nil || !strings.Contains(err.Error(), "start claude") {
		t.Fatalf("Start after failed warm: want start error, got %v", err)
	}
	if pc.poolSize() != 0 {
		t.Fatalf("pool size after failed turn = %d, want 0", pc.poolSize())
	}
}

func TestPersistentMaxProcsEvictsLRUIdle(t *testing.T) {
	bin := writePersistentFakeCLI(t)
	pc := NewPersistent(bin, t.TempDir())
	pc.MaxProcs = 2
	t.Cleanup(func() { _ = pc.Close() })

	startTurn(t, pc, "sess-1", "one")
	startTurn(t, pc, "sess-2", "two")
	// A third session over the cap evicts the least-recently-used idle
	// process (sess-1) to make room.
	startTurn(t, pc, "sess-3", "three")
	if pc.poolSize() != 2 {
		t.Fatalf("pool size at cap = %d, want 2", pc.poolSize())
	}
	if n := spawnCount(t, bin); n != 3 {
		t.Fatalf("spawns with cap 2 after three sessions = %d, want 3", n)
	}
	// sess-1 was evicted: resuming it spawns fresh (and evicts the next LRU,
	// sess-2, keeping the pool at the cap).
	startTurn(t, pc, "sess-1", "again")
	if n := spawnCount(t, bin); n != 4 {
		t.Fatalf("resume of evicted session must respawn, got %d spawns", n)
	}
	if pc.poolSize() != 2 {
		t.Fatalf("pool size after evict + respawn = %d, want 2", pc.poolSize())
	}
}

func TestPersistentMaxProcsSkipsBusy(t *testing.T) {
	bin := writeFakeCLIVariant(t, sleepOnceFake)
	pc := NewPersistent(bin, t.TempDir())
	pc.MaxProcs = 1
	t.Cleanup(func() { _ = pc.Close() })

	ctx, cancel := context.WithCancel(context.Background())
	done := make(chan error, 1)
	go func() {
		done <- pc.Start(ctx, "sess-1", "slow", WriterFunc(func(Event) error { return nil }))
	}()
	time.Sleep(300 * time.Millisecond) // sess-1 is mid-turn (busy)

	// A second session over the cap must not kill the busy process: there is
	// no idle victim, so the cap is exceeded briefly instead.
	startTurn(t, pc, "sess-2", "quick")
	if pc.poolSize() != 2 {
		t.Fatalf("pool size with a busy process = %d, want 2 (cap exceeded, not killed)", pc.poolSize())
	}
	if n := spawnCount(t, bin); n != 2 {
		t.Fatalf("spawns = %d, want 2", n)
	}
	cancel()
	select {
	case <-done:
	case <-time.After(10 * time.Second):
		t.Fatal("slow turn did not return after cancel")
	}
	waitFor(t, 10*time.Second, func() bool { return pc.poolSize() == 1 })
}

func TestPersistentMaxProcsAppliesToWarm(t *testing.T) {
	bin := writePersistentFakeCLI(t)
	pc := NewPersistent(bin, t.TempDir())
	pc.MaxProcs = 2
	t.Cleanup(func() { _ = pc.Close() })

	// Waiting for each argv line after every warm makes the eviction of
	// sess-1 deterministic (its line is written before sess-3 evicts it).
	for i, sid := range []string{"sess-1", "sess-2", "sess-3"} {
		if err := pc.Warm(sid); err != nil {
			t.Fatalf("Warm(%s): %v", sid, err)
		}
		waitSpawnCount(t, bin, i+1)
	}
	if pc.poolSize() != 2 {
		t.Fatalf("pool size after warm over cap = %d, want 2", pc.poolSize())
	}
}

// TestPersistentWarmWhileBusyDoesNotEvict pins the busy guard on Warm's idle
// timer: warming an in-flight session must not schedule an eviction for it
// (only the turn's own end may arm the timer).
func TestPersistentWarmWhileBusyDoesNotEvict(t *testing.T) {
	bin := writeFakeCLIVariant(t, slowOnceFake)
	pc := NewPersistent(bin, t.TempDir())
	pc.IdleTimeout = 300 * time.Millisecond
	t.Cleanup(func() { _ = pc.Close() })

	done := make(chan error, 1)
	go func() {
		done <- pc.Start(context.Background(), "sess-1", "slow", WriterFunc(func(Event) error { return nil }))
	}()
	time.Sleep(300 * time.Millisecond) // mid-turn

	if err := pc.Warm("sess-1"); err != nil {
		t.Fatalf("Warm while busy: %v", err)
	}
	select {
	case err := <-done:
		if err != nil {
			t.Fatalf("turn: %v", err)
		}
	case <-time.After(5 * time.Second):
		t.Fatal("turn did not finish")
	}
	// The turn armed its own idle timer at its end; Warm must not have marked
	// the process for eviction, so it is still pooled right after the turn.
	if pc.poolSize() != 1 {
		t.Fatalf("pool size after warm-while-busy turn = %d, want 1", pc.poolSize())
	}
}

func TestPersistentSessionsAreIsolated(t *testing.T) {
	bin := writeFakeCLIVariant(t, sessionIsolationFake)
	pc := NewPersistent(bin, t.TempDir())
	t.Cleanup(func() { _ = pc.Close() })

	startTurn(t, pc, "sess-a", "prompt for A")
	startTurn(t, pc, "sess-b", "prompt for B")

	// Each conversation got its own process with its own --session-id.
	argv := argvOf(t, bin)
	for _, want := range []string{"--session-id sess-a", "--session-id sess-b"} {
		if !strings.Contains(argv, want) {
			t.Fatalf("argv %q missing %q", argv, want)
		}
	}
	readLog := func(sid string) string {
		t.Helper()
		raw, err := os.ReadFile(filepath.Join(filepath.Dir(bin), sid+".log"))
		if err != nil {
			t.Fatal(err)
		}
		return string(raw)
	}
	// A turn on one session must never reach another session's process: each
	// process only ever sees its own conversation's stdin.
	a, b := readLog("sess-a"), readLog("sess-b")
	if !strings.Contains(a, "prompt for A") || !strings.Contains(b, "prompt for B") {
		t.Fatalf("session logs not routed: A=%q B=%q", a, b)
	}
	if strings.Contains(a, "prompt for B") || strings.Contains(b, "prompt for A") {
		t.Fatalf("history leaked across processes: A=%q B=%q", a, b)
	}
}

func TestPersistentGetOrSpawnSkipsEvictingProc(t *testing.T) {
	bin := writePersistentFakeCLI(t)
	pc := NewPersistent(bin, t.TempDir())
	t.Cleanup(func() { _ = pc.Close() })

	if err := pc.Warm("sess-1"); err != nil {
		t.Fatalf("Warm: %v", err)
	}
	// Mark the pooled process evicting (as a concurrent cancel or cap
	// eviction would): the next claim must discard it and spawn fresh rather
	// than reuse a dying process.
	pc.mu.Lock()
	p := pc.procs["sess-1"]
	pc.mu.Unlock()
	p.mu.Lock()
	p.evicting = true
	p.mu.Unlock()

	events := startTurn(t, pc, "sess-1", "question")
	if len(events) == 0 {
		t.Fatal("no deltas from the fresh spawn")
	}
	waitSpawnCount(t, bin, 2) // warm spawn + respawn
	if pc.poolSize() != 1 {
		t.Fatalf("pool size = %d, want 1", pc.poolSize())
	}
}

func TestPersistentMaxProcsRestoresVictimOnSpawnFailure(t *testing.T) {
	bin := writePersistentFakeCLI(t)
	pc := NewPersistent(bin, t.TempDir())
	pc.MaxProcs = 1
	t.Cleanup(func() { _ = pc.Close() })

	if err := pc.Warm("sess-1"); err != nil {
		t.Fatalf("Warm: %v", err)
	}
	// Point the bin at something that cannot start, then spawn over the cap:
	// the selected victim must be restored (still pooled), not left evicting.
	pc.Bin = filepath.Join(t.TempDir(), "missing-claude")
	err := pc.Start(context.Background(), "sess-2", "hello", WriterFunc(func(Event) error { return nil }))
	if err == nil || !strings.Contains(err.Error(), "start claude") {
		t.Fatalf("Start with missing bin: want start error, got %v", err)
	}
	if pc.poolSize() != 1 {
		t.Fatalf("pool size after failed spawn = %d, want 1 (victim restored)", pc.poolSize())
	}
}

func TestPersistentEndsTurnOnRealCLIResultShape(t *testing.T) {
	bin := writeFakeCLIVariant(t, realResultFake)
	pc := NewPersistent(bin, t.TempDir())
	t.Cleanup(func() { _ = pc.Close() })

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	var got []Event
	err := pc.Start(ctx, "sess-1", "question", WriterFunc(func(e Event) error {
		got = append(got, e)
		return nil
	}))
	if err != nil {
		t.Fatalf("Start: %v", err)
	}
	if len(got) == 0 || got[0].Text != "turn-1" {
		t.Fatalf("turn deltas: %+v", got)
	}
	if pc.poolSize() != 1 {
		t.Fatalf("pool size after turn = %d, want 1 (process must stay pooled)", pc.poolSize())
	}
}

func TestPersistentInitDrainBeforeFirstWrite(t *testing.T) {
	bin := writeFakeCLIVariant(t, initDrainFake)
	pc := NewPersistent(bin, t.TempDir())
	t.Cleanup(func() { _ = pc.Close() })

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	var got []Event
	err := pc.Start(ctx, "sess-1", "question", WriterFunc(func(e Event) error {
		got = append(got, e)
		return nil
	}))
	if err != nil {
		t.Fatalf("Start: %v", err)
	}
	if len(got) == 0 || got[0].Text != "turn-1" {
		t.Fatalf("turn deltas: %+v", got)
	}
}

func TestPersistentArgs(t *testing.T) {
	bin := writePersistentFakeCLI(t)
	pc := NewPersistent(bin, t.TempDir())
	t.Cleanup(func() { _ = pc.Close() })

	startTurn(t, pc, "sess-1", "what is in my wiki?")
	argv := argvOf(t, bin)
	for _, want := range []string{"-p", "--input-format", "stream-json", "--output-format", "stream-json", "--verbose", "--autocompact", "auto", "--session-id", "sess-1", "--dangerously-skip-permissions"} {
		if !strings.Contains(argv, want) {
			t.Fatalf("argv %q missing %q", argv, want)
		}
	}
	if strings.Contains(argv, "what is in my wiki?") {
		t.Fatalf("argv %q must not carry the prompt (it travels over stdin)", argv)
	}
}

func TestPersistentWithResume(t *testing.T) {
	bin := writePersistentFakeCLI(t)
	pc := NewPersistent(bin, t.TempDir())
	t.Cleanup(func() { _ = pc.Close() })

	startTurn(t, pc, "new-id", "question", WithResume("old-id"))
	argv := argvOf(t, bin)
	for _, want := range []string{"--session-id", "new-id", "--resume", "old-id", "--fork-session"} {
		if !strings.Contains(argv, want) {
			t.Fatalf("argv %q missing %q", argv, want)
		}
	}
}

func TestPersistentDebugDump(t *testing.T) {
	bin := writePersistentFakeCLI(t)
	dumpPath := filepath.Join(t.TempDir(), "stream-dump.json")
	pc := NewPersistent(bin, t.TempDir(), WithDebugStream(dumpPath))
	t.Cleanup(func() { _ = pc.Close() })

	startTurn(t, pc, "sess-1", "one")
	startTurn(t, pc, "sess-1", "two")
	raw, err := os.ReadFile(dumpPath)
	if err != nil {
		t.Fatal(err)
	}
	s := string(raw)
	for _, want := range []string{"turn-1", "turn-2"} {
		if !strings.Contains(s, want) {
			t.Fatalf("dump missing %q:\n%s", want, s)
		}
	}
	for _, line := range strings.Split(strings.TrimSpace(s), "\n") {
		var m map[string]any
		if err := json.Unmarshal([]byte(line), &m); err != nil {
			t.Fatalf("dump line is not JSON: %q", line)
		}
	}
}

func TestPersistentConcurrentStartRejected(t *testing.T) {
	bin := writeFakeCLIVariant(t, sleepOnceFake)
	pc := NewPersistent(bin, t.TempDir())
	t.Cleanup(func() { _ = pc.Close() })

	ctx, cancel := context.WithCancel(context.Background())
	done := make(chan error, 1)
	go func() {
		done <- pc.Start(ctx, "sess-1", "first", WriterFunc(func(Event) error { return nil }))
	}()
	time.Sleep(300 * time.Millisecond) // first turn in flight

	err := pc.Start(context.Background(), "sess-1", "second", WriterFunc(func(Event) error { return nil }))
	if err == nil || !strings.Contains(err.Error(), "busy") {
		t.Fatalf("expected busy error for concurrent turn, got %v", err)
	}
	cancel()
	select {
	case <-done:
	case <-time.After(5 * time.Second):
		t.Fatal("first Start did not return after cancel")
	}
}

func TestPersistentFlushOnDirChange(t *testing.T) {
	bin := writePersistentFakeCLI(t)
	pc := NewPersistent(bin, t.TempDir())
	t.Cleanup(func() { _ = pc.Close() })

	startTurn(t, pc, "sess-1", "one")
	if pc.poolSize() != 1 {
		t.Fatalf("pool size after turn = %d, want 1", pc.poolSize())
	}
	pc.Flush()
	if pc.poolSize() != 0 {
		t.Fatalf("pool size after Flush = %d, want 0", pc.poolSize())
	}
	startTurn(t, pc, "sess-1", "two")
	if n := spawnCount(t, bin); n != 2 {
		t.Fatalf("expected a respawn after Flush, got %d spawns", n)
	}
}

func TestPersistentCloseAndEmptySessionID(t *testing.T) {
	bin := writePersistentFakeCLI(t)
	pc := NewPersistent(bin, t.TempDir())

	err := pc.Start(context.Background(), "", "prompt", WriterFunc(func(Event) error { return nil }))
	if err == nil || !strings.Contains(err.Error(), "session id") {
		t.Fatalf("expected session id error, got %v", err)
	}

	startTurn(t, pc, "sess-1", "one")
	if err := pc.Close(); err != nil {
		t.Fatalf("Close: %v", err)
	}
	if pc.poolSize() != 0 {
		t.Fatalf("pool size after Close = %d, want 0", pc.poolSize())
	}
	err = pc.Start(context.Background(), "sess-1", "after close", WriterFunc(func(Event) error { return nil }))
	if err == nil || !strings.Contains(err.Error(), "closed") {
		t.Fatalf("expected closed error, got %v", err)
	}
}
