// Package cloudtasks implements schedule.Scheduler on Google Cloud Tasks: each
// Task becomes an HTTP task that calls back into this server at RunAt.
package cloudtasks

import (
	"bytes"
	"context"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"regexp"
	"strings"
	"time"

	"github.com/0muji4/Runa/apps/go/internal/schedule"
	"golang.org/x/oauth2"
	"golang.org/x/oauth2/jwt"
)

// DefaultEndpoint is Google's Cloud Tasks host.
const DefaultEndpoint = "https://cloudtasks.googleapis.com"

// CallbackTokenHeader carries CallbackToken on every callback.
const CallbackTokenHeader = "X-Push-Callback-Token"

// scope is the OAuth scope Cloud Tasks accepts; it has no narrower one.
const scope = "https://www.googleapis.com/auth/cloud-platform"

// maxResponseBytes caps what is read from Cloud Tasks (real bodies are one short JSON object).
const maxResponseBytes = 64 << 10

// taskNameRE is Cloud Tasks' allowed charset for the task id segment.
var taskNameRE = regexp.MustCompile(`^[A-Za-z0-9_-]{1,500}$`)

// Config is what NewClient needs; Endpoint and TokenURL default to Google's and are overridden by tests.
type Config struct {
	ProjectID string
	Location  string
	Queue     string
	// CallbackBaseURL is this server's public origin without a trailing slash; Task.Path is appended.
	CallbackBaseURL string
	// CallbackToken is sent as CallbackTokenHeader on every callback.
	CallbackToken string
	// ServiceAccountJSON is the key file downloaded from the GCP console.
	ServiceAccountJSON []byte
	Endpoint           string
	TokenURL           string
}

// serviceAccount mirrors the fields read from the key file.
type serviceAccount struct {
	ClientEmail string `json:"client_email"`
	PrivateKey  string `json:"private_key"`
	TokenURI    string `json:"token_uri"`
}

// Client is a schedule.Scheduler on Cloud Tasks. It is safe for concurrent use.
type Client struct {
	client        *http.Client
	queueURL      string // {endpoint}/v2/{queuePath}
	queuePath     string // projects/p/locations/l/queues/q
	callbackBase  string
	callbackToken string
	tokens        oauth2.TokenSource
}

var _ schedule.Scheduler = (*Client)(nil)

// NewClient validates cfg and prepares the token source; the injected client also carries the token exchange.
func NewClient(cfg Config, client *http.Client) (*Client, error) {
	switch {
	case cfg.ProjectID == "":
		return nil, errors.New("cloudtasks: ProjectID is required")
	case cfg.Location == "":
		return nil, errors.New("cloudtasks: Location is required")
	case cfg.Queue == "":
		return nil, errors.New("cloudtasks: Queue is required")
	case cfg.CallbackBaseURL == "":
		return nil, errors.New("cloudtasks: CallbackBaseURL is required")
	case cfg.CallbackToken == "":
		return nil, errors.New("cloudtasks: CallbackToken is required")
	case len(cfg.ServiceAccountJSON) == 0:
		return nil, errors.New("cloudtasks: ServiceAccountJSON is required")
	}
	var sa serviceAccount
	if err := json.Unmarshal(cfg.ServiceAccountJSON, &sa); err != nil {
		return nil, fmt.Errorf("cloudtasks: parse service account: %w", err)
	}
	if sa.ClientEmail == "" || sa.PrivateKey == "" {
		return nil, errors.New("cloudtasks: service account needs client_email and private_key")
	}
	tokenURL := cfg.TokenURL
	if tokenURL == "" {
		tokenURL = sa.TokenURI
	}
	if tokenURL == "" {
		return nil, errors.New("cloudtasks: service account has no token_uri and TokenURL is empty")
	}
	if client == nil {
		client = http.DefaultClient
	}
	endpoint := strings.TrimRight(cfg.Endpoint, "/")
	if endpoint == "" {
		endpoint = DefaultEndpoint
	}
	queuePath := "projects/" + cfg.ProjectID + "/locations/" + cfg.Location + "/queues/" + cfg.Queue
	return &Client{
		client:        client,
		queueURL:      endpoint + "/v2/" + queuePath,
		queuePath:     queuePath,
		callbackBase:  strings.TrimRight(cfg.CallbackBaseURL, "/"),
		callbackToken: cfg.CallbackToken,
		tokens:        tokenSource(sa, tokenURL, client),
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

// createTaskRequest is the Cloud Tasks v2 tasks.create body.
type createTaskRequest struct {
	Task struct {
		Name         string      `json:"name"`
		ScheduleTime string      `json:"scheduleTime"`
		HTTPRequest  httpRequest `json:"httpRequest"`
	} `json:"task"`
}

type httpRequest struct {
	URL        string            `json:"url"`
	HTTPMethod string            `json:"httpMethod"`
	Headers    map[string]string `json:"headers"`
	Body       string            `json:"body"`
}

// Schedule creates the task; 409 (name exists, also for a while after it ran) is success.
func (c *Client) Schedule(ctx context.Context, t schedule.Task) error {
	if !taskNameRE.MatchString(t.Name) {
		return fmt.Errorf("cloudtasks: task name %q must match %s", t.Name, taskNameRE)
	}
	var body createTaskRequest
	body.Task.Name = c.queuePath + "/tasks/" + t.Name
	body.Task.ScheduleTime = t.RunAt.UTC().Format(time.RFC3339)
	body.Task.HTTPRequest = httpRequest{
		URL:        c.callbackBase + t.Path,
		HTTPMethod: http.MethodPost,
		Headers: map[string]string{
			"Content-Type":      "application/json",
			CallbackTokenHeader: c.callbackToken,
		},
		Body: base64.StdEncoding.EncodeToString(t.Body),
	}
	raw, err := json.Marshal(body)
	if err != nil {
		return fmt.Errorf("cloudtasks: encode task: %w", err)
	}
	status, msg, err := c.do(ctx, http.MethodPost, c.queueURL+"/tasks", raw)
	if err != nil {
		return err
	}
	switch status {
	case http.StatusOK, http.StatusConflict:
		return nil
	}
	return fmt.Errorf("cloudtasks: create task %s: status %d: %s", t.Name, status, msg)
}

// Cancel deletes the task; a missing one (already ran or never existed) is success.
func (c *Client) Cancel(ctx context.Context, name string) error {
	if !taskNameRE.MatchString(name) {
		return fmt.Errorf("cloudtasks: task name %q must match %s", name, taskNameRE)
	}
	status, msg, err := c.do(ctx, http.MethodDelete, c.queueURL+"/tasks/"+name, nil)
	if err != nil {
		return err
	}
	switch status {
	case http.StatusOK, http.StatusNoContent, http.StatusNotFound:
		return nil
	}
	return fmt.Errorf("cloudtasks: delete task %s: status %d: %s", name, status, msg)
}

// do sends one authenticated request and returns the status plus any error.message.
func (c *Client) do(ctx context.Context, method, url string, body []byte) (int, string, error) {
	access, err := c.tokens.Token()
	if err != nil {
		return 0, "", fmt.Errorf("cloudtasks: access token: %w", err)
	}
	var rd io.Reader
	if body != nil {
		rd = bytes.NewReader(body)
	}
	req, err := http.NewRequestWithContext(ctx, method, url, rd)
	if err != nil {
		return 0, "", fmt.Errorf("cloudtasks: build request: %w", err)
	}
	req.Header.Set("Authorization", "Bearer "+access.AccessToken)
	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}
	res, err := c.client.Do(req)
	if err != nil {
		return 0, "", fmt.Errorf("cloudtasks: %s: %w", method, err)
	}
	defer res.Body.Close()
	raw, err := io.ReadAll(io.LimitReader(res.Body, maxResponseBytes))
	if err != nil {
		return 0, "", fmt.Errorf("cloudtasks: read response: %w", err)
	}
	var failure struct {
		Error struct {
			Message string `json:"message"`
		} `json:"error"`
	}
	// The message is advisory; an unparseable body still reports the status.
	_ = json.Unmarshal(raw, &failure)
	return res.StatusCode, failure.Error.Message, nil
}
