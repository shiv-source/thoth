package sync

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	awsconfig "github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/credentials"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/aws/aws-sdk-go-v2/service/s3/types"
	"github.com/aws/aws-sdk-go-v2/service/sts"
	"github.com/shiv-source/thoth/internal/wiki"
)

// Snapshot modes for the s3 driver. The mode is a plain config field
// (descriptor-driven form), so no code push is needed to change the scheme.
const (
	// SnapshotStable keeps the backwards-compatible stable key wiki.zip
	// (overwritten each push). The default, so existing connections and any
	// tooling pointed at wiki.zip keep working.
	SnapshotStable = "stable"
	// SnapshotHistory writes only timestamped thoth-wiki-YYYYMMDD-HHMMSS.zip
	// keys, like the local driver.
	SnapshotHistory = "history"
	// SnapshotBoth keeps the wiki.zip pointer (so stable tooling keeps
	// working) AND a timestamped history.
	SnapshotBoth = "both"
)

// snapshotPrefix is the timestamped snapshot key base; full keys look like
// <prefix>/thoth-wiki-20060102-150405.zip.
const snapshotPrefix = "thoth-wiki-"

// s3Driver pushes a wiki snapshot zip to an S3 bucket (or any S3-compatible
// endpoint via the provider's base_url). Verify proves the keys with
// sts:GetCallerIdentity on AWS (falling back to a bucket probe when sts is
// denied) or a bucket probe on custom endpoints that have no STS. Push honors
// the snapshot mode + retention config; the driver also restores (list +
// download) a stored archive for the API's import flow.
type s3Driver struct {
	endpoint string // provider base_url override; "" = real AWS
}

func (d *s3Driver) Kind() Kind { return KindS3 }

func (d *s3Driver) Fields() []Field {
	return []Field{
		{Key: "access_key_id", Label: "Access key ID", Secret: true, Required: true},
		{Key: "secret_access_key", Label: "Secret access key", Secret: true, Required: true},
		{Key: "region", Label: "Region", Required: true},
		{Key: "bucket", Label: "Bucket", Required: true},
		{Key: "prefix", Label: "Key prefix (optional)"},
		{Key: "snapshot", Label: "Snapshot mode (stable | history | both)"},
		{Key: "retention", Label: "Keep last N snapshots (0 = keep all)"},
		IntervalField,
	}
}

func (d *s3Driver) awsConfig(ctx context.Context, cfg Config) (aws.Config, error) {
	region := cfg["region"]
	if region == "" {
		region = "us-east-1"
	}
	return awsconfig.LoadDefaultConfig(ctx,
		awsconfig.WithRegion(region),
		awsconfig.WithCredentialsProvider(credentials.NewStaticCredentialsProvider(
			cfg["access_key_id"], cfg["secret_access_key"], "")),
	)
}

func (d *s3Driver) newS3(cfg aws.Config) *s3.Client {
	// Retry is the sync layer's job (pushWithRetry distinguishes transient
	// flakes from permanent errors): disabling the SDK's built-in retries
	// keeps the classification and backoff in one place.
	if d.endpoint != "" {
		return s3.NewFromConfig(cfg, func(o *s3.Options) {
			o.BaseEndpoint = aws.String(d.endpoint)
			o.RetryMaxAttempts = 1
		})
	}
	return s3.NewFromConfig(cfg, func(o *s3.Options) { o.RetryMaxAttempts = 1 })
}

func (d *s3Driver) Verify(ctx context.Context, cfg Config) (Identity, error) {
	if cfg["access_key_id"] == "" || cfg["secret_access_key"] == "" {
		return Identity{}, errors.New("access key ID and secret access key are required")
	}
	if cfg["bucket"] == "" {
		return Identity{}, errors.New("bucket is required")
	}
	acfg, err := d.awsConfig(ctx, cfg)
	if err != nil {
		return Identity{}, fmt.Errorf("configure aws: %w", err)
	}
	if d.endpoint != "" {
		// S3-compatible endpoint (MinIO/R2): no STS — prove the keys by
		// reaching the target bucket.
		if _, err := d.newS3(acfg).HeadBucket(ctx, &s3.HeadBucketInput{Bucket: aws.String(cfg["bucket"])}); err != nil {
			return Identity{}, ErrRejected
		}
		return Identity{Account: cfg["bucket"]}, nil
	}
	// Real AWS: sts:GetCallerIdentity names the account. An AccessDenied
	// (keys valid but no sts permission) degrades to the bucket probe.
	if out, err := sts.NewFromConfig(acfg).GetCallerIdentity(ctx, &sts.GetCallerIdentityInput{}); err == nil {
		return Identity{Account: aws.ToString(out.Account)}, nil
	}
	if _, err := d.newS3(acfg).HeadBucket(ctx, &s3.HeadBucketInput{Bucket: aws.String(cfg["bucket"])}); err == nil {
		return Identity{}, nil
	}
	return Identity{}, ErrRejected
}

func (d *s3Driver) Targets(context.Context, Config) ([]Target, error) { return nil, nil }

// baseKey joins the connection's prefix and name into a key: "<prefix>/name"
// with the prefix trimmed of slashes and "" when there is no prefix.
func (d *s3Driver) baseKey(cfg Config, name string) string {
	key := strings.Trim(cfg["prefix"], "/")
	if key != "" {
		key += "/"
	}
	return key + name
}

// snapshotKey returns the timestamped history key for a time.
func snapshotKey(ts time.Time) string {
	return snapshotPrefix + ts.UTC().Format(SnapshotTimeFormat) + ".zip"
}

// isHistoryKey reports whether a base key is a timestamped snapshot (vs the
// stable wiki.zip pointer).
func isHistoryKey(key string) bool {
	return strings.HasPrefix(key, snapshotPrefix) && strings.HasSuffix(key, ".zip")
}

// snapshotMode returns the connection's configured mode, defaulting to stable.
func snapshotMode(cfg Config) string {
	mode := cfg["snapshot"]
	if mode == "" {
		mode = SnapshotStable
	}
	return mode
}

// retention parses the retention config; 0 (or unset/invalid) keeps all.
func retention(cfg Config) int {
	n, err := strconv.Atoi(cfg["retention"])
	if err != nil || n < 0 {
		return 0
	}
	return n
}

// httpStatusCoder is implemented by the AWS SDK's response errors
// (*smithyhttp.ResponseError via the wrapped chain), exposing the HTTP status
// code that classifies the failure.
type httpStatusCoder interface{ HTTPStatusCode() int }

// isRetryableS3 classifies a PutObject error: a server-side fault (5xx) or a
// transport-level failure (no HTTP response — DNS/connect/timeout) is a
// transient flake worth retrying; a client fault (4xx — bad credentials,
// missing bucket) is permanent.
func isRetryableS3(err error) bool {
	var status httpStatusCoder
	if errors.As(err, &status) {
		return status.HTTPStatusCode() >= 500
	}
	return true // no HTTP response: transport / connection error
}

// Push zips the wiki and uploads it to the bucket under the connection's
// snapshot mode: stable wiki.zip (default, backwards compat), timestamped
// history keys only, or both. An optional retention prunes the oldest history
// snapshots after writing. Errors are sanitized fixed messages; the raw SDK
// error may echo bucket/key names. Transient failures wrap ErrRetryable so the
// push path can retry network flakes.
func (d *s3Driver) Push(ctx context.Context, cfg Config, root string, _ Identity) error {
	if cfg["bucket"] == "" {
		return errors.New("no bucket configured — set one in Settings")
	}
	if cfg["access_key_id"] == "" || cfg["secret_access_key"] == "" {
		return errors.New("no credentials stored — reconnect in Settings")
	}
	acfg, err := d.awsConfig(ctx, cfg)
	if err != nil {
		return fmt.Errorf("configure aws: %w", err)
	}
	var buf bytes.Buffer
	if err := wiki.New(root).ExportTo(&buf, wiki.ExportOptions{}); err != nil {
		return errors.New("could not create the wiki archive")
	}
	mode := snapshotMode(cfg)
	write := func(key string) error {
		if _, err := d.newS3(acfg).PutObject(ctx, &s3.PutObjectInput{
			Bucket:      aws.String(cfg["bucket"]),
			Key:         aws.String(key),
			Body:        bytes.NewReader(buf.Bytes()),
			ContentType: aws.String("application/zip"),
		}); err != nil {
			if isRetryableS3(err) {
				return retryable("could not upload the wiki to the bucket")
			}
			return errors.New("could not upload the wiki to the bucket — check your credentials and network access")
		}
		return nil
	}
	historyKey := d.baseKey(cfg, snapshotKey(time.Now()))
	switch mode {
	case SnapshotHistory:
		if err := write(historyKey); err != nil {
			return err
		}
	case SnapshotBoth:
		if err := write(d.baseKey(cfg, "wiki.zip")); err != nil {
			return err
		}
		if err := write(historyKey); err != nil {
			return err
		}
	default: // stable — the default and any unknown value keep wiki.zip
		if err := write(d.baseKey(cfg, "wiki.zip")); err != nil {
			return err
		}
	}
	if keep := retention(cfg); keep > 0 && mode != SnapshotStable {
		if err := d.pruneHistory(ctx, acfg, cfg, keep); err != nil {
			return err
		}
	}
	return nil
}

// pruneHistory keeps the newest keep timestamped snapshots under the
// connection's prefix and deletes the rest, bounding the history cost. The
// stable wiki.zip pointer is never pruned. Best-effort: a listing or delete
// failure is surfaced (sanitized) rather than silently succeeding.
func (d *s3Driver) pruneHistory(ctx context.Context, acfg aws.Config, cfg Config, keep int) error {
	prefix := d.baseKey(cfg, "")
	keys, err := d.listKeys(ctx, acfg, cfg["bucket"], prefix)
	if err != nil {
		return err
	}
	// Keys are lexically ordered (timestamp sortable): keep the newest N.
	var history []string
	for _, k := range keys {
		if isHistoryKey(k) {
			history = append(history, k)
		}
	}
	if len(history) <= keep {
		return nil
	}
	toDelete := history[:len(history)-keep]
	if len(toDelete) > 0 {
		if _, err := d.newS3(acfg).DeleteObjects(ctx, &s3.DeleteObjectsInput{
			Bucket: aws.String(cfg["bucket"]),
			Delete: &types.Delete{Objects: deleteObjects(toDelete)},
		}); err != nil {
			return errors.New("could not prune old snapshots in the bucket")
		}
	}
	return nil
}

func deleteObjects(keys []string) []types.ObjectIdentifier {
	out := make([]types.ObjectIdentifier, 0, len(keys))
	for _, k := range keys {
		out = append(out, types.ObjectIdentifier{Key: aws.String(k)})
	}
	return out
}

// listKeys returns the object keys under prefix, paginating continuation
// tokens. S3 returns keys sorted lexically (timestamps in the snapshot keys
// are sortable), which the prune and snapshots ordering rely on. Errors are
// sanitized fixed messages.
func (d *s3Driver) listKeys(ctx context.Context, acfg aws.Config, bucket, prefix string) ([]string, error) {
	var out []string
	var token *string
	for {
		outp, err := d.newS3(acfg).ListObjectsV2(ctx, &s3.ListObjectsV2Input{
			Bucket:            aws.String(bucket),
			Prefix:            aws.String(prefix),
			ContinuationToken: token,
		})
		if err != nil {
			return nil, errors.New("could not list the bucket — check your credentials and network access")
		}
		for _, o := range outp.Contents {
			if o.Key != nil {
				out = append(out, *o.Key)
			}
		}
		if outp.IsTruncated != nil && *outp.IsTruncated {
			token = outp.NextContinuationToken
			continue
		}
		sort.Strings(out)
		return out, nil
	}
}

// Snapshots lists the stored archives newest-first: the stable wiki.zip
// pointer (if present) then timestamped history keys descending. The latest
// history key is the newest point-in-time; wiki.zip is the always-current
// pointer and sorts first as the recommended restore target.
func (d *s3Driver) Snapshots(ctx context.Context, cfg Config) ([]Snapshot, error) {
	if cfg["bucket"] == "" {
		return nil, errors.New("no bucket configured")
	}
	acfg, err := d.awsConfig(ctx, cfg)
	if err != nil {
		return nil, fmt.Errorf("configure aws: %w", err)
	}
	keys, err := d.listKeys(ctx, acfg, cfg["bucket"], d.baseKey(cfg, ""))
	if err != nil {
		return nil, err
	}
	var history []Snapshot
	for _, k := range keys {
		if isHistoryKey(k) {
			history = append(history, Snapshot{Key: k, Time: snapshotTimeFromKey(k)})
		}
	}
	// Newest first: history keys sort lexically ascending, so reverse.
	sort.Slice(history, func(i, j int) bool { return history[i].Key > history[j].Key })
	out := make([]Snapshot, 0, len(history)+1)
	if hasStableKey(keys) {
		out = append(out, Snapshot{Key: d.baseKey(cfg, "wiki.zip"), Time: ""})
	}
	out = append(out, history...)
	return out, nil
}

func hasStableKey(keys []string) bool {
	for _, k := range keys {
		if k == "wiki.zip" || strings.HasSuffix(k, "/wiki.zip") {
			return true
		}
	}
	return false
}

// snapshotTimeFromKey extracts the RFC3339 time from a thoth-wiki-<ts>.zip
// key; "" when the key does not parse.
func snapshotTimeFromKey(key string) string {
	name := key
	if i := strings.LastIndex(name, "/"); i >= 0 {
		name = name[i+1:]
	}
	rest := strings.TrimPrefix(name, snapshotPrefix)
	rest = strings.TrimSuffix(rest, ".zip")
	t, err := time.Parse(SnapshotTimeFormat, rest)
	if err != nil {
		return ""
	}
	return t.UTC().Format(time.RFC3339)
}

// Restore returns the archive at key as a ReaderAt + size for the API's
// import flow. An empty key selects the latest: the stable wiki.zip pointer
// when present, otherwise the newest history snapshot.
func (d *s3Driver) Restore(ctx context.Context, cfg Config, key string) (io.ReaderAt, int64, error) {
	if cfg["bucket"] == "" {
		return nil, 0, errors.New("no bucket configured")
	}
	acfg, err := d.awsConfig(ctx, cfg)
	if err != nil {
		return nil, 0, fmt.Errorf("configure aws: %w", err)
	}
	if key == "" {
		snaps, err := d.Snapshots(ctx, cfg)
		if err != nil {
			return nil, 0, err
		}
		if len(snaps) == 0 {
			return nil, 0, errors.New("no snapshots in the bucket to restore")
		}
		key = snaps[0].Key
	}
	out, err := d.newS3(acfg).GetObject(ctx, &s3.GetObjectInput{Bucket: aws.String(cfg["bucket"]), Key: aws.String(key)})
	if err != nil {
		return nil, 0, errors.New("could not download the snapshot from the bucket")
	}
	defer func() { _ = out.Body.Close() }()
	var buf bytes.Buffer
	if _, err := io.Copy(&buf, out.Body); err != nil {
		return nil, 0, errors.New("could not download the snapshot from the bucket")
	}
	return bytes.NewReader(buf.Bytes()), int64(buf.Len()), nil
}
