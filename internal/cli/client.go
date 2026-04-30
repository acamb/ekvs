package cli

import (
	"crypto"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"time"
)

// ErrNotFound is returned by Client methods when the server responds with HTTP 404.
var ErrNotFound = errors.New("not found")

// secretEntry is the JSON representation of a single secret.
type secretEntry struct {
	Key   string `json:"key"`
	Value string `json:"value"`
}

// Client is a signed HTTP client for the EKVS server API.
type Client struct {
	BaseURL    string
	HTTPClient *http.Client
}

// NewClient creates a Client targeting baseURL (scheme://host:port).
func NewClient(baseURL string) *Client {
	return &Client{BaseURL: baseURL, HTTPClient: http.DefaultClient}
}

// ListSecrets fetches all secrets of a project.
// Returns ErrNotFound if the project does not exist (HTTP 404).
func (c *Client) ListSecrets(signer crypto.Signer, fingerprint, project string) ([]secretEntry, error) {
	path := "/projects/" + project + "/secrets"
	req, err := http.NewRequest(http.MethodGet, c.BaseURL+path, nil)
	if err != nil {
		return nil, fmt.Errorf("build request: %w", err)
	}

	headers, err := SignedHeaders(signer, fingerprint, http.MethodGet, path, time.Now())
	if err != nil {
		return nil, fmt.Errorf("sign request: %w", err)
	}
	for k, v := range headers {
		req.Header.Set(k, v)
	}

	resp, err := c.HTTPClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("GET %s: %w", path, err)
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusNotFound {
		return nil, ErrNotFound
	}
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("GET %s: unexpected status %s", path, resp.Status)
	}

	var body struct {
		Secrets []secretEntry `json:"secrets"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&body); err != nil {
		return nil, fmt.Errorf("decode response: %w", err)
	}
	return body.Secrets, nil
}

// GetSecret fetches a single secret by key.
// Returns ErrNotFound if the project or key does not exist (HTTP 404).
func (c *Client) GetSecret(signer crypto.Signer, fingerprint, project, key string) (*secretEntry, error) {
	path := "/projects/" + project + "/secrets/" + key
	req, err := http.NewRequest(http.MethodGet, c.BaseURL+path, nil)
	if err != nil {
		return nil, fmt.Errorf("build request: %w", err)
	}

	headers, err := SignedHeaders(signer, fingerprint, http.MethodGet, path, time.Now())
	if err != nil {
		return nil, fmt.Errorf("sign request: %w", err)
	}
	for k, v := range headers {
		req.Header.Set(k, v)
	}

	resp, err := c.HTTPClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("GET %s: %w", path, err)
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusNotFound {
		return nil, ErrNotFound
	}
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("GET %s: unexpected status %s", path, resp.Status)
	}

	var entry secretEntry
	if err := json.NewDecoder(resp.Body).Decode(&entry); err != nil {
		return nil, fmt.Errorf("decode response: %w", err)
	}
	return &entry, nil
}
