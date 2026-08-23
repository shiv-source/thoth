// Package sync is the multi-provider sync engine: the wiki can be pushed to
// any connected destination — a git remote (GitHub, GitLab), an S3 bucket,
// or a local folder of timestamped zip snapshots. A catalog row
// (sync_providers) names a provider and its driver; a connection
// (sync_connections) carries the credentials + target for one destination.
// Drivers are small, stateless implementations of the Driver interface; the
// API layer and the agent-tool wiring depend on the interface, never on a
// specific provider.
package sync

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
)

// Kind is the transport family of a sync provider, used to group providers in
// the UI and to decide which connections the agent's git tools bind to
// (git-kind only).
type Kind string

const (
	KindGit   Kind = "git"   // push via agent/git to a git remote (github, gitlab)
	KindS3    Kind = "s3"    // push a wiki snapshot zip to object storage
	KindLocal Kind = "local" // write timestamped snapshot zips to a local folder
)

// Field describes one credential/target input the connect form renders,
// driven by the provider's driver. Secret fields are password inputs whose
// values never round-trip over the wire.
type Field struct {
	Key      string `json:"key"`    // "token", "access_key_id", "path", …
	Label    string `json:"label"`  // "Personal access token", …
	Secret   bool   `json:"secret"` // never echoed; empty = leave unchanged on PUT
	Required bool   `json:"required"`
}

// Identity is the token-free view of a connection's credentials, stored as
// JSON on the connection row and shown in the UI. Git providers fill the
// account fields; S3 fills Account (from sts:GetCallerIdentity); local has
// no identity.
type Identity struct {
	Username    string `json:"username,omitempty"`
	DisplayName string `json:"display_name,omitempty"`
	Email       string `json:"email,omitempty"`
	AvatarURL   string `json:"avatar_url,omitempty"`
	ProfileURL  string `json:"profile_url,omitempty"`
	Scopes      string `json:"scopes,omitempty"`
	Account     string `json:"account,omitempty"`
}

// Target is one selectable sync destination for a connected account (a repo
// for git providers). S3 and local have no listing — the user types the
// bucket/folder.
type Target struct {
	FullName    string `json:"full_name"`
	URL         string `json:"url"`
	Private     bool   `json:"private"`
	Description string `json:"description"`
}

// Config is a connection's decoded config JSON: credentials + target fields
// keyed by the driver's Field names.
type Config map[string]string

// DecodeConfig parses a connection's raw config JSON into a Config.
func DecodeConfig(raw string) (Config, error) {
	cfg := Config{}
	if raw == "" {
		return cfg, nil
	}
	if err := json.Unmarshal([]byte(raw), &cfg); err != nil {
		return nil, fmt.Errorf("parse connection config: %w", err)
	}
	return cfg, nil
}

// EncodeConfig serializes a Config for storage on the connection row.
func EncodeConfig(cfg Config) (string, error) {
	b, err := json.Marshal(cfg)
	if err != nil {
		return "", fmt.Errorf("encode connection config: %w", err)
	}
	return string(b), nil
}

// DecodeIdentity parses a connection's stored identity JSON.
func DecodeIdentity(raw string) (Identity, error) {
	var id Identity
	if raw == "" {
		return id, nil
	}
	if err := json.Unmarshal([]byte(raw), &id); err != nil {
		return id, fmt.Errorf("parse connection identity: %w", err)
	}
	return id, nil
}

// EncodeIdentity serializes a verified identity for storage on the
// connection row.
func EncodeIdentity(id Identity) (string, error) {
	b, err := json.Marshal(id)
	if err != nil {
		return "", fmt.Errorf("encode connection identity: %w", err)
	}
	return string(b), nil
}

// ErrRetryable marks a push failure that is safe to retry — a transient
// network flake or a server-side fault, as opposed to a permanent credential
// or config error. Drivers wrap their transient failures with it; the push
// path retries those with backoff and leaves permanent errors alone.
var ErrRetryable = errors.New("transient sync failure")

// retryable wraps a sanitized fixed message so errors.Is(err, ErrRetryable)
// holds while err.Error() stays the clean user-facing text (no sentinel leak).
func retryable(msg string) error { return retryableError{msg: msg} }

type retryableError struct{ msg string }

func (e retryableError) Error() string { return e.msg }
func (e retryableError) Is(target error) bool {
	return target == ErrRetryable
}

// isRetryable reports whether err is a transient sync failure worth retrying.
func isRetryable(err error) bool { return errors.Is(err, ErrRetryable) }

// Snapshot is one restorable archive for a connection: an S3 object key or a
// local snapshot filename, with the point-in-time it was written. It feeds
// the restore picker in the UI and the Restorer capability below.
type Snapshot struct {
	Key  string `json:"key"`
	Time string `json:"time"` // UTC RFC3339 of the snapshot, when known
}

// Restorer is the optional restore capability of a driver: it can hand back
// a stored wiki archive so the API can import it (the merge-with-backup
// flow). Git-kind drivers are push-only today and do not implement it — a
// pull from a git remote is a separate feature. Drivers assert the interface,
// the API layer type-asserts it.
type Restorer interface {
	// Snapshots lists the stored archives newest-first; "" when none. S3
	// lists the bucket prefix; local lists the backup folder.
	Snapshots(ctx context.Context, cfg Config) ([]Snapshot, error)
	// Restore returns the archive at key ("" = the latest) as a ReaderAt +
	// size pair ready for wiki.ImportFrom. Implementations must not echo
	// credentials in errors.
	Restore(ctx context.Context, cfg Config, key string) (io.ReaderAt, int64, error)
}

// Driver syncs the wiki to one provider kind. Implementations are stateless:
// every method takes the connection's stored config, so a driver never holds
// credentials beyond the call.
type Driver interface {
	Kind() Kind
	// Fields describes the config inputs the connect form renders.
	Fields() []Field
	// Verify checks credentials and returns the token-free identity to store.
	Verify(ctx context.Context, cfg Config) (Identity, error)
	// Targets lists selectable sync destinations for a connected account.
	// Empty for kinds without a listing.
	Targets(ctx context.Context, cfg Config) ([]Target, error)
	// Push syncs the wiki at root to the connection's target. committer is
	// the stored identity, used by git providers for the commit author.
	Push(ctx context.Context, cfg Config, root string, committer Identity) error
}

// IntervalField is the shared "auto-sync interval" descriptor every driver's
// Fields() includes, so the descriptor-driven connect form renders it without
// a code push per provider. 0 (or absent) = no scheduled pushes.
var IntervalField = Field{Key: "interval", Label: "Auto-sync interval (minutes, 0 = off)"}

// SnapshotTimeFormat is the UTC timestamp embedded in history snapshot keys:
// thoth-wiki-20060102-150405.zip. Sortable lexically, which is how S3's key
// ordering and a sorted dir listing agree on newest.
const SnapshotTimeFormat = "20060102-150405"
