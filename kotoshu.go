// Package kotoshu is a Go client for the Kotoshu HTTP spell-check API.
package kotoshu

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"
)

const Version = "0.1.0"

// Suggestion is a single correction candidate.
type Suggestion struct {
	Word       string  `json:"word"`
	Distance   int     `json:"distance"`
	Confidence float64 `json:"confidence"`
	Source     string  `json:"source"`
}

// WordError is one misspelled word with its candidates.
type WordError struct {
	Word        string       `json:"word"`
	Position    *int         `json:"position"`
	Suggestions []Suggestion `json:"suggestions"`
}

// DocumentResult is the response from /v1/check.
type DocumentResult struct {
	File      string      `json:"file"`
	WordCount int         `json:"word_count"`
	Errors    []WordError `json:"errors"`
}

// Detection is the response from /v1/detect.
type Detection struct {
	Language   string  `json:"language"`
	Confidence float64 `json:"confidence"`
}

// Languages is the response from /v1/languages.
type Languages struct {
	Cached []string `json:"cached"`
}

// Health is the response from /v1/health.
type Health struct {
	Status    string            `json:"status"`
	Ready     map[string]bool   `json:"ready"`
	Timestamp string            `json:"timestamp,omitempty"`
}

// APIError wraps a server error response.
type APIError struct {
	Code    string
	Message string
}

func (e *APIError) Error() string { return e.Code + ": " + e.Message }

// IsResourceNotSetup reports whether err is a 422 resource_not_setup.
func IsResourceNotSetup(err error) bool {
	var apiErr *APIError
	if errors.As(err, &apiErr) {
		return apiErr.Code == "resource_not_setup"
	}
	return false
}

// Option configures a Client.
type Option func(*Client)

// WithLanguage sets the default language.
func WithLanguage(lang string) Option {
	return func(c *Client) { c.defaultLang = lang }
}

// WithTimeout sets the HTTP timeout.
func WithTimeout(d time.Duration) Option {
	return func(c *Client) { c.timeout = d }
}

// WithHTTPClient supplies a custom *http.Client.
func WithHTTPClient(h *http.Client) Option {
	return func(c *Client) { c.http = h }
}

// Client is the Kotoshu HTTP API client.
type Client struct {
	baseURL     *url.URL
	defaultLang string
	timeout     time.Duration
	http        *http.Client
}

// New constructs a client.
func New(baseURL string, opts ...Option) (*Client, error) {
	u, err := url.Parse(strings.TrimRight(baseURL, "/"))
	if err != nil {
		return nil, fmt.Errorf("kotoshu: invalid base URL %q: %w", baseURL, err)
	}
	c := &Client{
		baseURL:     u,
		defaultLang: "en",
		timeout:     30 * time.Second,
		http:        http.DefaultClient,
	}
	for _, opt := range opts {
		opt(c)
	}
	return c, nil
}

// Health probes /v1/health.
func (c *Client) Health(ctx context.Context) (*Health, error) {
	var h Health
	if err := c.get(ctx, "/v1/health", &h); err != nil {
		return nil, err
	}
	return &h, nil
}

// Languages returns cached languages.
func (c *Client) Languages(ctx context.Context) ([]string, error) {
	var l Languages
	if err := c.get(ctx, "/v1/languages", &l); err != nil {
		return nil, err
	}
	return l.Cached, nil
}

// CheckOptions controls a single /v1/check call.
type CheckOptions struct {
	Language string
	Format   string // "full" or "errors"
}

// Check posts text to /v1/check.
func (c *Client) Check(ctx context.Context, text string, opts *CheckOptions) (*DocumentResult, error) {
	lang := c.defaultLang
	fmt_ := "full"
	if opts != nil {
		if opts.Language != "" {
			lang = opts.Language
		}
		if opts.Format != "" {
			fmt_ = opts.Format
		}
	}
	body := map[string]string{"text": text, "language": lang, "format": fmt_}
	var r DocumentResult
	if err := c.post(ctx, "/v1/check", body, &r); err != nil {
		return nil, err
	}
	return &r, nil
}

// SuggestOptions controls a single /v1/suggest call.
type SuggestOptions struct {
	Language string
	Max      int
}

// Suggest posts word to /v1/suggest.
func (c *Client) Suggest(ctx context.Context, word string, opts *SuggestOptions) ([]Suggestion, error) {
	lang := c.defaultLang
	body := map[string]interface{}{"word": word, "language": lang}
	if opts != nil {
		if opts.Language != "" {
			body["language"] = opts.Language
		}
		if opts.Max > 0 {
			body["max"] = opts.Max
		}
	}
	var r struct {
		Word        string       `json:"word"`
		Suggestions []Suggestion `json:"suggestions"`
	}
	if err := c.post(ctx, "/v1/suggest", body, &r); err != nil {
		return nil, err
	}
	return r.Suggestions, nil
}

// Detect posts text to /v1/detect.
func (c *Client) Detect(ctx context.Context, text string) (*Detection, error) {
	var d Detection
	if err := c.post(ctx, "/v1/detect", map[string]string{"text": text}, &d); err != nil {
		return nil, err
	}
	return &d, nil
}

// Correct returns true if the word is in the dictionary.
func (c *Client) Correct(ctx context.Context, word string, opts *CheckOptions) (bool, error) {
	r, err := c.Check(ctx, word, opts)
	if err != nil {
		return false, err
	}
	return len(r.Errors) == 0, nil
}

// ---- Internals ----

func (c *Client) get(ctx context.Context, path string, out interface{}) error {
	return c.do(ctx, http.MethodGet, path, nil, out)
}

func (c *Client) post(ctx context.Context, path string, body interface{}, out interface{}) error {
	return c.do(ctx, http.MethodPost, path, body, out)
}

func (c *Client) do(ctx context.Context, method, path string, body interface{}, out interface{}) error {
	var bodyReader io.Reader
	if body != nil {
		b, err := json.Marshal(body)
		if err != nil {
			return fmt.Errorf("kotoshu: marshal body: %w", err)
		}
		bodyReader = bytes.NewReader(b)
	}

	u := *c.baseURL
	u.Path += path

	req, err := http.NewRequestWithContext(ctx, method, u.String(), bodyReader)
	if err != nil {
		return fmt.Errorf("kotoshu: new request: %w", err)
	}
	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}
	req.Header.Set("Accept", "application/json")

	client := c.http
	if client == nil {
		client = &http.Client{Timeout: c.timeout}
	}
	resp, err := client.Do(req)
	if err != nil {
		return fmt.Errorf("kotoshu: HTTP %s %s: %w", method, path, err)
	}
	defer resp.Body.Close()

	raw, err := io.ReadAll(resp.Body)
	if err != nil {
		return fmt.Errorf("kotoshu: read body: %w", err)
	}

	if resp.StatusCode >= 200 && resp.StatusCode < 300 {
		if out == nil || len(raw) == 0 {
			return nil
		}
		if err := json.Unmarshal(raw, out); err != nil {
			return fmt.Errorf("kotoshu: decode body: %w (raw=%q)", err, truncate(raw))
		}
		return nil
	}

	// Error response
	var errObj struct {
		Error   string `json:"error"`
		Message string `json:"message"`
	}
	_ = json.Unmarshal(raw, &errObj)
	if errObj.Error == "" {
		errObj.Error = "http_error"
		errObj.Message = fmt.Sprintf("HTTP %d: %s", resp.StatusCode, truncate(raw))
	}
	return &APIError{Code: errObj.Error, Message: errObj.Message}
}

func truncate(b []byte) string {
	if len(b) > 200 {
		return string(b[:200])
	}
	return string(b)
}
