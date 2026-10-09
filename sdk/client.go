// Package sdk provides an HTTP client for the PolicyLens API.
package sdk

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"
)

type Client struct {
	BaseURL string
	HTTP    *http.Client
}
type Policy struct {
	ID       string `json:"id"`
	Title    string `json:"title"`
	Category string `json:"category"`
	Summary  string `json:"summary"`
}
type Answer struct {
	Model     string   `json:"model,omitempty"`
	Text      string   `json:"answer"`
	Citations []string `json:"citations"`
	Mode      string   `json:"mode"`
	Abstained bool     `json:"abstained"`
}
type Result struct {
	PolicyID string `json:"policy_id"`
	Status   string `json:"status"`
	Message  string `json:"message"`
}
type Check struct {
	Engine   string   `json:"engine"`
	Resource string   `json:"resource"`
	Results  []Result `json:"results"`
	Summary  struct {
		Pass    int `json:"pass"`
		Fail    int `json:"fail"`
		Error   int `json:"error"`
		Skipped int `json:"skipped"`
	} `json:"summary"`
}
type APIError struct {
	Status    int
	Message   string
	RequestID string
}

func (e *APIError) Error() string {
	return fmt.Sprintf("PolicyLens HTTP %d: %s (request %s)", e.Status, e.Message, e.RequestID)
}
func New(baseURL string) *Client {
	return &Client{BaseURL: strings.TrimRight(baseURL, "/"), HTTP: &http.Client{Timeout: 75 * time.Second}}
}
func (c *Client) request(ctx context.Context, method, path string, input, output any) error {
	var body io.Reader
	if input != nil {
		b, err := json.Marshal(input)
		if err != nil {
			return err
		}
		body = bytes.NewReader(b)
	}
	req, err := http.NewRequestWithContext(ctx, method, c.BaseURL+path, body)
	if err != nil {
		return err
	}
	if input != nil {
		req.Header.Set("Content-Type", "application/json")
	}
	client := c.HTTP
	if client == nil {
		client = &http.Client{Timeout: 75 * time.Second}
	}
	resp, err := client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode >= 400 {
		var e struct {
			Message   string `json:"error"`
			RequestID string `json:"request_id"`
		}
		_ = json.NewDecoder(io.LimitReader(resp.Body, 1<<20)).Decode(&e)
		if e.Message == "" {
			e.Message = http.StatusText(resp.StatusCode)
		}
		return &APIError{resp.StatusCode, e.Message, e.RequestID}
	}
	return json.NewDecoder(io.LimitReader(resp.Body, 1<<20)).Decode(output)
}
func (c *Client) Policies(ctx context.Context) ([]Policy, error) {
	var result []Policy
	err := c.request(ctx, "GET", "/api/policies", nil, &result)
	return result, err
}
func (c *Client) Ask(ctx context.Context, question, policyID string) (Answer, error) {
	var result Answer
	err := c.request(ctx, "POST", "/api/ask", map[string]string{"question": question, "policy_id": policyID}, &result)
	return result, err
}
func (c *Client) Check(ctx context.Context, manifest string) (Check, error) {
	var result Check
	err := c.request(ctx, "POST", "/api/check", map[string]string{"manifest": manifest}, &result)
	return result, err
}

// AskWithMode compares retrieved excerpts with generated answers.
func (c *Client) AskWithMode(ctx context.Context, question, policyID, mode string) (Answer, error) {
	var result Answer
	err := c.request(ctx, "POST", "/api/ask", map[string]string{"question": question, "policy_id": policyID, "mode": mode}, &result)
	return result, err
}
