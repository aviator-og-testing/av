package vcs

import (
	"context"
	"strconv"
	"strings"

	"emperror.dev/errors"
	"github.com/aviator-co/av/internal/gh"
	"github.com/shurcooL/githubv4"
)

type GitHubProvider struct {
	client *gh.Client
}

func NewGitHubProvider(ctx context.Context, token string) (Provider, error) {
	client, err := gh.NewClient(ctx, token)
	if err != nil {
		return nil, errors.Wrap(err, "failed to create GitHub client")
	}
	return &GitHubProvider{client: client}, nil
}

type githubPullRequest struct {
	pr *gh.PullRequest
}

func (gpr *githubPullRequest) GetID() string {
	return gpr.pr.ID
}

func (gpr *githubPullRequest) GetNumber() int64 {
	return gpr.pr.Number
}

func (gpr *githubPullRequest) HeadBranchName() string {
	return gpr.pr.HeadBranchName()
}

func (gpr *githubPullRequest) BaseBranchName() string {
	return gpr.pr.BaseBranchName()
}

func (gpr *githubPullRequest) GetTitle() string {
	return gpr.pr.Title
}

func (gpr *githubPullRequest) GetBody() string {
	return gpr.pr.Body
}

func (gpr *githubPullRequest) GetPermalink() string {
	return gpr.pr.Permalink
}

func (gpr *githubPullRequest) IsDraft() bool {
	return gpr.pr.IsDraft
}

func (gpr *githubPullRequest) IsOpen() bool {
	return gpr.pr.State == githubv4.PullRequestStateOpen
}

func (gpr *githubPullRequest) IsMerged() bool {
	return gpr.pr.State == githubv4.PullRequestStateMerged
}

func (gpr *githubPullRequest) GetMergeCommit() string {
	return gpr.pr.GetMergeCommit()
}

type githubRepository struct {
	repo *gh.Repository
}

func (gr *githubRepository) GetID() string {
	return gr.repo.ID
}

func (gr *githubRepository) GetOwner() string {
	return gr.repo.Owner.Login
}

func (gr *githubRepository) GetName() string {
	return gr.repo.Name
}

func (gr *githubRepository) GetFullName() string {
	return gr.repo.Owner.Login + "/" + gr.repo.Name
}

type githubUser struct {
	user *gh.User
	login string
	name string
}

func (gu *githubUser) GetID() string {
	if gu.user != nil {
		return string(gu.user.ID)
	}
	return ""
}

func (gu *githubUser) GetLogin() string {
	if gu.user != nil {
		return gu.user.Login
	}
	return gu.login
}

type githubViewer struct {
	viewer *gh.Viewer
}

func (gv *githubViewer) GetID() string {
	return ""
}

func (gv *githubViewer) GetLogin() string {
	return gv.viewer.Login
}

func (g *GitHubProvider) CreatePR(ctx context.Context, input CreatePullRequestInput) (PullRequest, error) {
	ghInput := githubv4.CreatePullRequestInput{
		RepositoryID: githubv4.ID(input.RepositoryID),
		Title:        githubv4.String(input.Title),
		Body:         gh.Ptr(githubv4.String(input.Body)),
		HeadRefName:  githubv4.String(input.HeadRefName),
		BaseRefName:  githubv4.String(input.BaseRefName),
		Draft:        gh.Ptr(githubv4.Boolean(input.Draft)),
	}
	
	pr, err := g.client.CreatePullRequest(ctx, ghInput)
	if err != nil {
		return nil, errors.Wrap(err, "failed to create pull request")
	}
	
	return &githubPullRequest{pr: pr}, nil
}

func (g *GitHubProvider) GetPR(ctx context.Context, id string) (PullRequest, error) {
	pr, err := g.client.PullRequest(ctx, id)
	if err != nil {
		return nil, errors.Wrap(err, "failed to get pull request")
	}
	
	return &githubPullRequest{pr: pr}, nil
}

func (g *GitHubProvider) UpdatePR(ctx context.Context, input UpdatePullRequestInput) (PullRequest, error) {
	ghInput := githubv4.UpdatePullRequestInput{
		PullRequestID: githubv4.ID(input.ID),
	}
	
	if input.Title != nil {
		ghInput.Title = gh.Ptr(githubv4.String(*input.Title))
	}
	if input.Body != nil {
		ghInput.Body = gh.Ptr(githubv4.String(*input.Body))
	}
	
	pr, err := g.client.UpdatePullRequest(ctx, ghInput)
	if err != nil {
		return nil, errors.Wrap(err, "failed to update pull request")
	}
	
	return &githubPullRequest{pr: pr}, nil
}

func (g *GitHubProvider) GetPRs(ctx context.Context, input GetPullRequestsInput) (*GetPullRequestsPage, error) {
	var states []githubv4.PullRequestState
	for _, state := range input.States {
		switch strings.ToLower(state) {
		case "open":
			states = append(states, githubv4.PullRequestStateOpen)
		case "closed":
			states = append(states, githubv4.PullRequestStateClosed)
		case "merged":
			states = append(states, githubv4.PullRequestStateMerged)
		}
	}
	
	ghInput := gh.GetPullRequestsInput{
		Owner:       input.Owner,
		Repo:        input.Repo,
		HeadRefName: input.HeadRefName,
		BaseRefName: input.BaseRefName,
		States:      states,
		First:       input.First,
		After:       input.After,
	}
	
	result, err := g.client.GetPullRequests(ctx, ghInput)
	if err != nil {
		return nil, errors.Wrap(err, "failed to get pull requests")
	}
	
	var prs []PullRequest
	for i := range result.PullRequests {
		prs = append(prs, &githubPullRequest{pr: &result.PullRequests[i]})
	}
	
	return &GetPullRequestsPage{
		PullRequests: prs,
		HasNextPage:  result.PageInfo.HasNextPage,
		EndCursor:    result.PageInfo.EndCursor,
	}, nil
}

func (g *GitHubProvider) RequestReviews(ctx context.Context, input RequestReviewsInput) (PullRequest, error) {
	var userIDs []githubv4.ID
	for _, userID := range input.UserIDs {
		userIDs = append(userIDs, githubv4.ID(userID))
	}
	
	var teamIDs []githubv4.ID
	for _, teamID := range input.TeamIDs {
		teamIDs = append(teamIDs, githubv4.ID(teamID))
	}
	
	ghInput := githubv4.RequestReviewsInput{
		PullRequestID: githubv4.ID(input.PullRequestID),
		UserIds:       &userIDs,
		TeamIds:       &teamIDs,
		Union:         gh.Ptr(githubv4.Boolean(input.Union)),
	}
	
	pr, err := g.client.RequestReviews(ctx, ghInput)
	if err != nil {
		return nil, errors.Wrap(err, "failed to request reviews")
	}
	
	return &githubPullRequest{pr: pr}, nil
}

func (g *GitHubProvider) ConvertToDraft(ctx context.Context, id string) (PullRequest, error) {
	pr, err := g.client.ConvertPullRequestToDraft(ctx, id)
	if err != nil {
		return nil, errors.Wrap(err, "failed to convert pull request to draft")
	}
	
	return &githubPullRequest{pr: pr}, nil
}

func (g *GitHubProvider) MarkReadyForReview(ctx context.Context, id string) (PullRequest, error) {
	pr, err := g.client.MarkPullRequestReadyForReview(ctx, id)
	if err != nil {
		return nil, errors.Wrap(err, "failed to mark pull request ready for review")
	}
	
	return &githubPullRequest{pr: pr}, nil
}

func (g *GitHubProvider) GetRepository(ctx context.Context, owner, repo string) (Repository, error) {
	slug := owner + "/" + repo
	ghRepo, err := g.client.GetRepositoryBySlug(ctx, slug)
	if err != nil {
		return nil, errors.Wrap(err, "failed to get repository")
	}
	
	return &githubRepository{repo: ghRepo}, nil
}

func (g *GitHubProvider) GetUser(ctx context.Context, login string) (User, error) {
	user, err := g.client.User(ctx, login)
	if err != nil {
		return nil, errors.Wrap(err, "failed to get user")
	}
	
	return &githubUser{user: user}, nil
}

func (g *GitHubProvider) GetViewer(ctx context.Context) (User, error) {
	viewer, err := g.client.Viewer(ctx)
	if err != nil {
		return nil, errors.Wrap(err, "failed to get viewer")
	}
	
	return &githubViewer{viewer: viewer}, nil
}

func init() {
	NewGitHubProvider = func(ctx context.Context, token string) (Provider, error) {
		client, err := gh.NewClient(ctx, token)
		if err != nil {
			return nil, errors.Wrap(err, "failed to create GitHub client")
		}
		return &GitHubProvider{client: client}, nil
	}
}