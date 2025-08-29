package gitlab

import (
	"context"
	"net/http"
	"time"

	"emperror.dev/errors"
)

// Client represents the GitLab API client
type Client struct {
	httpClient *http.Client
	baseURL    string
	token      string
}

// NewClient creates a new GitLab client with the provided token and base URL
func NewClient(ctx context.Context, token, baseURL string) (*Client, error) {
	if token == "" {
		return nil, errors.New("GitLab token cannot be empty")
	}
	
	if baseURL == "" {
		baseURL = "https://gitlab.com"
	}
	
	client := &Client{
		httpClient: &http.Client{
			Timeout: 30 * time.Second,
		},
		baseURL: baseURL,
		token:   token,
	}
	
	// Validate the client by testing authentication
	if err := client.TestAPIAccess(ctx); err != nil {
		return nil, errors.Wrap(err, "failed to validate GitLab client")
	}
	
	return client, nil
}

// GetBaseURL returns the configured GitLab base URL
func (c *Client) GetBaseURL() string {
	return c.baseURL
}

// GetToken returns the configured token (masked for security)
func (c *Client) GetToken() string {
	if len(c.token) <= 8 {
		return "***"
	}
	return c.token[:4] + "***" + c.token[len(c.token)-4:]
}