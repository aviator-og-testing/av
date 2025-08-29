package gitlab

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewClient_Success(t *testing.T) {
	ctx := context.Background()
	token := "test-token"
	baseURL := "https://gitlab.example.com"
	
	// Since TestAPIAccess currently returns nil, this should succeed
	client, err := NewClient(ctx, token, baseURL)
	
	assert.NoError(t, err)
	assert.NotNil(t, client)
	assert.Equal(t, baseURL, client.GetBaseURL())
	assert.Contains(t, client.GetToken(), "test")
}

func TestNewClient_DefaultBaseURL(t *testing.T) {
	ctx := context.Background()
	token := "test-token"
	
	client, err := NewClient(ctx, token, "")
	
	assert.NoError(t, err)
	assert.NotNil(t, client)
	assert.Equal(t, "https://gitlab.com", client.GetBaseURL())
}

func TestNewClient_EmptyToken(t *testing.T) {
	ctx := context.Background()
	
	client, err := NewClient(ctx, "", "https://gitlab.com")
	
	assert.Error(t, err)
	assert.Nil(t, client)
	assert.Contains(t, err.Error(), "GitLab token cannot be empty")
}

func TestClient_GetBaseURL(t *testing.T) {
	tests := []struct {
		name     string
		baseURL  string
		expected string
	}{
		{
			name:     "Custom GitLab instance",
			baseURL:  "https://gitlab.company.com",
			expected: "https://gitlab.company.com",
		},
		{
			name:     "GitLab.com",
			baseURL:  "https://gitlab.com",
			expected: "https://gitlab.com",
		},
		{
			name:     "Empty baseURL defaults to GitLab.com",
			baseURL:  "",
			expected: "https://gitlab.com",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx := context.Background()
			client, err := NewClient(ctx, "test-token", tt.baseURL)
			require.NoError(t, err)
			
			assert.Equal(t, tt.expected, client.GetBaseURL())
		})
	}
}

func TestClient_GetToken(t *testing.T) {
	tests := []struct {
		name     string
		token    string
		expected string
	}{
		{
			name:     "Long token",
			token:    "glpat-xxxxxxxxxxxxxxxxxxxx",
			expected: "glpa***xxxx",
		},
		{
			name:     "Short token",
			token:    "short",
			expected: "***",
		},
		{
			name:     "Very short token",
			token:    "abc",
			expected: "***",
		},
		{
			name:     "Minimum length token",
			token:    "12345678",
			expected: "***",
		},
		{
			name:     "Token just over minimum",
			token:    "123456789",
			expected: "1234***6789",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx := context.Background()
			client, err := NewClient(ctx, tt.token, "https://gitlab.com")
			require.NoError(t, err)
			
			masked := client.GetToken()
			assert.Equal(t, tt.expected, masked)
		})
	}
}

func TestIsHTTPUnauthorized(t *testing.T) {
	tests := []struct {
		name        string
		errorMsg    string
		expectMatch bool
	}{
		{
			name:        "HTTP 401 with status code text",
			errorMsg:    "HTTP request failed with status code: 401",
			expectMatch: true,
		},
		{
			name:        "Simple 401 error",
			errorMsg:    "401 Unauthorized",
			expectMatch: true,
		},
		{
			name:        "Unauthorized text error",
			errorMsg:    "Request failed: Unauthorized",
			expectMatch: true,
		},
		{
			name:        "Other HTTP error",
			errorMsg:    "HTTP 404 Not Found",
			expectMatch: false,
		},
		{
			name:        "Network error",
			errorMsg:    "connection timeout",
			expectMatch: false,
		},
		{
			name:        "Empty error",
			errorMsg:    "",
			expectMatch: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := assert.AnError
			// Mock error message
			err = &testError{message: tt.errorMsg}
			
			result := IsHTTPUnauthorized(err)
			assert.Equal(t, tt.expectMatch, result)
		})
	}
}

func TestClient_HTTPClientConfiguration(t *testing.T) {
	ctx := context.Background()
	client, err := NewClient(ctx, "test-token", "https://gitlab.com")
	require.NoError(t, err)

	// Verify that the HTTP client has appropriate timeout
	assert.NotNil(t, client.httpClient)
	assert.Equal(t, 30*time.Second, client.httpClient.Timeout)
}

func TestTestAPIAccess_CurrentImplementation(t *testing.T) {
	ctx := context.Background()
	client, err := NewClient(ctx, "test-token", "https://gitlab.com")
	require.NoError(t, err)

	// Test the current implementation which returns nil
	err = client.TestAPIAccess(ctx)
	assert.NoError(t, err, "TestAPIAccess should return nil in current implementation")
}

// testError implements error interface for testing
type testError struct {
	message string
}

func (e *testError) Error() string {
	return e.message
}