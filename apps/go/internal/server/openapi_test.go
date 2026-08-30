package server_test

import (
	"bytes"
	"context"
	"io"
	"net/http"
	"strings"
	"sync"
	"testing"

	"github.com/getkin/kin-openapi/openapi3"
	"github.com/getkin/kin-openapi/openapi3filter"
	"github.com/getkin/kin-openapi/routers"
	"github.com/getkin/kin-openapi/routers/gorillamux"
)

// Validates every exchange the flow tests make against api/openapi.yaml, the
// hand-maintained contract the Android and iOS clients are written against. An
// undocumented route, an unlisted status code or an off-schema body fails here.

var (
	specOnce   sync.Once
	specRouter routers.Router
	specErr    error
)

// loadSpec parses and validates the contract once per test binary.
func loadSpec() (routers.Router, error) {
	specOnce.Do(func() {
		loader := &openapi3.Loader{IsExternalRefsAllowed: true}
		doc, err := loader.LoadFromFile("../../api/openapi.yaml")
		if err != nil {
			specErr = err
			return
		}
		if err := doc.Validate(context.Background()); err != nil {
			specErr = err
			return
		}
		// The flow tests address the handler directly, so match on path alone
		// rather than the documented dev server hosts.
		doc.Servers = openapi3.Servers{{URL: "/"}}
		specRouter, specErr = gorillamux.NewRouter(doc)
	})
	return specRouter, specErr
}

// checkAgainstSpec validates one exchange. The bodies are passed in because both
// have been consumed by the time the caller gets here.
func checkAgainstSpec(t *testing.T, req *http.Request, res *http.Response, reqBody, resBody []byte) {
	t.Helper()

	router, err := loadSpec()
	if err != nil {
		t.Fatalf("loading api/openapi.yaml: %v", err)
	}

	// The validator reads the body, so give it its own copy.
	specReq := req.Clone(req.Context())
	specReq.Body = io.NopCloser(bytes.NewReader(reqBody))
	// The flow helper posts JSON without always setting the header.
	if len(reqBody) > 0 && specReq.Header.Get("Content-Type") == "" {
		specReq.Header.Set("Content-Type", "application/json")
	}

	route, pathParams, err := router.FindRoute(specReq)
	if err != nil {
		t.Errorf("%s %s is not described in api/openapi.yaml: %v",
			req.Method, req.URL.Path, err)
		return
	}

	input := &openapi3filter.RequestValidationInput{
		Request:    specReq,
		PathParams: pathParams,
		Route:      route,
		Options: &openapi3filter.Options{
			// The flow tests exercise auth themselves.
			AuthenticationFunc: openapi3filter.NoopAuthenticationFunc,
		},
	}
	// Only when the server accepted the request: the 400-path tests send
	// deliberately malformed bodies, which the spec rightly rejects too. What
	// matters is a request the server accepted that the contract forbids.
	if res.StatusCode < http.StatusBadRequest {
		if err := openapi3filter.ValidateRequest(context.Background(), input); err != nil {
			t.Errorf("%s %s: the server accepted a request that violates api/openapi.yaml: %v",
				req.Method, req.URL.RequestURI(), err)
		}
	}

	if err := openapi3filter.ValidateResponse(context.Background(), &openapi3filter.ResponseValidationInput{
		RequestValidationInput: input,
		Status:                 res.StatusCode,
		Header:                 res.Header,
		Body:                   io.NopCloser(bytes.NewReader(resBody)),
		Options:                input.Options,
	}); err != nil {
		t.Errorf("%s %s -> %d: the response violates api/openapi.yaml: %v\nbody: %s",
			req.Method, req.URL.RequestURI(), res.StatusCode, err, truncate(resBody))
	}
}

// truncate keeps a failure message readable when a body is large.
func truncate(b []byte) string {
	const max = 512
	s := strings.TrimSpace(string(b))
	if len(s) > max {
		return s[:max] + "…"
	}
	return s
}
