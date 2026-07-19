package storage

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"time"

	"github.com/minio/minio-go/v7"
	"github.com/minio/minio-go/v7/pkg/credentials"
)

// Config holds the S3-compatible object-storage settings. Endpoint is the host
// the server reaches the store on (e.g. "minio:9000" inside docker); PublicEndpoint
// is the host the CLIENT can reach and is used only to build presigned URLs. When
// they differ (the common docker case) the two must resolve to the same store.
type Config struct {
	Endpoint       string
	PublicEndpoint string
	Region         string
	Bucket         string
	AccessKey      string
	SecretKey      string
	UseSSL         bool
}

// MinioObjectStore implements ObjectStore against MinIO/S3. It holds two clients:
//   - internal: real requests (Stat/Remove/EnsureBucket) against Endpoint.
//   - presign:  URL signing against PublicEndpoint. Presigning is pure HMAC with
//     no network call, so it is safe that the server itself may not be able to
//     reach PublicEndpoint — only the client needs to.
type MinioObjectStore struct {
	internal *minio.Client
	presign  *minio.Client
	bucket   string
}

var _ ObjectStore = (*MinioObjectStore)(nil)

// NewMinioObjectStore builds the store from config. It returns (nil, nil) when
// Endpoint is empty so the caller can boot with gallery storage disabled (the
// gallery endpoints then answer 503) without failing liveness — mirroring the
// nil-pool tolerance of the repositories.
func NewMinioObjectStore(cfg Config) (*MinioObjectStore, error) {
	if cfg.Endpoint == "" {
		return nil, nil
	}
	creds := credentials.NewStaticV4(cfg.AccessKey, cfg.SecretKey, "")
	opts := &minio.Options{Creds: creds, Secure: cfg.UseSSL, Region: cfg.Region}

	internal, err := minio.New(cfg.Endpoint, opts)
	if err != nil {
		return nil, fmt.Errorf("storage: init internal client: %w", err)
	}
	publicEndpoint := cfg.PublicEndpoint
	if publicEndpoint == "" {
		publicEndpoint = cfg.Endpoint
	}
	presign, err := minio.New(publicEndpoint, opts)
	if err != nil {
		return nil, fmt.Errorf("storage: init presign client: %w", err)
	}
	return &MinioObjectStore{internal: internal, presign: presign, bucket: cfg.Bucket}, nil
}

func (s *MinioObjectStore) EnsureBucket(ctx context.Context) error {
	exists, err := s.internal.BucketExists(ctx, s.bucket)
	if err != nil {
		return fmt.Errorf("storage: bucket exists: %w", err)
	}
	if exists {
		return nil
	}
	if err := s.internal.MakeBucket(ctx, s.bucket, minio.MakeBucketOptions{}); err != nil {
		return fmt.Errorf("storage: make bucket: %w", err)
	}
	return nil
}

func (s *MinioObjectStore) PresignPut(ctx context.Context, key string, ttl time.Duration) (string, error) {
	u, err := s.presign.PresignedPutObject(ctx, s.bucket, key, ttl)
	if err != nil {
		return "", fmt.Errorf("storage: presign put: %w", err)
	}
	return u.String(), nil
}

func (s *MinioObjectStore) PresignGet(ctx context.Context, key string, ttl time.Duration) (string, error) {
	u, err := s.presign.PresignedGetObject(ctx, s.bucket, key, ttl, nil)
	if err != nil {
		return "", fmt.Errorf("storage: presign get: %w", err)
	}
	return u.String(), nil
}

func (s *MinioObjectStore) Stat(ctx context.Context, key string) (ObjectInfo, error) {
	info, err := s.internal.StatObject(ctx, s.bucket, key, minio.StatObjectOptions{})
	if err != nil {
		if isNotFound(err) {
			return ObjectInfo{}, ErrObjectNotFound
		}
		return ObjectInfo{}, fmt.Errorf("storage: stat: %w", err)
	}
	return ObjectInfo{Size: info.Size, ContentType: info.ContentType}, nil
}

func (s *MinioObjectStore) Remove(ctx context.Context, key string) error {
	if err := s.internal.RemoveObject(ctx, s.bucket, key, minio.RemoveObjectOptions{}); err != nil {
		if isNotFound(err) {
			return nil // idempotent
		}
		return fmt.Errorf("storage: remove: %w", err)
	}
	return nil
}

// isNotFound reports whether a minio error is a 404/NoSuchKey, which the store
// surfaces as ErrObjectNotFound.
func isNotFound(err error) bool {
	if errors.Is(err, ErrObjectNotFound) {
		return true
	}
	resp := minio.ToErrorResponse(err)
	return resp.StatusCode == http.StatusNotFound ||
		resp.Code == "NoSuchKey" || resp.Code == "NoSuchBucket"
}
