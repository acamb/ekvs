package client

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"ekvs/internal/tui/auth"
	"ekvs/internal/tui/session"
)

// Client is a signed HTTP client for the EKVS server API.
// It is safe to share across goroutines.
type Client struct {
	baseURL    string
	sess       *session.Session
	httpClient *http.Client
}

// New creates a Client that sends requests to baseURL, signing each request
// with the credentials stored in sess.
func New(baseURL string, sess *session.Session) *Client {
	return &Client{
		baseURL:    strings.TrimRight(baseURL, "/"),
		sess:       sess,
		httpClient: &http.Client{Timeout: 10 * time.Second},
	}
}

// do builds, signs, and executes an HTTP request.
func (c *Client) do(method, path string, body io.Reader) (*http.Response, error) {
	url := c.baseURL + path

	req, err := http.NewRequest(method, url, body)
	if err != nil {
		return nil, fmt.Errorf("build request: %w", err)
	}

	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}

	headers, err := auth.SignRequest(c.sess, method, path, time.Now())
	if err != nil {
		return nil, fmt.Errorf("sign request: %w", err)
	}
	for k, v := range headers {
		req.Header.Set(k, v)
	}

	return c.httpClient.Do(req)
}

// mapStatus converts a non-2xx HTTP response into a typed error.
// The response body is consumed and closed.
func mapStatus(resp *http.Response) error {
	defer resp.Body.Close()
	raw, _ := io.ReadAll(resp.Body)
	body := strings.TrimSpace(string(raw))

	switch resp.StatusCode {
	case http.StatusUnauthorized:
		return ErrUnauthorized
	case http.StatusNotFound:
		return ErrNotFound
	case http.StatusConflict:
		return ErrConflict
	default:
		return &ServerError{StatusCode: resp.StatusCode, Body: body}
	}
}

// ListProjects returns the names of all projects owned by the authenticated user.
func (c *Client) ListProjects() ([]string, error) {
	resp, err := c.do(http.MethodGet, "/projects", nil)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, mapStatus(resp)
	}

	var payload struct {
		Projects []string `json:"projects"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&payload); err != nil {
		return nil, fmt.Errorf("decode response: %w", err)
	}
	if payload.Projects == nil {
		payload.Projects = []string{}
	}
	return payload.Projects, nil
}

// CreateProject creates a new project with the given name.
func (c *Client) CreateProject(name string) error {
	resp, err := c.do(http.MethodPost, "/projects/"+name, nil)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusCreated {
		return mapStatus(resp)
	}
	return nil
}

// SecretEntry represents a single key-value secret as returned by the server.
// The Value field contains the encrypted blob stored on the server.
type SecretEntry struct {
	Key   string
	Value string // encrypted blob
}

// ListSecrets returns all secrets for the given project.
// Values are encrypted blobs as stored on the server.
func (c *Client) ListSecrets(project string) ([]SecretEntry, error) {
	resp, err := c.do(http.MethodGet, "/projects/"+project+"/secrets", nil)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, mapStatus(resp)
	}

	var payload struct {
		Secrets []struct {
			Key   string `json:"key"`
			Value string `json:"value"`
		} `json:"secrets"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&payload); err != nil {
		return nil, fmt.Errorf("decode response: %w", err)
	}
	entries := make([]SecretEntry, 0, len(payload.Secrets))
	for _, s := range payload.Secrets {
		entries = append(entries, SecretEntry{Key: s.Key, Value: s.Value})
	}
	return entries, nil
}

// SetSecret creates or updates a secret. value must be the encrypted blob.
func (c *Client) SetSecret(project, key, value string) error {
	body, err := json.Marshal(map[string]string{"value": value})
	if err != nil {
		return fmt.Errorf("encode body: %w", err)
	}
	resp, err := c.do(http.MethodPut, "/projects/"+project+"/secrets/"+key, strings.NewReader(string(body)))
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return mapStatus(resp)
	}
	return nil
}

// DeleteSecret removes a secret from the given project.
func (c *Client) DeleteSecret(project, key string) error {
	resp, err := c.do(http.MethodDelete, "/projects/"+project+"/secrets/"+key, nil)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusNoContent {
		return mapStatus(resp)
	}
	return nil
}

// DeleteProject deletes the project with the given name.
func (c *Client) DeleteProject(name string) error {
	resp, err := c.do(http.MethodDelete, "/projects/"+name, nil)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusNoContent {
		return mapStatus(resp)
	}
	return nil
}
