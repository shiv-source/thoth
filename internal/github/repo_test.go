package github

import (
	"path/filepath"
	"testing"

	"github.com/shiv-source/thoth/internal/store"
)

// openTestRepo builds a temp db whose schema comes from the store's
// migrations, then opens the github repo on it.
func openTestRepo(t *testing.T) *Repo {
	t.Helper()
	path := filepath.Join(t.TempDir(), "test.db")
	st, err := store.Open(path)
	if err != nil {
		t.Fatal(err)
	}
	if err := st.Close(); err != nil {
		t.Fatal(err)
	}
	r, err := OpenRepo(path)
	if err != nil {
		t.Fatalf("OpenRepo: %v", err)
	}
	t.Cleanup(func() { _ = r.Close() })
	return r
}

func saved(t *testing.T, r *Repo, token, username string) Auth {
	t.Helper()
	a := Auth{
		Token: token, Username: username, DisplayName: "D", Email: "e@x",
		AvatarURL: "a", ProfileURL: "https://github.com/" + username,
		Scopes: "repo", AccountCreatedAt: "2018-05-01T10:00:00Z", AccountUpdatedAt: "2026-08-01T10:00:00Z",
	}
	if err := r.Save(a); err != nil {
		t.Fatalf("Save: %v", err)
	}
	return a
}

func TestRepoRoundTrip(t *testing.T) {
	r := openTestRepo(t)
	if a, ok, err := r.Get(); err != nil || ok || a != (Auth{}) {
		t.Fatalf("empty repo: %+v %v %v", a, ok, err)
	}

	in := saved(t, r, "ghp_one", "octo")
	got, ok, err := r.Get()
	if err != nil || !ok {
		t.Fatalf("Get: %v %v", err, ok)
	}
	if got.Token != in.Token || got.Username != in.Username || got.Email != in.Email || got.Scopes != in.Scopes ||
		got.ProfileURL != in.ProfileURL || got.AccountCreatedAt != in.AccountCreatedAt || got.AccountUpdatedAt != in.AccountUpdatedAt {
		t.Fatalf("round trip = %+v, want %+v", got, in)
	}

	// A reconnect (second Save) keeps created_at, bumps updated_at, and
	// replaces the identity fields.
	var firstCreated, firstUpdated string
	if err := r.db.QueryRow(`SELECT created_at, updated_at FROM github_auth`).Scan(&firstCreated, &firstUpdated); err != nil {
		t.Fatal(err)
	}
	// Force a distinctly old updated_at so the bump is deterministic (two
	// Saves within the same second would stamp identical RFC3339 strings).
	if _, err := r.db.Exec(`UPDATE github_auth SET updated_at = '2020-01-01T00:00:00Z'`); err != nil {
		t.Fatal(err)
	}
	saved(t, r, "ghp_two", "newuser")
	got, _, err = r.Get()
	if err != nil {
		t.Fatal(err)
	}
	if got.Token != "ghp_two" || got.Username != "newuser" {
		t.Fatalf("after reconnect = %+v, want new identity", got)
	}
	var secondCreated, secondUpdated string
	if err := r.db.QueryRow(`SELECT created_at, updated_at FROM github_auth`).Scan(&secondCreated, &secondUpdated); err != nil {
		t.Fatal(err)
	}
	if secondCreated != firstCreated {
		t.Fatalf("created_at changed on reconnect: %q -> %q", firstCreated, secondCreated)
	}
	if secondUpdated == "2020-01-01T00:00:00Z" {
		t.Fatal("updated_at must bump on reconnect")
	}
}

func TestRepoClear(t *testing.T) {
	r := openTestRepo(t)
	if err := r.Clear(); err != nil {
		t.Fatalf("Clear on empty repo must be a no-op: %v", err)
	}
	saved(t, r, "t", "octo")
	if err := r.Clear(); err != nil {
		t.Fatal(err)
	}
	if _, ok, err := r.Get(); err != nil || ok {
		t.Fatalf("after Clear: ok=%v err=%v, want not found", ok, err)
	}
}

func TestRepoSingleRowConstraint(t *testing.T) {
	r := openTestRepo(t)
	if _, err := r.db.Exec(
		`INSERT INTO github_auth (id, token, username, created_at, updated_at) VALUES (2, 't', 'u', 'x', 'x')`); err == nil {
		t.Fatal("second github_auth row must violate the id = 1 constraint")
	}
}

func TestRepoClosedErrors(t *testing.T) {
	r := openTestRepo(t)
	if err := r.Close(); err != nil {
		t.Fatal(err)
	}
	if err := r.Save(Auth{Token: "t", Username: "u"}); err == nil {
		t.Fatal("Save on closed repo must error")
	}
	if _, _, err := r.Get(); err == nil {
		t.Fatal("Get on closed repo must error")
	}
	if err := r.Clear(); err == nil {
		t.Fatal("Clear on closed repo must error")
	}
	if err := r.Clear(); err == nil {
		t.Fatal("Clear on closed repo must error")
	}
}
