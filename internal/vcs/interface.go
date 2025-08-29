package vcs

import (
	"context"
	"fmt"
	"regexp"
	"strings"

	"emperror.dev/errors"
	"github.com/aviator-co/av/internal/config"
)

type PullRequest interface {
	GetID() string
	GetNumber() int64
	HeadBranchName() string
	BaseBranchName() string
	GetTitle() string
	GetBody() string
	GetPermalink() string
	IsDraft() bool
	IsOpen() bool
	IsMerged() bool
	GetMergeCommit() string
}

type Repository interface {
	GetID() string
	GetOwner() string
	GetName() string
	GetFullName() string
}

type User interface {
	GetID() string
	GetLogin() string
}

type Team interface {
	GetID() string
	GetName() string
}

type CreatePullRequestInput struct {
	RepositoryID string
	Title        string
	Body         string
	HeadRefName  string
	BaseRefName  string
	Draft        bool
}

type UpdatePullRequestInput struct {
	ID    string
	Title *string
	Body  *string
	Draft *bool
}

type RequestReviewsInput struct {
	PullRequestID string
	UserIDs       []string
	TeamIDs       []string
	Union         bool
}

type GetPullRequestsInput struct {
	Owner       string
	Repo        string
	HeadRefName string
	BaseRefName string
	States      []string
	First       int32
	After       string
}

type GetPullRequestsPage struct {
	PullRequests []PullRequest
	HasNextPage  bool
	EndCursor    string
}

type Provider interface {
	CreatePR(ctx context.Context, input CreatePullRequestInput) (PullRequest, error)
	GetPR(ctx context.Context, id string) (PullRequest, error)
	UpdatePR(ctx context.Context, input UpdatePullRequestInput) (PullRequest, error)
	GetPRs(ctx context.Context, input GetPullRequestsInput) (*GetPullRequestsPage, error)
	RequestReviews(ctx context.Context, input RequestReviewsInput) (PullRequest, error)
	ConvertToDraft(ctx context.Context, id string) (PullRequest, error)
	MarkReadyForReview(ctx context.Context, id string) (PullRequest, error)
	
	GetRepository(ctx context.Context, owner, repo string) (Repository, error)
	GetUser(ctx context.Context, login string) (User, error)
	GetViewer(ctx context.Context) (User, error)
	GetOrganizationTeam(ctx context.Context, org, team string) (Team, error)
}

type ProviderType string

const (
	ProviderTypeGitHub ProviderType = "github"
	ProviderTypeGitLab ProviderType = "gitlab"
)

var (
	githubURLPattern = regexp.MustCompile(`(?i)github\.com`)
	gitlabURLPattern = regexp.MustCompile(`(?i)gitlab\.com|gitlab\.`)
)

func DetectProviderFromURL(remoteURL string) ProviderType {
	// Check for explicit GitLab patterns first
	if gitlabURLPattern.MatchString(remoteURL) {
		return ProviderTypeGitLab
	}
	// Check for explicit GitHub patterns
	if githubURLPattern.MatchString(remoteURL) {
		return ProviderTypeGitHub
	}
	
	// Default fallback to GitHub for unknown URLs
	return ProviderTypeGitHub
}

type ProviderError struct {
	Provider    ProviderType
	Message     string
	Suggestions []string
}

func (e ProviderError) Error() string {
	msg := e.Message
	if len(e.Suggestions) > 0 {
		msg += "\n\nTo fix this, you can:"
		for _, suggestion := range e.Suggestions {
			msg += "\n  - " + suggestion
		}
	}
	return msg
}

func NewProvider(ctx context.Context, providerType ProviderType) (Provider, error) {
	switch providerType {
	case ProviderTypeGitHub:
		if config.Av.GitHub.Token == "" {
			return nil, ProviderError{
				Provider: ProviderTypeGitHub,
				Message:  "GitHub token not configured",
				Suggestions: []string{
					"Set the AV_GITHUB_TOKEN environment variable",
					"Set the GITHUB_TOKEN environment variable",
					"Add 'github.token' to your av config file",
					"Run 'av auth github' to configure authentication",
				},
			}
		}
		return NewGitHubProvider(ctx, config.Av.GitHub.Token)
	case ProviderTypeGitLab:
		if config.Av.GitLab.Token == "" {
			return nil, ProviderError{
				Provider: ProviderTypeGitLab,
				Message:  "GitLab token not configured",
				Suggestions: []string{
					"Set the AV_GITLAB_TOKEN environment variable",
					"Set the GITLAB_TOKEN environment variable",
					"Add 'gitlab.token' to your av config file",
					"Create a personal access token at https://gitlab.com/-/profile/personal_access_tokens",
				},
			}
		}
		return NewGitLabProvider(ctx, config.Av.GitLab.Token, config.Av.GitLab.BaseURL)
	default:
		return nil, errors.Errorf("unsupported provider type: %s", providerType)
	}
}

var NewGitHubProvider func(ctx context.Context, token string) (Provider, error)
var NewGitLabProvider func(ctx context.Context, token, baseURL string) (Provider, error)

func DetectAndCreateProvider(ctx context.Context, remoteURL string) (Provider, error) {
	return DetectAndCreateProviderWithOverride(ctx, remoteURL, "")
}

func DetectAndCreateProviderWithOverride(ctx context.Context, remoteURL, providerOverride string) (Provider, error) {
	var providerType ProviderType
	
	// Check for explicit provider override first
	if providerOverride != "" {
		switch strings.ToLower(providerOverride) {
		case "github":
			providerType = ProviderTypeGitHub
		case "gitlab":
			providerType = ProviderTypeGitLab
		default:
			return nil, errors.Errorf("invalid provider override: %s (must be 'github' or 'gitlab')", providerOverride)
		}
	} else {
		// Auto-detect from URL
		providerType = DetectProviderFromURL(remoteURL)
	}
	
	provider, err := NewProvider(ctx, providerType)
	if err != nil {
		// Add context about which repository triggered this error
		if providerError, ok := err.(ProviderError); ok {
			providerError.Message = fmt.Sprintf("Repository at %s requires %s authentication, but %s", 
				remoteURL, providerError.Provider, providerError.Message)
			return nil, providerError
		}
		return nil, errors.Wrapf(err, "failed to create %s provider for repository at %s", providerType, remoteURL)
	}
	
	return provider, nil
}

func ValidateProviderConfiguration(providerType ProviderType) error {
	switch providerType {
	case ProviderTypeGitHub:
		if config.Av.GitHub.Token == "" {
			return ProviderError{
				Provider: ProviderTypeGitHub,
				Message:  "GitHub token not configured",
				Suggestions: []string{
					"Set the AV_GITHUB_TOKEN environment variable",
					"Set the GITHUB_TOKEN environment variable", 
					"Add 'github.token' to your av config file",
				},
			}
		}
		return nil
	case ProviderTypeGitLab:
		if config.Av.GitLab.Token == "" {
			return ProviderError{
				Provider: ProviderTypeGitLab,
				Message:  "GitLab token not configured",
				Suggestions: []string{
					"Set the AV_GITLAB_TOKEN environment variable",
					"Set the GITLAB_TOKEN environment variable",
					"Add 'gitlab.token' to your av config file",
					"Create a personal access token at https://gitlab.com/-/profile/personal_access_tokens",
				},
			}
		}
		return nil
	default:
		return errors.Errorf("unknown provider type: %s", providerType)
	}
}

func ParseRepositorySlug(slug string) (owner, repo string, err error) {
	parts := strings.Split(strings.TrimSpace(slug), "/")
	if len(parts) != 2 || parts[0] == "" || parts[1] == "" {
		return "", "", errors.Errorf("invalid repository slug format (expected owner/repo): %q", slug)
	}
	return parts[0], parts[1], nil
}