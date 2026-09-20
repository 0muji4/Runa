package cloudtasks_test

import (
	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"encoding/json"
	"encoding/pem"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/0muji4/Runa/apps/go/internal/schedule"
	"github.com/0muji4/Runa/apps/go/internal/schedule/cloudtasks"
	"github.com/golang-jwt/jwt/v5"
	"github.com/google/go-cmp/cmp"
)

const (
	saEmail   = "runa@p.iam.gserviceaccount.com"
	wantScope = "https://www.googleapis.com/auth/cloud-platform"
	queuePath = "/v2/projects/p/locations/asia-northeast1/queues/q/tasks"
)

// serviceAccountJSON builds a GCP key file around a fresh RSA key.
func serviceAccountJSON(t *testing.T, tokenURI string) (*rsa.PrivateKey, []byte) {
	t.Helper()
	key, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		t.Fatal(err)
	}
	der, err := x509.MarshalPKCS8PrivateKey(key)
	if err != nil {
		t.Fatal(err)
	}
	pemKey := pem.EncodeToMemory(&pem.Block{Type: "PRIVATE KEY", Bytes: der})
	sa, err := json.Marshal(map[string]string{
		"type":         "service_account",
		"project_id":   "p",
		"client_email": saEmail,
		"private_key":  string(pemKey),
		"token_uri":    tokenURI,
	})
	if err != nil {
		t.Fatal(err)
	}
	return key, sa
}

// call is one tasks request the fake received.
type call struct {
	method string
	path   string
	bearer string
	body   map[string]any
}

// googleFake serves the token endpoint and the tasks collection from one server.
type googleFake struct {
	srv       *httptest.Server
	key       *rsa.PrivateKey
	mu        sync.Mutex
	status    int
	body      string
	tokenHits int
	calls     []call
}

func newGoogleFake(t *testing.T, status int, body string) (*googleFake, []byte) {
	t.Helper()
	f := &googleFake{status: status, body: body}
	mux := http.NewServeMux()
	mux.HandleFunc("/token", func(w http.ResponseWriter, r *http.Request) {
		f.mu.Lock()
		f.tokenHits++
		f.mu.Unlock()
		if err := r.ParseForm(); err != nil {
			t.Errorf("parse token form: %v", err)
		}
		if got := r.Form.Get("grant_type"); got != "urn:ietf:params:oauth:grant-type:jwt-bearer" {
			t.Errorf("grant_type = %q", got)
		}
		claims := jwt.MapClaims{}
		_, err := jwt.ParseWithClaims(r.Form.Get("assertion"), claims,
			func(*jwt.Token) (any, error) { return &f.key.PublicKey, nil },
			jwt.WithValidMethods([]string{"RS256"}))
		if err != nil {
			t.Errorf("parse assertion: %v", err)
		}
		if claims["iss"] != saEmail || claims["scope"] != wantScope || claims["aud"] != f.srv.URL+"/token" {
			t.Errorf("assertion claims = %v", claims)
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"access_token":"at","token_type":"Bearer","expires_in":3600}`))
	})
	mux.HandleFunc(queuePath, func(w http.ResponseWriter, r *http.Request) { f.record(t, w, r) })
	mux.HandleFunc(queuePath+"/", func(w http.ResponseWriter, r *http.Request) { f.record(t, w, r) })
	f.srv = httptest.NewServer(mux)
	t.Cleanup(f.srv.Close)
	key, sa := serviceAccountJSON(t, f.srv.URL+"/token")
	f.key = key
	return f, sa
}

func (f *googleFake) record(t *testing.T, w http.ResponseWriter, r *http.Request) {
	c := call{method: r.Method, path: r.URL.Path, bearer: r.Header.Get("Authorization")}
	if r.Method == http.MethodPost {
		if err := json.NewDecoder(r.Body).Decode(&c.body); err != nil {
			t.Errorf("decode task body: %v", err)
		}
	}
	f.mu.Lock()
	f.calls = append(f.calls, c)
	status, body := f.status, f.body
	f.mu.Unlock()
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_, _ = w.Write([]byte(body))
}

func (f *googleFake) received() []call {
	f.mu.Lock()
	defer f.mu.Unlock()
	return append([]call(nil), f.calls...)
}

func (f *googleFake) tokenRequests() int {
	f.mu.Lock()
	defer f.mu.Unlock()
	return f.tokenHits
}

func newClient(t *testing.T, f *googleFake, sa []byte) *cloudtasks.Client {
	t.Helper()
	c, err := cloudtasks.NewClient(cloudtasks.Config{
		ProjectID: "p", Location: "asia-northeast1", Queue: "q",
		CallbackBaseURL: "https://runa.example/", CallbackToken: "secret",
		ServiceAccountJSON: sa, Endpoint: f.srv.URL,
	}, f.srv.Client())
	if err != nil {
		t.Fatalf("NewClient() error = %v", err)
	}
	return c
}

func TestClient_Schedule_RequestShape(t *testing.T) {
	t.Parallel()
	f, sa := newGoogleFake(t, http.StatusOK, `{"name":"projects/p/locations/asia-northeast1/queues/q/tasks/reminder-u1-2026-09-21"}`)

	err := newClient(t, f, sa).Schedule(t.Context(), schedule.Task{
		Name:  "reminder-u1-2026-09-21",
		RunAt: time.Date(2026, 9, 21, 21, 0, 0, 0, time.FixedZone("JST", 9*60*60)),
		Path:  "/api/v1/hooks/push/reminder",
		Body:  []byte(`{"user_id":"u1"}`),
	})
	if err != nil {
		t.Fatalf("Schedule() error = %v", err)
	}

	calls := f.received()
	if len(calls) != 1 {
		t.Fatalf("calls = %d, want 1", len(calls))
	}
	if calls[0].method != http.MethodPost || calls[0].path != queuePath {
		t.Errorf("request = %s %s, want POST %s", calls[0].method, calls[0].path, queuePath)
	}
	if calls[0].bearer != "Bearer at" {
		t.Errorf("Authorization = %q, want Bearer at", calls[0].bearer)
	}
	want := map[string]any{
		"task": map[string]any{
			"name":         "projects/p/locations/asia-northeast1/queues/q/tasks/reminder-u1-2026-09-21",
			"scheduleTime": "2026-09-21T12:00:00Z",
			"httpRequest": map[string]any{
				"url":        "https://runa.example/api/v1/hooks/push/reminder",
				"httpMethod": "POST",
				"headers": map[string]any{
					"Content-Type":          "application/json",
					"X-Push-Callback-Token": "secret",
				},
				"body": "eyJ1c2VyX2lkIjoidTEifQ==",
			},
		},
	}
	if diff := cmp.Diff(want, calls[0].body); diff != "" {
		t.Errorf("body mismatch (-want +got):\n%s", diff)
	}
}

func TestClient_Schedule_Status(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name    string
		status  int
		body    string
		wantErr string // empty means success
	}{
		{name: "200 is success", status: 200, body: `{"name":"x"}`},
		{name: "409 ALREADY_EXISTS is success", status: 409, body: `{"error":{"code":409,"message":"Requested entity already exists","status":"ALREADY_EXISTS"}}`},
		{name: "500 is an error with the message", status: 500, body: `{"error":{"code":500,"message":"backend blew up","status":"INTERNAL"}}`, wantErr: "backend blew up"},
		{name: "403 is an error", status: 403, body: `{"error":{"code":403,"message":"denied","status":"PERMISSION_DENIED"}}`, wantErr: "status 403"},
		{name: "unparseable failure still reports the status", status: 502, body: `<html>`, wantErr: "status 502"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			f, sa := newGoogleFake(t, tt.status, tt.body)

			err := newClient(t, f, sa).Schedule(t.Context(), schedule.Task{Name: "n", RunAt: time.Now(), Path: "/p"})
			if tt.wantErr == "" {
				if err != nil {
					t.Fatalf("Schedule() error = %v, want nil", err)
				}
				return
			}
			if err == nil || !strings.Contains(err.Error(), tt.wantErr) {
				t.Errorf("Schedule() error = %v, want it to mention %q", err, tt.wantErr)
			}
		})
	}
}

func TestClient_Schedule_RejectsBadNamesBeforeSending(t *testing.T) {
	t.Parallel()
	f, sa := newGoogleFake(t, http.StatusOK, `{}`)
	c := newClient(t, f, sa)

	for _, name := range []string{"a/b", "", "with space", "日本語", strings.Repeat("x", 501)} {
		if err := c.Schedule(t.Context(), schedule.Task{Name: name, RunAt: time.Now(), Path: "/p"}); err == nil {
			t.Errorf("Schedule(%q) error = nil, want a validation error", name)
		}
	}
	if got := f.received(); len(got) != 0 {
		t.Errorf("calls = %d, want 0 for invalid names", len(got))
	}
}

func TestClient_Cancel(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name    string
		status  int
		body    string
		wantErr bool
	}{
		{name: "200 is success", status: 200, body: `{}`},
		{name: "204 is success", status: 204},
		{name: "404 missing task is success", status: 404, body: `{"error":{"code":404,"message":"not found","status":"NOT_FOUND"}}`},
		{name: "500 is an error", status: 500, body: `{"error":{"code":500,"message":"boom","status":"INTERNAL"}}`, wantErr: true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			f, sa := newGoogleFake(t, tt.status, tt.body)

			err := newClient(t, f, sa).Cancel(t.Context(), "reminder-u1")
			if (err != nil) != tt.wantErr {
				t.Fatalf("Cancel() error = %v, wantErr %v", err, tt.wantErr)
			}
			calls := f.received()
			if len(calls) != 1 {
				t.Fatalf("calls = %d, want 1", len(calls))
			}
			if calls[0].method != http.MethodDelete || calls[0].path != queuePath+"/reminder-u1" {
				t.Errorf("request = %s %s, want DELETE %s/reminder-u1", calls[0].method, calls[0].path, queuePath)
			}
			if calls[0].bearer != "Bearer at" {
				t.Errorf("Authorization = %q, want Bearer at", calls[0].bearer)
			}
		})
	}
}

func TestClient_Cancel_RejectsBadNames(t *testing.T) {
	t.Parallel()
	f, sa := newGoogleFake(t, http.StatusOK, `{}`)

	if err := newClient(t, f, sa).Cancel(t.Context(), "a/b"); err == nil {
		t.Error("Cancel(\"a/b\") error = nil, want a validation error")
	}
	if got := f.received(); len(got) != 0 {
		t.Errorf("calls = %d, want 0", len(got))
	}
}

func TestClient_ReusesAccessToken(t *testing.T) {
	t.Parallel()
	f, sa := newGoogleFake(t, http.StatusOK, `{}`)
	c := newClient(t, f, sa)

	if err := c.Schedule(t.Context(), schedule.Task{Name: "n", RunAt: time.Now(), Path: "/p"}); err != nil {
		t.Fatalf("Schedule() error = %v", err)
	}
	if err := c.Cancel(t.Context(), "n"); err != nil {
		t.Fatalf("Cancel() error = %v", err)
	}
	if hits := f.tokenRequests(); hits != 1 {
		t.Errorf("token endpoint hits = %d, want 1", hits)
	}
}

func TestClient_TokenEndpointFailureIsAnError(t *testing.T) {
	t.Parallel()
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/token" {
			w.WriteHeader(http.StatusServiceUnavailable)
			return
		}
		t.Errorf("unexpected request %s %s", r.Method, r.URL)
	}))
	t.Cleanup(srv.Close)
	_, sa := serviceAccountJSON(t, srv.URL+"/token")
	c, err := cloudtasks.NewClient(cloudtasks.Config{
		ProjectID: "p", Location: "l", Queue: "q", CallbackBaseURL: "https://x", CallbackToken: "s",
		ServiceAccountJSON: sa, Endpoint: srv.URL,
	}, srv.Client())
	if err != nil {
		t.Fatalf("NewClient() error = %v", err)
	}

	if err := c.Schedule(t.Context(), schedule.Task{Name: "n", RunAt: time.Now(), Path: "/p"}); err == nil {
		t.Error("Schedule() error = nil, want an error when the token exchange fails")
	}
}

func TestNewClient_Validation(t *testing.T) {
	t.Parallel()
	_, sa := serviceAccountJSON(t, "https://oauth2.googleapis.com/token")
	valid := cloudtasks.Config{
		ProjectID: "p", Location: "l", Queue: "q", CallbackBaseURL: "https://x", CallbackToken: "s",
		ServiceAccountJSON: sa,
	}
	tests := []struct {
		name   string
		mutate func(*cloudtasks.Config)
	}{
		{name: "missing ProjectID", mutate: func(c *cloudtasks.Config) { c.ProjectID = "" }},
		{name: "missing Location", mutate: func(c *cloudtasks.Config) { c.Location = "" }},
		{name: "missing Queue", mutate: func(c *cloudtasks.Config) { c.Queue = "" }},
		{name: "missing CallbackBaseURL", mutate: func(c *cloudtasks.Config) { c.CallbackBaseURL = "" }},
		{name: "missing CallbackToken", mutate: func(c *cloudtasks.Config) { c.CallbackToken = "" }},
		{name: "missing key file", mutate: func(c *cloudtasks.Config) { c.ServiceAccountJSON = nil }},
		{name: "malformed key file", mutate: func(c *cloudtasks.Config) { c.ServiceAccountJSON = []byte("{") }},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			cfg := valid
			tt.mutate(&cfg)
			if _, err := cloudtasks.NewClient(cfg, nil); err == nil {
				t.Error("NewClient() error = nil, want an error")
			}
		})
	}
	if _, err := cloudtasks.NewClient(valid, nil); err != nil {
		t.Errorf("NewClient(valid) error = %v", err)
	}
}
