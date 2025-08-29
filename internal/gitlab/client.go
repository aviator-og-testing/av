package gitlab

import (
	"context"
	"net/http"
	"strings"
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

// IsHTTPUnauthorized returns true if the given error is an HTTP 401 Unauthorized error.
func IsHTTPUnauthorized(err error) bool {
	// This checks for GitLab API authentication errors
	return strings.Contains(err.Error(), "401") || strings.Contains(err.Error(), "status code: 401") || strings.Contains(err.Error(), "Unauthorized")
}

// TestAPIAccess tests if the GitLab API is accessible with current credentials
func (c *Client) TestAPIAccess(ctx context.Context) error {
	// For now, we'll implement a basic test by attempting to get current user info
	// This will be replaced with actual API calls when the full client is implemented
	
	// Temporarily return nil to allow client creation during development
	// TODO: Replace with actual API test call
	return nil
}