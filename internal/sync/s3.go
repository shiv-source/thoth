package sync

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"strings"

	"github.com/aws/aws-sdk-go-v2/aws"
	awsconfig "github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/credentials"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/aws/aws-sdk-go-v2/service/sts"
	"github.com/shiv-source/thoth/internal/wiki"
)

// s3Driver pushes a wiki snapshot zip to an S3 bucket (or any S3-compatible
// endpoint via the provider's base_url). Verify proves the keys with
// sts:GetCallerIdentity on AWS (falling back to a bucket probe when sts is
// denied) or a bucket probe on custom endpoints that have no STS.
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
	if d.endpoint != "" {
		return s3.NewFromConfig(cfg, func(o *s3.Options) { o.BaseEndpoint = aws.String(d.endpoint) })
	}
	return s3.NewFromConfig(cfg)
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

// Push zips the wiki and uploads it to s3://bucket[/prefix]/wiki.zip. Errors
// are sanitized fixed messages; the raw SDK error may echo bucket/key names.
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
	key := strings.Trim(cfg["prefix"], "/")
	if key != "" {
		key += "/"
	}
	key += "wiki.zip"
	if _, err := d.newS3(acfg).PutObject(ctx, &s3.PutObjectInput{
		Bucket:      aws.String(cfg["bucket"]),
		Key:         aws.String(key),
		Body:        bytes.NewReader(buf.Bytes()),
		ContentType: aws.String("application/zip"),
	}); err != nil {
		return errors.New("could not upload the wiki to the bucket — check your credentials and network access")
	}
	return nil
}
