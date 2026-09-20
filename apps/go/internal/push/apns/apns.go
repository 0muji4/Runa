// Package apns delivers push.Notification to iOS devices through Apple's
// HTTP/2 provider API using token-based (JWT) authentication.
package apns

import (
	"bytes"
	"context"
	"crypto/ecdsa"
	"crypto/rand"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/0muji4/Runa/apps/go/internal/push"
	"github.com/golang-jwt/jwt/v5"
)

// Apple's two provider hosts; a token is valid on only the one matching the app's signing.
const (
	SandboxHost    = "https://api.sandbox.push.apple.com"
	ProductionHost = "https://api.push.apple.com"
)

// Apple wants the provider token refreshed at least hourly but not more often
// than every 20 minutes; minRefreshInterval guards only the 403-driven refresh.
const (
	tokenTTL           = 50 * time.Minute
	minRefreshInterval = 20 * time.Minute
)

// maxResponseBytes caps the error body read from Apple (real bodies are one short JSON object).
const maxResponseBytes = 64 << 10

// Config is what NewClient needs; every field is required.
type Config struct {
	TeamID string
	KeyID  string
	// PrivateKeyPEM is the PKCS#8 EC P-256 key from Apple's AuthKey_XXXX.p8.
	PrivateKeyPEM []byte
	// BundleID is sent as apns-topic.
	BundleID string
	// Host is SandboxHost or ProductionHost; tests point it at a fake.
	Host string
}

// HostFor maps the APNS_ENVIRONMENT setting to a provider host.
func HostFor(environment string) (string, error) {
	switch environment {
	case "sandbox":
		return SandboxHost, nil
	case "production":
		return ProductionHost, nil
	default:
		return "", fmt.Errorf("apns: unknown environment %q (want sandbox or production)", environment)
	}
}

// Client is a push.Sender for APNs. It is safe for concurrent use.
type Client struct {
	client *http.Client
	host   string
	teamID string
	keyID  string
	topic  string
	key    *ecdsa.PrivateKey
	now    func() time.Time

	mu          sync.Mutex
	token       string
	tokenMinted time.Time
	// tokenForced marks a token minted by a 403 retry, so a second 403 within
	// minRefreshInterval reuses it instead of minting again.
	tokenForced bool
}

var _ push.Sender = (*Client)(nil)

// NewClient parses the signing key and validates cfg; a nil client uses
// http.DefaultClient (HTTP/2 over TLS, as APNs requires).
func NewClient(cfg Config, client *http.Client) (*Client, error) {
	switch {
	case cfg.TeamID == "":
		return nil, errors.New("apns: TeamID is required")
	case cfg.KeyID == "":
		return nil, errors.New("apns: KeyID is required")
	case cfg.BundleID == "":
		return nil, errors.New("apns: BundleID is required")
	case cfg.Host == "":
		return nil, errors.New("apns: Host is required")
	case len(cfg.PrivateKeyPEM) == 0:
		return nil, errors.New("apns: PrivateKeyPEM is required")
	}
	key, err := jwt.ParseECPrivateKeyFromPEM(cfg.PrivateKeyPEM)
	if err != nil {
		return nil, fmt.Errorf("apns: parse private key: %w", err)
	}
	if client == nil {
		client = http.DefaultClient
	}
	return &Client{
		client: client,
		host:   strings.TrimRight(cfg.Host, "/"),
		teamID: cfg.TeamID,
		keyID:  cfg.KeyID,
		topic:  cfg.BundleID,
		key:    key,
		now:    time.Now,
	}, nil
}

// Send posts n to token; a 403 for a bad provider token forces one refresh and one retry.
func (c *Client) Send(ctx context.Context, token string, n push.Notification) (push.Result, error) {
	body, err := payload(n)
	if err != nil {
		return push.Result{}, err
	}
	bearer, err := c.providerToken(false)
	if err != nil {
		return push.Result{}, err
	}
	res, err := c.post(ctx, token, n, body, bearer)
	if err != nil {
		return push.Result{}, err
	}
	if res.status == http.StatusForbidden && (res.reason == "ExpiredProviderToken" || res.reason == "InvalidProviderToken") {
		if bearer, err = c.providerToken(true); err != nil {
			return push.Result{}, err
		}
		if res, err = c.post(ctx, token, n, body, bearer); err != nil {
			return push.Result{}, err
		}
	}
	return res.result()
}

// response is what Send needs from one APNs reply.
type response struct {
	status int
	apnsID string
	reason string
}

// result maps Apple's status and reason to the push seam's error classes.
func (r response) result() (push.Result, error) {
	if r.status == http.StatusOK {
		return push.Result{ProviderMessageID: r.apnsID}, nil
	}
	switch r.reason {
	case "BadDeviceToken", "Unregistered", "ExpiredToken", "DeviceTokenNotForTopic", "TopicDisallowed":
		return push.Result{}, fmt.Errorf("apns: %s (status %d): %w", r.reason, r.status, push.ErrInvalidToken)
	case "Shutdown", "ServiceUnavailable", "InternalServerError", "TooManyRequests":
		return push.Result{}, fmt.Errorf("apns: %s (status %d): %w", r.reason, r.status, push.ErrTransient)
	}
	if r.status == http.StatusTooManyRequests || r.status >= 500 {
		return push.Result{}, fmt.Errorf("apns: %s (status %d): %w", r.reason, r.status, push.ErrTransient)
	}
	return push.Result{}, fmt.Errorf("apns: send rejected: status %d reason %q", r.status, r.reason)
}

// post performs one request; transport failures are transient from the caller's view.
func (c *Client) post(ctx context.Context, token string, n push.Notification, body []byte, bearer string) (response, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, c.host+"/3/device/"+token, bytes.NewReader(body))
	if err != nil {
		return response{}, fmt.Errorf("apns: build request: %w", err)
	}
	apnsID := newID()
	req.Header.Set("authorization", "bearer "+bearer)
	req.Header.Set("apns-topic", c.topic)
	req.Header.Set("apns-push-type", "alert")
	req.Header.Set("apns-priority", "10")
	req.Header.Set("apns-id", apnsID)
	if n.CollapseID != "" {
		req.Header.Set("apns-collapse-id", n.CollapseID)
	}
	if !n.Expiry.IsZero() {
		req.Header.Set("apns-expiration", strconv.FormatInt(n.Expiry.Unix(), 10))
	}
	res, err := c.client.Do(req)
	if err != nil {
		// *url.Error embeds the URL, and the URL embeds the device token: report only the cause.
		var ue *url.Error
		if errors.As(err, &ue) {
			err = ue.Err
		}
		return response{}, fmt.Errorf("apns: send: %v: %w", err, push.ErrTransient)
	}
	defer res.Body.Close()

	out := response{status: res.StatusCode, apnsID: res.Header.Get("apns-id")}
	if out.apnsID == "" {
		out.apnsID = apnsID
	}
	if res.StatusCode != http.StatusOK {
		var errBody struct {
			Reason string `json:"reason"`
		}
		// The reason is advisory; an unparseable body still maps by status.
		_ = json.NewDecoder(io.LimitReader(res.Body, maxResponseBytes)).Decode(&errBody)
		out.reason = errBody.Reason
	}
	return out, nil
}

// payload builds the APNs JSON; a data key named "aps" is dropped rather than clobbering the alert.
func payload(n push.Notification) ([]byte, error) {
	doc := make(map[string]any, len(n.Data)+1)
	for k, v := range n.Data {
		if k == "aps" {
			continue
		}
		doc[k] = v
	}
	doc["aps"] = map[string]any{
		"alert": map[string]string{"title": n.Title, "body": n.Body},
		"sound": "default",
	}
	b, err := json.Marshal(doc)
	if err != nil {
		return nil, fmt.Errorf("apns: encode payload: %w", err)
	}
	return b, nil
}

// providerToken returns the cached JWT, reminting past tokenTTL; force (after a
// 403) remints early once, then waits minRefreshInterval.
func (c *Client) providerToken(force bool) (string, error) {
	c.mu.Lock()
	defer c.mu.Unlock()
	now := c.now()
	age := now.Sub(c.tokenMinted)
	expired := c.token == "" || age >= tokenTTL
	if !expired && !(force && (!c.tokenForced || age >= minRefreshInterval)) {
		return c.token, nil
	}
	tok := jwt.NewWithClaims(jwt.SigningMethodES256, jwt.MapClaims{
		"iss": c.teamID,
		"iat": now.Unix(),
	})
	tok.Header["kid"] = c.keyID
	signed, err := tok.SignedString(c.key)
	if err != nil {
		return "", fmt.Errorf("apns: sign provider token: %w", err)
	}
	c.token, c.tokenMinted, c.tokenForced = signed, now, !expired
	return signed, nil
}

// newID returns a random v4-style UUID for the apns-id header.
func newID() string {
	var b [16]byte
	if _, err := rand.Read(b[:]); err != nil {
		panic(err)
	}
	b[6] = (b[6] & 0x0f) | 0x40
	b[8] = (b[8] & 0x3f) | 0x80
	return fmt.Sprintf("%x-%x-%x-%x-%x", b[0:4], b[4:6], b[6:8], b[8:10], b[10:16])
}
