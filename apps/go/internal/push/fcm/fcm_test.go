package fcm_test

import (
	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"encoding/json"
	"encoding/pem"
	"errors"
	"net/http"
	"net/http/httptest"
	"strconv"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/0muji4/Runa/apps/go/internal/push"
	"github.com/0muji4/Runa/apps/go/internal/push/fcm"
	"github.com/golang-jwt/jwt/v5"
	"github.com/google/go-cmp/cmp"
)

const (
	saEmail   = "runa@p.iam.gserviceaccount.com"
	wantScope = "https://www.googleapis.com/auth/firebase.messaging"
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

// reply is one canned FCM answer.
type reply struct {
	status int
	body   string
}

// googleFake serves the token endpoint and messages:send from one server.
type googleFake struct {
	srv       *httptest.Server
	key       *rsa.PrivateKey
	mu        sync.Mutex
	reply     reply
	tokenHits int
	bearers   []string
	bodies    []map[string]any
}

func newGoogleFake(t *testing.T, rep reply) (*googleFake, []byte) {
	t.Helper()
	f := &googleFake{reply: rep}
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
		if claims["iss"] != saEmail {
			t.Errorf("assertion iss = %v, want %s", claims["iss"], saEmail)
		}
		if claims["scope"] != wantScope {
			t.Errorf("assertion scope = %v, want %s", claims["scope"], wantScope)
		}
		if claims["aud"] != f.srv.URL+"/token" {
			t.Errorf("assertion aud = %v, want %s", claims["aud"], f.srv.URL+"/token")
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"access_token":"at","token_type":"Bearer","expires_in":3600}`))
	})
	mux.HandleFunc("/v1/projects/p/messages:send", func(w http.ResponseWriter, r *http.Request) {
		var body map[string]any
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			t.Errorf("decode send body: %v", err)
		}
		f.mu.Lock()
		f.bearers = append(f.bearers, r.Header.Get("Authorization"))
		f.bodies = append(f.bodies, body)
		rep := f.reply
		f.mu.Unlock()
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(rep.status)
		_, _ = w.Write([]byte(rep.body))
	})
	f.srv = httptest.NewServer(mux)
	t.Cleanup(f.srv.Close)
	key, sa := serviceAccountJSON(t, f.srv.URL+"/token")
	f.key = key
	return f, sa
}

// sends returns the bearers and bodies of every messages:send call so far.
func (f *googleFake) sends() ([]string, []map[string]any) {
	f.mu.Lock()
	defer f.mu.Unlock()
	return append([]string(nil), f.bearers...), append([]map[string]any(nil), f.bodies...)
}

func (f *googleFake) tokenRequests() int {
	f.mu.Lock()
	defer f.mu.Unlock()
	return f.tokenHits
}

func newClient(t *testing.T, f *googleFake, sa []byte) *fcm.Client {
	t.Helper()
	c, err := fcm.NewClient(fcm.Config{ProjectID: "p", ServiceAccountJSON: sa, Endpoint: f.srv.URL}, f.srv.Client())
	if err != nil {
		t.Fatalf("NewClient() error = %v", err)
	}
	return c
}

const okBody = `{"name":"projects/p/messages/0:123"}`

func TestClient_Send_RequestShape(t *testing.T) {
	t.Parallel()
	f, sa := newGoogleFake(t, reply{http.StatusOK, okBody})
	c := newClient(t, f, sa)

	got, err := c.Send(t.Context(), "dev-token", push.Notification{
		Title:      "月夜のしおり",
		Body:       "今日の記録を残しましょう",
		Data:       map[string]string{"kind": "reminder", "date": "2026-09-21"},
		CollapseID: "reminder",
		Expiry:     time.Now().Add(1 * time.Hour),
	})
	if err != nil {
		t.Fatalf("Send() error = %v", err)
	}
	if got.ProviderMessageID != "projects/p/messages/0:123" {
		t.Errorf("ProviderMessageID = %q", got.ProviderMessageID)
	}
	bearers, bodies := f.sends()
	if len(bodies) != 1 {
		t.Fatalf("send requests = %d, want 1", len(bodies))
	}
	if bearers[0] != "Bearer at" {
		t.Errorf("Authorization = %q, want Bearer at", bearers[0])
	}

	body := bodies[0]
	android := body["message"].(map[string]any)["android"].(map[string]any)
	ttl, _ := android["ttl"].(string)
	// The clock moved between Send and the assertion, so accept a second of drift.
	if ttl != "3600s" && ttl != "3599s" {
		t.Errorf("ttl = %q, want about 3600s", ttl)
	}
	delete(android, "ttl")
	want := map[string]any{
		"message": map[string]any{
			"token": "dev-token",
			"data": map[string]any{
				"kind": "reminder", "date": "2026-09-21",
				"title": "月夜のしおり", "body": "今日の記録を残しましょう",
			},
			"android": map[string]any{"priority": "HIGH", "collapse_key": "reminder"},
		},
	}
	if diff := cmp.Diff(want, body); diff != "" {
		t.Errorf("body mismatch (-want +got):\n%s", diff)
	}
}

func TestClient_Send_OmitsTTLAndCollapseKeyWhenUnset(t *testing.T) {
	t.Parallel()
	f, sa := newGoogleFake(t, reply{http.StatusOK, okBody})

	if _, err := newClient(t, f, sa).Send(t.Context(), "tok", push.Notification{Title: "t", Body: "b"}); err != nil {
		t.Fatalf("Send() error = %v", err)
	}
	want := map[string]any{
		"message": map[string]any{
			"token":   "tok",
			"data":    map[string]any{"title": "t", "body": "b"},
			"android": map[string]any{"priority": "HIGH"},
		},
	}
	_, bodies := f.sends()
	if diff := cmp.Diff(want, bodies[0]); diff != "" {
		t.Errorf("body mismatch (-want +got):\n%s", diff)
	}
}

func TestClient_Send_PastExpiryClampsTTLToZero(t *testing.T) {
	t.Parallel()
	f, sa := newGoogleFake(t, reply{http.StatusOK, okBody})

	n := push.Notification{Title: "t", Expiry: time.Now().Add(-time.Hour)}
	if _, err := newClient(t, f, sa).Send(t.Context(), "tok", n); err != nil {
		t.Fatalf("Send() error = %v", err)
	}
	_, bodies := f.sends()
	android := bodies[0]["message"].(map[string]any)["android"].(map[string]any)
	if android["ttl"] != "0s" {
		t.Errorf("ttl = %v, want 0s", android["ttl"])
	}
}

func fcmError(code int, status, errorCode, msg string) string {
	details := ""
	if errorCode != "" {
		details = `,"details":[{"@type":"type.googleapis.com/google.firebase.fcm.v1.FcmError","errorCode":"` + errorCode + `"}]`
	}
	return `{"error":{"code":` + strconv.Itoa(code) + `,"message":"` + msg + `","status":"` + status + `"` + details + `}}`
}

func TestClient_Send_ErrorMapping(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name     string
		reply    reply
		wantIs   error // nil means a plain error
		wantText string
	}{
		{
			name:   "UNREGISTERED is invalid token",
			reply:  reply{404, fcmError(404, "NOT_FOUND", "UNREGISTERED", "Requested entity was not found.")},
			wantIs: push.ErrInvalidToken,
		},
		{
			name:   "INVALID_ARGUMENT mentioning the token is invalid token",
			reply:  reply{400, fcmError(400, "INVALID_ARGUMENT", "INVALID_ARGUMENT", "The registration token is not a valid FCM registration token")},
			wantIs: push.ErrInvalidToken,
		},
		{
			name:     "INVALID_ARGUMENT about something else is a plain error",
			reply:    reply{400, fcmError(400, "INVALID_ARGUMENT", "INVALID_ARGUMENT", "Invalid value at message.android.ttl")},
			wantText: "INVALID_ARGUMENT",
		},
		{
			name:   "429 QUOTA_EXCEEDED is transient",
			reply:  reply{429, fcmError(429, "RESOURCE_EXHAUSTED", "QUOTA_EXCEEDED", "Sending limit exceeded")},
			wantIs: push.ErrTransient,
		},
		{
			name:   "503 UNAVAILABLE is transient",
			reply:  reply{503, fcmError(503, "UNAVAILABLE", "UNAVAILABLE", "Try again later")},
			wantIs: push.ErrTransient,
		},
		{
			name:   "500 INTERNAL is transient",
			reply:  reply{500, fcmError(500, "INTERNAL", "INTERNAL", "Internal error")},
			wantIs: push.ErrTransient,
		},
		{
			name:   "5xx without a JSON body is transient",
			reply:  reply{502, "<html>bad gateway</html>"},
			wantIs: push.ErrTransient,
		},
		{
			name:     "SENDER_ID_MISMATCH is a plain error",
			reply:    reply{403, fcmError(403, "PERMISSION_DENIED", "SENDER_ID_MISMATCH", "SenderId mismatch")},
			wantText: "SENDER_ID_MISMATCH",
		},
		{
			name:     "THIRD_PARTY_AUTH_ERROR is a plain error",
			reply:    reply{401, fcmError(401, "UNAUTHENTICATED", "THIRD_PARTY_AUTH_ERROR", "Auth error from APNS")},
			wantText: "THIRD_PARTY_AUTH_ERROR",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			f, sa := newGoogleFake(t, tt.reply)

			_, err := newClient(t, f, sa).Send(t.Context(), "tok", push.Notification{Title: "t"})
			if err == nil {
				t.Fatal("Send() error = nil, want an error")
			}
			if tt.wantIs != nil && !errors.Is(err, tt.wantIs) {
				t.Errorf("Send() error = %v, want errors.Is %v", err, tt.wantIs)
			}
			if tt.wantIs == nil && (errors.Is(err, push.ErrInvalidToken) || errors.Is(err, push.ErrTransient)) {
				t.Errorf("Send() error = %v, want a plain error", err)
			}
			if !strings.Contains(err.Error(), tt.wantText) {
				t.Errorf("Send() error = %v, want it to mention %q", err, tt.wantText)
			}
		})
	}
}

func TestClient_Send_TokenEndpointFailureIsTransient(t *testing.T) {
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
	c, err := fcm.NewClient(fcm.Config{ProjectID: "p", ServiceAccountJSON: sa, Endpoint: srv.URL}, srv.Client())
	if err != nil {
		t.Fatalf("NewClient() error = %v", err)
	}

	_, err = c.Send(t.Context(), "tok", push.Notification{Title: "t"})
	if !errors.Is(err, push.ErrTransient) {
		t.Errorf("Send() error = %v, want errors.Is ErrTransient", err)
	}
}

func TestClient_Send_ReusesAccessToken(t *testing.T) {
	t.Parallel()
	f, sa := newGoogleFake(t, reply{http.StatusOK, okBody})
	c := newClient(t, f, sa)

	for i := 0; i < 2; i++ {
		if _, err := c.Send(t.Context(), "tok", push.Notification{Title: "t"}); err != nil {
			t.Fatalf("Send() #%d error = %v", i+1, err)
		}
	}
	if hits := f.tokenRequests(); hits != 1 {
		t.Errorf("token endpoint hits = %d, want 1", hits)
	}
	if bearers, _ := f.sends(); len(bearers) != 2 || bearers[0] != bearers[1] {
		t.Errorf("bearers = %q, want the same one twice", bearers)
	}
}

func TestClient_Send_TokenURLOverridesKeyFile(t *testing.T) {
	t.Parallel()
	f, _ := newGoogleFake(t, reply{http.StatusOK, okBody})
	_, sa := serviceAccountJSON(t, "https://oauth2.googleapis.com/token")
	// The fake checks aud against its own URL, so the override must win over token_uri.
	c, err := fcm.NewClient(fcm.Config{
		ProjectID: "p", ServiceAccountJSON: sa, Endpoint: f.srv.URL, TokenURL: f.srv.URL + "/token",
	}, f.srv.Client())
	if err != nil {
		t.Fatalf("NewClient() error = %v", err)
	}
	f.key = mustParseKey(t, sa)

	if _, err := c.Send(t.Context(), "tok", push.Notification{Title: "t"}); err != nil {
		t.Fatalf("Send() error = %v", err)
	}
}

func mustParseKey(t *testing.T, sa []byte) *rsa.PrivateKey {
	t.Helper()
	var doc struct {
		PrivateKey string `json:"private_key"`
	}
	if err := json.Unmarshal(sa, &doc); err != nil {
		t.Fatal(err)
	}
	key, err := jwt.ParseRSAPrivateKeyFromPEM([]byte(doc.PrivateKey))
	if err != nil {
		t.Fatal(err)
	}
	return key
}

func TestNewClient_Validation(t *testing.T) {
	t.Parallel()
	_, sa := serviceAccountJSON(t, "https://oauth2.googleapis.com/token")
	_, saNoTokenURI := serviceAccountJSON(t, "")
	tests := []struct {
		name    string
		cfg     fcm.Config
		wantErr bool
	}{
		{name: "valid", cfg: fcm.Config{ProjectID: "p", ServiceAccountJSON: sa}},
		{name: "TokenURL fills a missing token_uri", cfg: fcm.Config{ProjectID: "p", ServiceAccountJSON: saNoTokenURI, TokenURL: "https://x/token"}},
		{name: "missing ProjectID", cfg: fcm.Config{ServiceAccountJSON: sa}, wantErr: true},
		{name: "missing key file", cfg: fcm.Config{ProjectID: "p"}, wantErr: true},
		{name: "malformed key file", cfg: fcm.Config{ProjectID: "p", ServiceAccountJSON: []byte("{")}, wantErr: true},
		{name: "key file without private_key", cfg: fcm.Config{ProjectID: "p", ServiceAccountJSON: []byte(`{"client_email":"a@b"}`)}, wantErr: true},
		{name: "no token URL anywhere", cfg: fcm.Config{ProjectID: "p", ServiceAccountJSON: saNoTokenURI}, wantErr: true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			_, err := fcm.NewClient(tt.cfg, nil)
			if (err != nil) != tt.wantErr {
				t.Errorf("NewClient() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}
