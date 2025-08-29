package vcs

import (
	"context"
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
	if githubURLPattern.MatchString(remoteURL) {
		return ProviderTypeGitHub
	}
	if gitlabURLPattern.MatchString(remoteURL) {
		return ProviderTypeGitLab
	}
	
	return ProviderTypeGitHub
}

func NewProvider(ctx context.Context, providerType ProviderType) (Provider, error) {
	switch providerType {
	case ProviderTypeGitHub:
		if config.Av.GitHub.Token == "" {
			return nil, errors.New("GitHub token not configured")
		}
		return NewGitHubProvider(ctx, config.Av.GitHub.Token)
	case ProviderTypeGitLab:
		if config.Av.GitLab.Token == "" {
			return nil, errors.New("GitLab token not configured")
		}
		return NewGitLabProvider(ctx, config.Av.GitLab.Token, config.Av.GitLab.BaseURL)
	default:
		return nil, errors.Errorf("unknown provider type: %s", providerType)
	}
}

var NewGitHubProvider func(ctx context.Context, token string) (Provider, error)
var NewGitLabProvider func(ctx context.Context, token, baseURL string) (Provider, error)

func DetectAndCreateProvider(ctx context.Context, remoteURL string) (Provider, error) {
	providerType := DetectProviderFromURL(remoteURL)
	return NewProvider(ctx, providerType)
}

func ParseRepositorySlug(slug string) (owner, repo string, err error) {
	parts := strings.Split(strings.TrimSpace(slug), "/")
	if len(parts) != 2 || parts[0] == "" || parts[1] == "" {
		return "", "", errors.Errorf("invalid repository slug format (expected owner/repo): %q", slug)
	}
	return parts[0], parts[1], nil
}