package auth

import (
	"context"
	"crypto/rand"
	"crypto/rsa"
	"encoding/base64"
	"encoding/json"
	"math/big"
	"net/http"
	"net/http/httptest"
	"sync/atomic"
	"testing"
	"time"
)

// RemoteJWKS: the path that talks to Apple's and Google's key endpoints. In
// package auth because the cache clock and TTL are unexported; from outside,
// testing the cache would mean waiting an hour.

// jwksServer counts requests, so a test can tell a cache hit from a refetch.
type jwksServer struct {
	*httptest.Server
	requests atomic.Int64
}

// newJWKSServer serves the given keys, keyed by kid.
func newJWKSServer(t *testing.T, keys map[string]*rsa.PublicKey) *jwksServer {
	t.Helper()
	s := &jwksServer{}
	s.Server = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		s.requests.Add(1)
		w.Header().Set("Content-Type", "application/json")
		if err := json.NewEncoder(w).Encode(jwksDocumentFor(keys)); err != nil {
			t.Errorf("encoding the JWKS response: %v", err)
		}
	}))
	t.Cleanup(s.Close)
	return s
}

// jwksDocumentFor renders keys in the JWKS shape the providers publish.
func jwksDocumentFor(keys map[string]*rsa.PublicKey) map[string]any {
	entries := make([]map[string]string, 0, len(keys))
	for kid, pub := range keys {
		entries = append(entries, map[string]string{
			"kty": "RSA",
			"kid": kid,
			"alg": "RS256",
			"use": "sig",
			"n":   base64.RawURLEncoding.EncodeToString(pub.N.Bytes()),
			"e":   base64.RawURLEncoding.EncodeToString(big.NewInt(int64(pub.E)).Bytes()),
		})
	}
	return map[string]any{"keys": entries}
}

func newTestKey(t *testing.T) *rsa.PrivateKey {
	t.Helper()
	key, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		t.Fatalf("rsa.GenerateKey() error = %v, want nil", err)
	}
	return key
}

func TestRemoteJWKS_Keys(t *testing.T) {
	t.Parallel()

	priv := newTestKey(t)
	server := newJWKSServer(t, map[string]*rsa.PublicKey{"kid-1": &priv.PublicKey})

	r := NewRemoteJWKS(server.URL)
	keys, err := r.Keys(t.Context())
	if err != nil {
		t.Fatalf("Keys() error = %v, want nil", err)
	}
	got, ok := keys["kid-1"]
	if !ok {
		t.Fatalf("Keys() returned %d keys with no \"kid-1\", want the served key", len(keys))
	}
	if got.N.Cmp(priv.PublicKey.N) != 0 {
		t.Error("the decoded modulus differs from the served key")
	}
	if got.E != priv.PublicKey.E {
		t.Errorf("the decoded exponent = %d, want %d", got.E, priv.PublicKey.E)
	}
}

func TestRemoteJWKS_CachesWithinTTL(t *testing.T) {
	t.Parallel()

	priv := newTestKey(t)
	server := newJWKSServer(t, map[string]*rsa.PublicKey{"kid-1": &priv.PublicKey})

	base := time.Now()
	now := base
	r := NewRemoteJWKS(server.URL)
	r.now = func() time.Time { return now }

	tests := []struct {
		name         string
		advance      time.Duration
		wantRequests int64
	}{
		{
			name:         "初回は取得しにいく",
			advance:      0,
			wantRequests: 1,
		},
		{
			name:         "TTL内は再取得しない",
			advance:      r.ttl / 2,
			wantRequests: 1,
		},
		{
			name:         "TTL直前もまだキャッシュ",
			advance:      r.ttl - time.Second,
			wantRequests: 1,
		},
		{
			name:         "TTL経過で取り直す",
			advance:      r.ttl + time.Second,
			wantRequests: 2,
		},
	}
	for _, tt := range tests {
		// Sequential on purpose: each step advances the same shared clock.
		t.Run(tt.name, func(t *testing.T) {
			now = base.Add(tt.advance)
			if _, err := r.Keys(t.Context()); err != nil {
				t.Fatalf("Keys() at t+%s error = %v, want nil", tt.advance, err)
			}
			if got := server.requests.Load(); got != tt.wantRequests {
				t.Errorf("after t+%s the endpoint saw %d requests, want %d",
					tt.advance, got, tt.wantRequests)
			}
		})
	}
}

func TestRemoteJWKS_FetchFailures(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		handler http.HandlerFunc
	}{
		{
			name: "5xxは失敗として扱う",
			handler: func(w http.ResponseWriter, r *http.Request) {
				w.WriteHeader(http.StatusInternalServerError)
			},
		},
		{
			name: "404は失敗として扱う",
			handler: func(w http.ResponseWriter, r *http.Request) {
				w.WriteHeader(http.StatusNotFound)
			},
		},
		{
			name: "壊れたJSONは失敗として扱う",
			handler: func(w http.ResponseWriter, r *http.Request) {
				_, _ = w.Write([]byte("{not json"))
			},
		},
		{
			name:    "空ボディは失敗として扱う",
			handler: func(w http.ResponseWriter, r *http.Request) {},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			server := httptest.NewServer(tt.handler)
			t.Cleanup(server.Close)

			if _, err := NewRemoteJWKS(server.URL).Keys(t.Context()); err == nil {
				t.Error("Keys() error = nil, want a fetch failure")
			}
		})
	}
}

func TestRemoteJWKS_SkipsNonRSAKeys(t *testing.T) {
	t.Parallel()

	priv := newTestKey(t)
	// An unsupported key type must be skipped, not turned into a decode failure
	// that blocks every login.
	doc := jwksDocumentFor(map[string]*rsa.PublicKey{"rsa-kid": &priv.PublicKey})
	doc["keys"] = append(doc["keys"].([]map[string]string), map[string]string{
		"kty": "EC", "kid": "ec-kid", "crv": "P-256", "x": "abc", "y": "def",
	})

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if err := json.NewEncoder(w).Encode(doc); err != nil {
			t.Errorf("encoding the JWKS response: %v", err)
		}
	}))
	t.Cleanup(server.Close)

	keys, err := NewRemoteJWKS(server.URL).Keys(t.Context())
	if err != nil {
		t.Fatalf("Keys() error = %v, want nil", err)
	}
	if _, ok := keys["rsa-kid"]; !ok {
		t.Error("the RSA key was dropped, want it kept")
	}
	if _, ok := keys["ec-kid"]; ok {
		t.Error("the EC key was decoded, want it skipped")
	}
}

func TestRemoteJWKS_HonoursContextCancellation(t *testing.T) {
	t.Parallel()

	// A provider that never answers: the context is the only way out.
	blocked := make(chan struct{})
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		<-blocked
	}))
	t.Cleanup(func() {
		close(blocked)
		server.Close()
	})

	ctx, cancel := context.WithCancel(t.Context())
	go func() {
		time.Sleep(50 * time.Millisecond)
		cancel()
	}()

	if _, err := NewRemoteJWKS(server.URL).Keys(ctx); err == nil {
		t.Error("Keys() error = nil, want the cancelled context to abort the fetch")
	}
}
