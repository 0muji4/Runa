package apns_test

import (
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/x509"
	"encoding/json"
	"encoding/pem"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/0muji4/Runa/apps/go/internal/push"
	"github.com/0muji4/Runa/apps/go/internal/push/apns"
	"github.com/golang-jwt/jwt/v5"
	"github.com/google/go-cmp/cmp"
)

// testKey is a P-256 key in the PKCS#8 PEM form Apple ships in AuthKey_*.p8.
func testKey(t *testing.T) (*ecdsa.PrivateKey, []byte) {
	t.Helper()
	key, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	if err != nil {
		t.Fatal(err)
	}
	der, err := x509.MarshalPKCS8PrivateKey(key)
	if err != nil {
		t.Fatal(err)
	}
	return key, pem.EncodeToMemory(&pem.Block{Type: "PRIVATE KEY", Bytes: der})
}

// seen is one request the fake APNs received.
type seen struct {
	path    string
	headers http.Header
	body    map[string]any
}

// reply is one canned APNs answer.
type reply struct {
	status int
	reason string
}

// appleFake is an HTTP/2 APNs stand-in that answers replies in order (the last one repeats).
type appleFake struct {
	srv     *httptest.Server
	mu      sync.Mutex
	replies []reply
	seen    []seen
}

func newAppleFake(t *testing.T, replies ...reply) *appleFake {
	t.Helper()
	f := &appleFake{replies: replies}
	f.srv = httptest.NewUnstartedServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.ProtoMajor != 2 {
			t.Errorf("request proto = %s, want HTTP/2", r.Proto)
		}
		var body map[string]any
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			t.Errorf("decode body: %v", err)
		}
		f.mu.Lock()
		f.seen = append(f.seen, seen{path: r.URL.Path, headers: r.Header.Clone(), body: body})
		rep := f.replies[0]
		if len(f.replies) > 1 {
			f.replies = f.replies[1:]
		}
		f.mu.Unlock()

		w.Header().Set("apns-id", r.Header.Get("apns-id"))
		w.WriteHeader(rep.status)
		if rep.reason != "" {
			_ = json.NewEncoder(w).Encode(map[string]string{"reason": rep.reason})
		}
	}))
	f.srv.EnableHTTP2 = true
	f.srv.StartTLS()
	t.Cleanup(f.srv.Close)
	return f
}

func (f *appleFake) requests() []seen {
	f.mu.Lock()
	defer f.mu.Unlock()
	return append([]seen(nil), f.seen...)
}

func newClient(t *testing.T, f *appleFake, pemKey []byte) *apns.Client {
	t.Helper()
	c, err := apns.NewClient(apns.Config{
		TeamID: "TEAM123456", KeyID: "KEY1234567", PrivateKeyPEM: pemKey,
		BundleID: "com.example.runa", Host: f.srv.URL,
	}, f.srv.Client())
	if err != nil {
		t.Fatalf("NewClient() error = %v", err)
	}
	return c
}

func TestClient_Send_RequestShape(t *testing.T) {
	t.Parallel()
	key, pemKey := testKey(t)
	f := newAppleFake(t, reply{status: http.StatusOK})
	c := newClient(t, f, pemKey)

	expiry := time.Date(2026, 9, 21, 12, 0, 0, 0, time.UTC)
	got, err := c.Send(t.Context(), "abc123", push.Notification{
		Title:      "月夜のしおり",
		Body:       "今日の記録を残しましょう",
		Data:       map[string]string{"kind": "reminder", "date": "2026-09-21", "aps": "ignored"},
		CollapseID: "reminder",
		Expiry:     expiry,
	})
	if err != nil {
		t.Fatalf("Send() error = %v", err)
	}

	reqs := f.requests()
	if len(reqs) != 1 {
		t.Fatalf("requests = %d, want 1", len(reqs))
	}
	r := reqs[0]
	if r.path != "/3/device/abc123" {
		t.Errorf("path = %q, want /3/device/abc123", r.path)
	}
	wantHeaders := map[string]string{
		"apns-topic":       "com.example.runa",
		"apns-push-type":   "alert",
		"apns-priority":    "10",
		"apns-collapse-id": "reminder",
		"apns-expiration":  "1789992000",
	}
	for k, want := range wantHeaders {
		if v := r.headers.Get(k); v != want {
			t.Errorf("header %s = %q, want %q", k, v, want)
		}
	}
	apnsID := r.headers.Get("apns-id")
	if len(apnsID) != 36 {
		t.Errorf("apns-id = %q, want a UUID", apnsID)
	}
	if got.ProviderMessageID != apnsID {
		t.Errorf("ProviderMessageID = %q, want the apns-id %q", got.ProviderMessageID, apnsID)
	}

	wantBody := map[string]any{
		"aps": map[string]any{
			"alert": map[string]any{"title": "月夜のしおり", "body": "今日の記録を残しましょう"},
			"sound": "default",
		},
		"kind": "reminder",
		"date": "2026-09-21",
	}
	if diff := cmp.Diff(wantBody, r.body); diff != "" {
		t.Errorf("body mismatch (-want +got):\n%s", diff)
	}

	bearer := strings.TrimPrefix(r.headers.Get("authorization"), "bearer ")
	claims := jwt.MapClaims{}
	tok, err := jwt.ParseWithClaims(bearer, claims, func(*jwt.Token) (any, error) { return &key.PublicKey, nil },
		jwt.WithValidMethods([]string{"ES256"}))
	if err != nil {
		t.Fatalf("parse provider token: %v", err)
	}
	if tok.Header["kid"] != "KEY1234567" {
		t.Errorf("kid = %v, want KEY1234567", tok.Header["kid"])
	}
	if claims["iss"] != "TEAM123456" {
		t.Errorf("iss = %v, want TEAM123456", claims["iss"])
	}
	if _, ok := claims["iat"]; !ok {
		t.Error("iat claim missing")
	}
}

func TestClient_Send_OmitsOptionalHeaders(t *testing.T) {
	t.Parallel()
	_, pemKey := testKey(t)
	f := newAppleFake(t, reply{status: http.StatusOK})

	if _, err := newClient(t, f, pemKey).Send(t.Context(), "tok", push.Notification{Title: "t"}); err != nil {
		t.Fatalf("Send() error = %v", err)
	}
	h := f.requests()[0].headers
	for _, k := range []string{"apns-collapse-id", "apns-expiration"} {
		if _, ok := h[http.CanonicalHeaderKey(k)]; ok {
			t.Errorf("header %s present, want omitted", k)
		}
	}
}

func TestClient_Send_ErrorMapping(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name    string
		reply   reply
		wantIs  error // nil means a plain error
		wantErr bool
	}{
		{name: "BadDeviceToken is invalid token", reply: reply{400, "BadDeviceToken"}, wantIs: push.ErrInvalidToken, wantErr: true},
		{name: "Unregistered is invalid token", reply: reply{410, "Unregistered"}, wantIs: push.ErrInvalidToken, wantErr: true},
		{name: "ExpiredToken is invalid token", reply: reply{400, "ExpiredToken"}, wantIs: push.ErrInvalidToken, wantErr: true},
		{name: "DeviceTokenNotForTopic is invalid token", reply: reply{400, "DeviceTokenNotForTopic"}, wantIs: push.ErrInvalidToken, wantErr: true},
		{name: "TopicDisallowed is invalid token", reply: reply{400, "TopicDisallowed"}, wantIs: push.ErrInvalidToken, wantErr: true},
		{name: "429 is transient", reply: reply{429, "TooManyRequests"}, wantIs: push.ErrTransient, wantErr: true},
		{name: "503 is transient", reply: reply{503, "ServiceUnavailable"}, wantIs: push.ErrTransient, wantErr: true},
		{name: "500 is transient", reply: reply{500, "InternalServerError"}, wantIs: push.ErrTransient, wantErr: true},
		{name: "Shutdown is transient", reply: reply{503, "Shutdown"}, wantIs: push.ErrTransient, wantErr: true},
		{name: "5xx without a reason is transient", reply: reply{status: 502}, wantIs: push.ErrTransient, wantErr: true},
		{name: "BadMessageId is a plain error", reply: reply{400, "BadMessageId"}, wantErr: true},
		{name: "PayloadTooLarge is a plain error", reply: reply{413, "PayloadTooLarge"}, wantErr: true},
		{name: "MissingTopic is a plain error", reply: reply{400, "MissingTopic"}, wantErr: true},
		{name: "200 is success", reply: reply{status: 200}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			_, pemKey := testKey(t)
			f := newAppleFake(t, tt.reply)

			_, err := newClient(t, f, pemKey).Send(t.Context(), "tok", push.Notification{Title: "t"})
			if !tt.wantErr {
				if err != nil {
					t.Fatalf("Send() error = %v, want nil", err)
				}
				return
			}
			if err == nil {
				t.Fatal("Send() error = nil, want an error")
			}
			if tt.wantIs != nil && !errors.Is(err, tt.wantIs) {
				t.Errorf("Send() error = %v, want errors.Is %v", err, tt.wantIs)
			}
			if tt.wantIs == nil && (errors.Is(err, push.ErrInvalidToken) || errors.Is(err, push.ErrTransient)) {
				t.Errorf("Send() error = %v, want a plain error", err)
			}
			if tt.reply.reason != "" && !strings.Contains(err.Error(), tt.reply.reason) {
				t.Errorf("Send() error = %v, want it to mention %s", err, tt.reply.reason)
			}
			if len(f.requests()) != 1 {
				t.Errorf("requests = %d, want 1 (no retry)", len(f.requests()))
			}
		})
	}
}

func TestClient_Send_TransportFailureIsTransient(t *testing.T) {
	t.Parallel()
	_, pemKey := testKey(t)
	f := newAppleFake(t, reply{status: http.StatusOK})
	c := newClient(t, f, pemKey)
	f.srv.Close()

	_, err := c.Send(t.Context(), "tok", push.Notification{Title: "t"})
	if !errors.Is(err, push.ErrTransient) {
		t.Errorf("Send() error = %v, want errors.Is ErrTransient", err)
	}
}

func TestClient_Send_ReusesProviderToken(t *testing.T) {
	t.Parallel()
	_, pemKey := testKey(t)
	f := newAppleFake(t, reply{status: http.StatusOK})
	c := newClient(t, f, pemKey)

	for i := 0; i < 2; i++ {
		if _, err := c.Send(t.Context(), "tok", push.Notification{Title: "t"}); err != nil {
			t.Fatalf("Send() #%d error = %v", i+1, err)
		}
	}
	reqs := f.requests()
	if len(reqs) != 2 {
		t.Fatalf("requests = %d, want 2", len(reqs))
	}
	if a, b := reqs[0].headers.Get("authorization"), reqs[1].headers.Get("authorization"); a != b {
		t.Errorf("second send used a different bearer:\n%s\n%s", a, b)
	}
}

func TestClient_Send_ExpiredProviderTokenRetriesOnceWithFreshToken(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name   string
		reason string
	}{
		{name: "ExpiredProviderToken", reason: "ExpiredProviderToken"},
		{name: "InvalidProviderToken", reason: "InvalidProviderToken"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			_, pemKey := testKey(t)
			f := newAppleFake(t, reply{403, tt.reason}, reply{status: http.StatusOK})
			c := newClient(t, f, pemKey)

			if _, err := c.Send(t.Context(), "tok", push.Notification{Title: "t"}); err != nil {
				t.Fatalf("Send() error = %v, want nil after retry", err)
			}
			reqs := f.requests()
			if len(reqs) != 2 {
				t.Fatalf("requests = %d, want 2 (one retry)", len(reqs))
			}
			if a, b := reqs[0].headers.Get("authorization"), reqs[1].headers.Get("authorization"); a == b {
				t.Error("retry reused the rejected bearer, want a fresh one")
			}

			// A later 403 within the 20-minute floor must not mint again, and Send must not loop.
			f.mu.Lock()
			f.replies = []reply{{403, tt.reason}}
			f.mu.Unlock()
			if _, err := c.Send(t.Context(), "tok", push.Notification{Title: "t"}); err == nil {
				t.Fatal("Send() error = nil, want the 403 surfaced")
			}
			reqs = f.requests()
			if len(reqs) != 4 {
				t.Fatalf("requests = %d, want 4", len(reqs))
			}
			if reqs[2].headers.Get("authorization") != reqs[3].headers.Get("authorization") {
				t.Error("second forced refresh minted within 20 minutes, want reuse")
			}
			if reqs[1].headers.Get("authorization") != reqs[2].headers.Get("authorization") {
				t.Error("third send did not reuse the refreshed bearer")
			}
		})
	}
}

func TestHostFor(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name    string
		env     string
		want    string
		wantErr bool
	}{
		{name: "sandbox", env: "sandbox", want: apns.SandboxHost},
		{name: "production", env: "production", want: apns.ProductionHost},
		{name: "unknown is an error", env: "staging", wantErr: true},
		{name: "empty is an error", env: "", wantErr: true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			got, err := apns.HostFor(tt.env)
			if (err != nil) != tt.wantErr {
				t.Fatalf("HostFor(%q) error = %v, wantErr %v", tt.env, err, tt.wantErr)
			}
			if got != tt.want {
				t.Errorf("HostFor(%q) = %q, want %q", tt.env, got, tt.want)
			}
		})
	}
}

func TestNewClient_Validation(t *testing.T) {
	t.Parallel()
	_, pemKey := testKey(t)
	valid := apns.Config{TeamID: "T", KeyID: "K", PrivateKeyPEM: pemKey, BundleID: "b", Host: apns.SandboxHost}
	tests := []struct {
		name   string
		mutate func(*apns.Config)
	}{
		{name: "missing TeamID", mutate: func(c *apns.Config) { c.TeamID = "" }},
		{name: "missing KeyID", mutate: func(c *apns.Config) { c.KeyID = "" }},
		{name: "missing BundleID", mutate: func(c *apns.Config) { c.BundleID = "" }},
		{name: "missing Host", mutate: func(c *apns.Config) { c.Host = "" }},
		{name: "missing key", mutate: func(c *apns.Config) { c.PrivateKeyPEM = nil }},
		{name: "malformed key", mutate: func(c *apns.Config) { c.PrivateKeyPEM = []byte("not a pem") }},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			cfg := valid
			tt.mutate(&cfg)
			if _, err := apns.NewClient(cfg, nil); err == nil {
				t.Error("NewClient() error = nil, want an error")
			}
		})
	}
	if _, err := apns.NewClient(valid, nil); err != nil {
		t.Errorf("NewClient(valid) error = %v", err)
	}
}
