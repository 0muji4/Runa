// Package fcm delivers push.Notification to Android devices through the
// Firebase Cloud Messaging HTTP v1 API, authenticated as a GCP service account.
package fcm

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/0muji4/Runa/apps/go/internal/push"
	"golang.org/x/oauth2"
	"golang.org/x/oauth2/jwt"
)

// DefaultEndpoint is Google's FCM host.
const DefaultEndpoint = "https://fcm.googleapis.com"

// scope is the only OAuth scope the v1 send endpoint needs.
const scope = "https://www.googleapis.com/auth/firebase.messaging"

// maxResponseBytes caps what Send reads from FCM (real bodies are one short JSON object).
const maxResponseBytes = 64 << 10

// Config is what NewClient needs; Endpoint and TokenURL default to Google's and are overridden by tests.
type Config struct {
	ProjectID string
	// ServiceAccountJSON is the key file downloaded from the GCP console.
	ServiceAccountJSON []byte
	Endpoint           string
	// TokenURL overrides the token_uri inside ServiceAccountJSON.
	TokenURL string
}

// serviceAccount mirrors the fields read from the key file.
type serviceAccount struct {
	ClientEmail string `json:"client_email"`
	PrivateKey  string `json:"private_key"`
	TokenURI    string `json:"token_uri"`
}

// Client is a push.Sender for FCM. It is safe for concurrent use.
type Client struct {
	client  *http.Client
	sendURL string
	tokens  oauth2.TokenSource
	now     func() time.Time
}

var _ push.Sender = (*Client)(nil)

// NewClient validates cfg and prepares the token source; the injected client also carries the token exchange.
func NewClient(cfg Config, client *http.Client) (*Client, error) {
	if cfg.ProjectID == "" {
		return nil, errors.New("fcm: ProjectID is required")
	}
	if len(cfg.ServiceAccountJSON) == 0 {
		return nil, errors.New("fcm: ServiceAccountJSON is required")
	}
	var sa serviceAccount
	if err := json.Unmarshal(cfg.ServiceAccountJSON, &sa); err != nil {
		return nil, fmt.Errorf("fcm: parse service account: %w", err)
	}
	if sa.ClientEmail == "" || sa.PrivateKey == "" {
		return nil, errors.New("fcm: service account needs client_email and private_key")
	}
	tokenURL := cfg.TokenURL
	if tokenURL == "" {
		tokenURL = sa.TokenURI
	}
	if tokenURL == "" {
		return nil, errors.New("fcm: service account has no token_uri and TokenURL is empty")
	}
	if client == nil {
		client = http.DefaultClient
	}
	endpoint := strings.TrimRight(cfg.Endpoint, "/")
	if endpoint == "" {
		endpoint = DefaultEndpoint
	}
	return &Client{
		client:  client,
		sendURL: endpoint + "/v1/projects/" + cfg.ProjectID + "/messages:send",
		tokens:  tokenSource(sa, tokenURL, client),
		now:     time.Now,
	}, nil
}

// tokenSource exchanges a signed service-account JWT for a cached access token.
func tokenSource(sa serviceAccount, tokenURL string, client *http.Client) oauth2.TokenSource {
	jc := &jwt.Config{
		Email:      sa.ClientEmail,
		PrivateKey: []byte(sa.PrivateKey),
		Scopes:     []string{scope},
		TokenURL:   tokenURL,
	}
	ctx := context.WithValue(context.Background(), oauth2.HTTPClient, client)
	return oauth2.ReuseTokenSource(nil, jc.TokenSource(ctx))
}

// message is the FCM v1 request body. Title and body ride in data, not in a
// notification block: the Android app composes the notification itself.
type message struct {
	Message struct {
		Token   string            `json:"token"`
		Data    map[string]string `json:"data"`
		Android androidConfig     `json:"android"`
	} `json:"message"`
}

type androidConfig struct {
	Priority    string `json:"priority"`
	TTL         string `json:"ttl,omitempty"`
	CollapseKey string `json:"collapse_key,omitempty"`
}

// errorResponse mirrors the google.rpc.Status FCM returns on failure.
type errorResponse struct {
	Error struct {
		Code    int    `json:"code"`
		Message string `json:"message"`
		Status  string `json:"status"`
		Details []struct {
			Type      string `json:"@type"`
			ErrorCode string `json:"errorCode"`
		} `json:"details"`
	} `json:"error"`
}

// Send posts a data-only, high-priority message to token.
func (c *Client) Send(ctx context.Context, token string, n push.Notification) (push.Result, error) {
	body, err := json.Marshal(c.message(token, n))
	if err != nil {
		return push.Result{}, fmt.Errorf("fcm: encode message: %w", err)
	}
	// A token-endpoint failure is a provider outage from the caller's view, not a bad device.
	access, err := c.tokens.Token()
	if err != nil {
		return push.Result{}, fmt.Errorf("fcm: access token: %v: %w", err, push.ErrTransient)
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, c.sendURL, bytes.NewReader(body))
	if err != nil {
		return push.Result{}, fmt.Errorf("fcm: build request: %w", err)
	}
	req.Header.Set("Authorization", "Bearer "+access.AccessToken)
	req.Header.Set("Content-Type", "application/json")
	res, err := c.client.Do(req)
	if err != nil {
		return push.Result{}, fmt.Errorf("fcm: send: %v: %w", err, push.ErrTransient)
	}
	defer res.Body.Close()
	raw, err := io.ReadAll(io.LimitReader(res.Body, maxResponseBytes))
	if err != nil {
		return push.Result{}, fmt.Errorf("fcm: read response: %v: %w", err, push.ErrTransient)
	}

	if res.StatusCode == http.StatusOK {
		var ok struct {
			Name string `json:"name"`
		}
		if err := json.Unmarshal(raw, &ok); err != nil {
			return push.Result{}, fmt.Errorf("fcm: decode response: %w", err)
		}
		return push.Result{ProviderMessageID: ok.Name}, nil
	}
	return push.Result{}, mapError(res.StatusCode, raw)
}

// message assembles the request body; ttl is the time left until Expiry.
func (c *Client) message(token string, n push.Notification) message {
	var m message
	m.Message.Token = token
	m.Message.Data = make(map[string]string, len(n.Data)+2)
	for k, v := range n.Data {
		m.Message.Data[k] = v
	}
	m.Message.Data["title"] = n.Title
	m.Message.Data["body"] = n.Body
	m.Message.Android = androidConfig{Priority: "HIGH", CollapseKey: n.CollapseID}
	if !n.Expiry.IsZero() {
		ttl := max(0, int64(n.Expiry.Sub(c.now()).Seconds()))
		m.Message.Android.TTL = strconv.FormatInt(ttl, 10) + "s"
	}
	return m
}

// mapError classifies a non-200 reply into the push seam's error classes.
func mapError(status int, raw []byte) error {
	var er errorResponse
	// The body is advisory; an unparseable one still maps by status.
	_ = json.Unmarshal(raw, &er)
	code := er.Error.Status
	for _, d := range er.Error.Details {
		if d.ErrorCode != "" {
			code = d.ErrorCode
		}
	}
	msg := er.Error.Message

	switch code {
	case "UNREGISTERED":
		return fmt.Errorf("fcm: %s (status %d): %s: %w", code, status, msg, push.ErrInvalidToken)
	case "INVALID_ARGUMENT":
		if strings.Contains(strings.ToLower(msg), "token") {
			return fmt.Errorf("fcm: %s (status %d): %s: %w", code, status, msg, push.ErrInvalidToken)
		}
	case "UNAVAILABLE", "INTERNAL", "QUOTA_EXCEEDED":
		return fmt.Errorf("fcm: %s (status %d): %s: %w", code, status, msg, push.ErrTransient)
	}
	if status == http.StatusTooManyRequests || status >= 500 {
		return fmt.Errorf("fcm: %s (status %d): %s: %w", code, status, msg, push.ErrTransient)
	}
	return fmt.Errorf("fcm: send rejected: status %d code %q: %s", status, code, msg)
}
