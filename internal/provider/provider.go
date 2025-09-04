package provider

import (
	"context"
	"strings"
)

// Provider represents the type of Git hosting provider
type Provider string

const (
	// ProviderGitHub represents GitHub as the Git hosting provider
	ProviderGitHub Provider = "github"
	// ProviderGitLab represents GitLab as the Git hosting provider
	ProviderGitLab Provider = "gitlab"
)

// ProviderConfig holds provider-specific configuration
type ProviderConfig struct {
	// Provider type (GitHub or GitLab)
	Provider Provider
	// API token for authentication
	Token string
	// Base URL for API (useful for self-hosted instances)
	BaseURL string
	// Repository owner/organization
	Owner string
	// Repository name
	Repo string
}

// PullRequestInfo contains information about a pull/merge request
type PullRequestInfo struct {
	ID          string
	Number      int
	Title       string
	Body        string
	State       string
	Draft       bool
	URL         string
	HeadBranch  string
	BaseBranch  string
	Author      string
	Reviewers   []string
}

// CreatePullRequestOptions contains options for creating a pull/merge request
type CreatePullRequestOptions struct {
	Title      string
	Body       string
	HeadBranch string
	BaseBranch string
	Draft      bool
}

// UpdatePullRequestOptions contains options for updating a pull/merge request
type UpdatePullRequestOptions struct {
	Title      *string
	Body       *string
	BaseBranch *string
	Draft      *bool
}

// PullRequestProvider defines the interface for pull/merge request operations
type PullRequestProvider interface {
	// CreatePullRequest creates a new pull/merge request
	CreatePullRequest(ctx context.Context, opts CreatePullRequestOptions) (*PullRequestInfo, error)
	
	// UpdatePullRequest updates an existing pull/merge request
	UpdatePullRequest(ctx context.Context, id string, opts UpdatePullRequestOptions) (*PullRequestInfo, error)
	
	// GetPullRequests retrieves pull/merge requests, optionally filtered by branch
	GetPullRequests(ctx context.Context, branch string) ([]*PullRequestInfo, error)
	
	// ConvertToDraft converts a pull/merge request to draft status
	ConvertToDraft(ctx context.Context, id string) error
	
	// ConvertToReady converts a pull/merge request from draft to ready status
	ConvertToReady(ctx context.Context, id string) error
	
	// AddReviewers adds reviewers to a pull/merge request
	AddReviewers(ctx context.Context, id string, reviewers []string) error
}

// RepositoryInfo contains information about a repository
type RepositoryInfo struct {
	Owner         string
	Name          string
	DefaultBranch string
	URL           string
}

// ViewerInfo contains information about the authenticated user
type ViewerInfo struct {
	Login string
	Email string
	Name  string
}

// RepositoryProvider defines the interface for repository operations
type RepositoryProvider interface {
	// GetRepository retrieves repository information
	GetRepository(ctx context.Context) (*RepositoryInfo, error)
	
	// GetViewer retrieves information about the authenticated user
	GetViewer(ctx context.Context) (*ViewerInfo, error)
	
	// GetDefaultBranch retrieves the default branch of the repository
	GetDefaultBranch(ctx context.Context) (string, error)
}

// Client combines both pull request and repository providers
type Client interface {
	PullRequestProvider
	RepositoryProvider
}

// DetectProviderFromURL detects the provider type from a remote URL
func DetectProviderFromURL(remoteURL string) Provider {
	// Normalize the URL to lowercase for case-insensitive matching
	lowerURL := strings.ToLower(remoteURL)
	
	// Check for GitLab indicators
	if strings.Contains(lowerURL, "gitlab.com") || strings.Contains(lowerURL, "gitlab") {
		return ProviderGitLab
	}
	
	// Check for GitHub indicators
	if strings.Contains(lowerURL, "github.com") || strings.Contains(lowerURL, "github") {
		return ProviderGitHub
	}
	
	// Default to GitHub for backward compatibility
	return ProviderGitHub
}