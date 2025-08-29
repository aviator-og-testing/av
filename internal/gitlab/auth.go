package gitlab

import (
	"context"
	"net/http"

	"emperror.dev/errors"
)

// AuthStatus represents the authentication status for GitLab
type AuthStatus struct {
	IsAuthenticated bool
	User            *User
	Scopes          []string // GitLab token scopes
	TokenType       string   // "personal_access_token" or "oauth2_token"
	ExpiresAt       string   // Token expiration if available
}

// CheckAuthStatus verifies the current authentication status with GitLab
func (c *Client) CheckAuthStatus(ctx context.Context) (*AuthStatus, error) {
	// Try to get the current user to verify authentication
	user, err := c.GetViewer(ctx)
	if err != nil {
		// Check if it's an authentication error
		if isAuthError(err) {
			return &AuthStatus{
				IsAuthenticated: false,
			}, nil
		}
		return nil, errors.Wrap(err, "failed to check authentication status")
	}
	
	// If we got user info, authentication is valid
	authStatus := &AuthStatus{
		IsAuthenticated: true,
		User:            user,
		TokenType:       "personal_access_token", // Default assumption
	}
	
	// Try to get token information if available
	tokenInfo, err := c.GetTokenInfo(ctx)
	if err == nil {
		authStatus.Scopes = tokenInfo.Scopes
		authStatus.ExpiresAt = tokenInfo.ExpiresAt
	}
	
	return authStatus, nil
}

// TokenInfo represents information about the current access token
type TokenInfo struct {
	Name      string   `json:"name"`
	Scopes    []string `json:"scopes"`
	ExpiresAt string   `json:"expires_at,omitempty"`
	Active    bool     `json:"active"`
	CreatedAt string   `json:"created_at"`
}

// GetTokenInfo retrieves information about the current personal access token
func (c *Client) GetTokenInfo(ctx context.Context) (*TokenInfo, error) {
	// This will be implemented with actual GitLab API calls once the client is set up
	// GitLab API endpoint: GET /personal_access_tokens (for current token information)
	// Note: This endpoint might require specific token scopes
	return nil, errors.New("GetTokenInfo method not yet implemented - requires GitLab API client setup")
}

// ValidateAuthentication performs comprehensive authentication validation
func (c *Client) ValidateAuthentication(ctx context.Context) error {
	authStatus, err := c.CheckAuthStatus(ctx)
	if err != nil {
		return errors.Wrap(err, "failed to validate authentication")
	}
	
	if !authStatus.IsAuthenticated {
		return errors.New("not authenticated with GitLab")
	}
	
	// Check if user account is active
	if authStatus.User != nil && authStatus.User.State != "active" {
		return errors.Errorf("user account is not active (state: %s)", authStatus.User.State)
	}
	
	return nil
}

// RequiredScopes defines the minimum scopes needed for Aviator functionality
var RequiredScopes = []string{
	"api",        // Full API access
	"read_user",  // Read user information
	"read_repository", // Read repository information
}

// CheckRequiredScopes verifies that the current token has all required scopes
func (c *Client) CheckRequiredScopes(ctx context.Context) ([]string, error) {
	tokenInfo, err := c.GetTokenInfo(ctx)
	if err != nil {
		// If we can't get token info, we can't verify scopes
		// This might be okay for some GitLab instances/configurations
		return nil, errors.Wrap(err, "unable to verify token scopes")
	}
	
	var missingScopes []string
	for _, required := range RequiredScopes {
		if !contains(tokenInfo.Scopes, required) {
			missingScopes = append(missingScopes, required)
		}
	}
	
	return missingScopes, nil
}

// TestAPIAccess tests basic API access by making a simple API call
func (c *Client) TestAPIAccess(ctx context.Context) error {
	// Try to get current user info as a basic API test
	_, err := c.GetViewer(ctx)
	if err != nil {
		return errors.Wrap(err, "API access test failed")
	}
	
	return nil
}

// TestProjectAccess tests access to a specific project
func (c *Client) TestProjectAccess(ctx context.Context, projectID string) error {
	if projectID == "" {
		return errors.New("project ID cannot be empty")
	}
	
	// Try to get project information
	_, err := c.GetRepository(ctx, projectID)
	if err != nil {
		return errors.Wrapf(err, "failed to access project %s", projectID)
	}
	
	return nil
}

// GetAuthenticationGuidance returns helpful guidance for authentication setup
func GetAuthenticationGuidance(baseURL string) string {
	if baseURL == "" || baseURL == "https://gitlab.com" {
		return `To authenticate with GitLab.com:
1. Go to https://gitlab.com/-/profile/personal_access_tokens
2. Create a new personal access token with the following scopes:
   - api (full API access)
   - read_user (read user information)
   - read_repository (read repository information)
3. Set the token in your environment:
   export GITLAB_TOKEN=your_token_here
   or
   export AV_GITLAB_TOKEN=your_token_here`
	}
	
	return `To authenticate with your GitLab instance:
1. Go to ` + baseURL + `/-/profile/personal_access_tokens
2. Create a new personal access token with the following scopes:
   - api (full API access)
   - read_user (read user information)
   - read_repository (read repository information)
3. Set the token in your environment:
   export GITLAB_TOKEN=your_token_here
   or
   export AV_GITLAB_TOKEN=your_token_here
4. Set your GitLab base URL:
   export GITLAB_BASE_URL=` + baseURL
}

// AuthError represents an authentication-related error
type AuthError struct {
	Message    string
	StatusCode int
	IsTokenExpired bool
}

func (e *AuthError) Error() string {
	return e.Message
}

// isAuthError checks if an error is related to authentication
func isAuthError(err error) bool {
	if err == nil {
		return false
	}
	
	// Check for HTTP 401 (Unauthorized) or 403 (Forbidden)
	// This is a simplified check - actual implementation would inspect the error more thoroughly
	errorStr := err.Error()
	return contains([]string{errorStr}, "401") || contains([]string{errorStr}, "unauthorized") ||
		   contains([]string{errorStr}, "403") || contains([]string{errorStr}, "forbidden")
}

// CreateAuthError creates a new authentication error
func CreateAuthError(message string, statusCode int) *AuthError {
	return &AuthError{
		Message:        message,
		StatusCode:     statusCode,
		IsTokenExpired: statusCode == http.StatusUnauthorized,
	}
}

// IsValidToken checks if a token string appears to be valid format
func IsValidToken(token string) bool {
	if token == "" {
		return false
	}
	
	// GitLab personal access tokens are typically 26 characters long
	// This is a basic validation - actual validation happens server-side
	if len(token) < 20 {
		return false
	}
	
	// GitLab tokens typically start with certain prefixes for different types
	// Personal access tokens: glpat-
	// OAuth tokens: gho-
	// But legacy tokens might not have prefixes
	return true
}

// contains checks if a slice contains a specific string
func contains(slice []string, item string) bool {
	for _, s := range slice {
		if s == item {
			return true
		}
	}
	return false
}