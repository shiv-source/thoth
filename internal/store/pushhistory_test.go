package store

import (
	"testing"
)

// TestPushHistoryCapped verifies the per-connection history is bounded: after
// maxPushHistory appends, the oldest entries are pruned and only the newest
// remain.
func TestPushHistoryCapped(t *testing.T) {
	s := openStore(t)
	p, err := s.SyncProviderBySlug("github")
	if err != nil {
		t.Fatal(err)
	}
	c, err := s.CreateConnection(p.ID, "work", `{"token":"t"}`, `{"username":"octo"}`, true)
	if err != nil {
		t.Fatalf("CreateConnection: %v", err)
	}
	for i := 0; i < maxPushHistory+5; i++ {
		if err := s.SetConnectionSyncResult(c.ID, i%2 == 0, "detail"); err != nil {
			t.Fatalf("SetConnectionSyncResult %d: %v", i, err)
		}
	}
	history, err := s.ListPushHistory(c.ID)
	if err != nil {
		t.Fatal(err)
	}
	if len(history) != maxPushHistory {
		t.Fatalf("history length = %d, want capped at %d", len(history), maxPushHistory)
	}
	// Newest first: the newest entry is an even (success) index.
	if !history[0].OK {
		t.Fatalf("newest history entry not first: %+v", history[0])
	}
}
