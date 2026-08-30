package storage_test

import (
	"bytes"
	"context"
	"flag"
	"fmt"
	"io"
	"net/http"
	"os"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"github.com/testcontainers/testcontainers-go"
	tcminio "github.com/testcontainers/testcontainers-go/modules/minio"

	"github.com/0muji4/Runa/apps/go/internal/storage"
	"github.com/0muji4/Runa/apps/go/internal/storage/objecttest"
)

// The MinIO client against a real S3-compatible server. The in-package tests only
// inspect the presigned URL strings; these check the URLs actually work.

var (
	minioEndpoint  string
	minioAccessKey string
	minioSecretKey string
	skipReason     string

	bucketSeq atomic.Int64
)

func TestMain(m *testing.M) {
	// testing.Short() reads a flag, and TestMain runs before they are parsed.
	flag.Parse()
	if testing.Short() {
		skipReason = "-short is set; skipping the MinIO-backed tests"
		os.Exit(m.Run())
	}

	teardown, err := startMinio(context.Background())
	if err != nil {
		skipReason = fmt.Sprintf(
			"could not start a MinIO container (%v); "+
				"start Docker to run the object-storage tests, or pass -short to skip them", err)
		os.Exit(m.Run())
	}

	code := m.Run()
	teardown()
	os.Exit(code)
}

func startMinio(ctx context.Context) (func(), error) {
	const (
		user     = "runa"
		password = "runa-secret"
	)
	container, err := tcminio.Run(ctx, "minio/minio:latest",
		tcminio.WithUsername(user),
		tcminio.WithPassword(password),
	)
	if err != nil {
		return nil, fmt.Errorf("run container: %w", err)
	}
	terminate := func() { _ = testcontainers.TerminateContainer(container) }

	endpoint, err := container.ConnectionString(ctx)
	if err != nil {
		terminate()
		return nil, fmt.Errorf("connection string: %w", err)
	}
	minioEndpoint = strings.TrimPrefix(endpoint, "http://")
	minioAccessKey = user
	minioSecretKey = password
	return terminate, nil
}

func requireMinio(t *testing.T) {
	t.Helper()
	if skipReason != "" {
		t.Skip(skipReason)
	}
}

// newStore builds a store over a bucket of its own.
func newStore(t *testing.T) *storage.MinioObjectStore {
	t.Helper()
	cfg := storage.Config{
		Endpoint:  minioEndpoint,
		Region:    "us-east-1",
		Bucket:    fmt.Sprintf("runa-test-%d", bucketSeq.Add(1)),
		AccessKey: minioAccessKey,
		SecretKey: minioSecretKey,
		UseSSL:    false,
	}
	s, err := storage.NewMinioObjectStore(cfg)
	if err != nil {
		t.Fatalf("NewMinioObjectStore(%+v) error = %v, want nil", cfg, err)
	}
	if s == nil {
		t.Fatalf("NewMinioObjectStore(%+v) = nil, want a store", cfg)
	}
	if err := s.EnsureBucket(t.Context()); err != nil {
		t.Fatalf("EnsureBucket() error = %v, want nil", err)
	}
	return s
}

// presignWriter seeds objects the way production does: through a presigned PUT.
type presignWriter struct{ *storage.MinioObjectStore }

func (w *presignWriter) PutForTest(ctx context.Context, key string, body []byte, contentType string) error {
	url, err := w.PresignPut(ctx, key, 15*time.Minute)
	if err != nil {
		return fmt.Errorf("presign put: %w", err)
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPut, url, bytes.NewReader(body))
	if err != nil {
		return fmt.Errorf("build request: %w", err)
	}
	req.Header.Set("Content-Type", contentType)
	res, err := http.DefaultClient.Do(req)
	if err != nil {
		return fmt.Errorf("upload: %w", err)
	}
	defer res.Body.Close()
	if res.StatusCode != http.StatusOK {
		return fmt.Errorf("upload: status %d", res.StatusCode)
	}
	return nil
}

func TestMinioMeetsTheObjectStoreContract(t *testing.T) {
	requireMinio(t)
	t.Parallel()
	objecttest.RunObjectStoreSuite(t, func(t *testing.T) storage.ObjectStore {
		t.Helper()
		return &presignWriter{newStore(t)}
	})
}

// TestPresignedURLRoundTrip uploads and downloads through the presigned URLs,
// which is the whole point of the design: the API never streams image bytes.
func TestPresignedURLRoundTrip(t *testing.T) {
	requireMinio(t)
	t.Parallel()

	s := newStore(t)
	ctx := t.Context()
	key := "gallery/user-1/round-trip.jpg"
	body := []byte("\xff\xd8\xff\xe0 not really a jpeg, but bytes are bytes")

	putURL, err := s.PresignPut(ctx, key, 15*time.Minute)
	if err != nil {
		t.Fatalf("PresignPut(%q) error = %v, want nil", key, err)
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPut, putURL, bytes.NewReader(body))
	if err != nil {
		t.Fatalf("building the upload request: %v", err)
	}
	req.Header.Set("Content-Type", "image/jpeg")
	putRes, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatalf("PUT to the presigned URL: %v", err)
	}
	defer putRes.Body.Close()
	if putRes.StatusCode != http.StatusOK {
		t.Fatalf("PUT to the presigned URL = %d, want %d", putRes.StatusCode, http.StatusOK)
	}

	getURL, err := s.PresignGet(ctx, key, time.Hour)
	if err != nil {
		t.Fatalf("PresignGet(%q) error = %v, want nil", key, err)
	}
	getRes, err := http.Get(getURL)
	if err != nil {
		t.Fatalf("GET from the presigned URL: %v", err)
	}
	defer getRes.Body.Close()
	if getRes.StatusCode != http.StatusOK {
		t.Fatalf("GET from the presigned URL = %d, want %d", getRes.StatusCode, http.StatusOK)
	}
	got, err := io.ReadAll(getRes.Body)
	if err != nil {
		t.Fatalf("reading the downloaded body: %v", err)
	}
	if !bytes.Equal(got, body) {
		t.Errorf("downloaded %d bytes, want the %d uploaded", len(got), len(body))
	}
}

// TestPresignedURLExpires pins the TTL: a leaked URL must not grant permanent
// access to a private image.
func TestPresignedURLExpires(t *testing.T) {
	requireMinio(t)
	t.Parallel()

	s := newStore(t)
	ctx := t.Context()
	key := "gallery/user-1/expiring.jpg"

	if err := (&presignWriter{s}).PutForTest(ctx, key, []byte("bytes"), "image/jpeg"); err != nil {
		t.Fatalf("seeding the object: %v", err)
	}

	// The one real sleep in the suite: the clock that matters belongs to the MinIO
	// server validating the signature, so synctest cannot virtualize it. 1s is the
	// shortest TTL an S3 signature allows.
	getURL, err := s.PresignGet(ctx, key, time.Second)
	if err != nil {
		t.Fatalf("PresignGet(%q) error = %v, want nil", key, err)
	}
	time.Sleep(2 * time.Second)

	res, err := http.Get(getURL)
	if err != nil {
		t.Fatalf("GET from the expired URL: %v", err)
	}
	defer res.Body.Close()
	if res.StatusCode == http.StatusOK {
		t.Errorf("GET from an expired presigned URL = %d, want a rejection", res.StatusCode)
	}
}
