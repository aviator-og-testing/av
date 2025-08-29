package vcs

import (
	"context"
	"strconv"
	"strings"

	"emperror.dev/errors"
	"github.com/aviator-co/av/internal/gitlab"
)

// GitLabProvider implements the Provider interface for GitLab repositories
// It acts as an adapter between the common VCS interface and GitLab-specific types
type GitLabProvider struct {
	client *gitlab.Client
}

func NewGitLabProvider(ctx context.Context, token, baseURL string) (Provider, error) {
	client, err := gitlab.NewClient(ctx, token, baseURL)
	if err != nil {
		return nil, errors.Wrap(err, "failed to create GitLab client")
	}
	return &GitLabProvider{client: client}, nil
}

type gitlabPullRequest struct {
	mr *gitlab.MergeRequest
}

func (gpr *gitlabPullRequest) GetID() string {
	return gpr.mr.ID
}

func (gpr *gitlabPullRequest) GetNumber() int64 {
	return gpr.mr.IID
}

func (gpr *gitlabPullRequest) HeadBranchName() string {
	return gpr.mr.HeadBranchName()
}

func (gpr *gitlabPullRequest) BaseBranchName() string {
	return gpr.mr.BaseBranchName()
}

func (gpr *gitlabPullRequest) GetTitle() string {
	return gpr.mr.Title
}

func (gpr *gitlabPullRequest) GetBody() string {
	if gpr.mr.Body != "" {
		return gpr.mr.Body
	}
	return gpr.mr.Description
}

func (gpr *gitlabPullRequest) GetPermalink() string {
	return gpr.mr.WebURL
}

func (gpr *gitlabPullRequest) IsDraft() bool {
	return gpr.mr.Draft
}

func (gpr *gitlabPullRequest) IsOpen() bool {
	return gpr.mr.State == gitlab.MergeRequestStateOpened
}

func (gpr *gitlabPullRequest) IsMerged() bool {
	return gpr.mr.State == gitlab.MergeRequestStateMerged
}

func (gpr *gitlabPullRequest) GetMergeCommit() string {
	return gpr.mr.GetMergeCommit()
}

type gitlabRepository struct {
	repo *gitlab.Repository
}

func (gr *gitlabRepository) GetID() string {
	return strconv.FormatInt(gr.repo.ID, 10)
}

func (gr *gitlabRepository) GetOwner() string {
	return gr.repo.Owner()
}

func (gr *gitlabRepository) GetName() string {
	return gr.repo.Name
}

func (gr *gitlabRepository) GetFullName() string {
	return gr.repo.FullName()
}

type gitlabUser struct {
	user *gitlab.User
}

func (gu *gitlabUser) GetID() string {
	if gu.user != nil {
		return strconv.FormatInt(gu.user.ID, 10)
	}
	return ""
}

func (gu *gitlabUser) GetLogin() string {
	if gu.user != nil {
		return gu.user.Username
	}
	return ""
}

type gitlabTeam struct {
	team *gitlab.Team
}

func (gt *gitlabTeam) GetID() string {
	if gt.team != nil {
		return strconv.FormatInt(gt.team.ID, 10)
	}
	return ""
}

func (gt *gitlabTeam) GetName() string {
	if gt.team != nil {
		return gt.team.Name
	}
	return ""
}

func (g *GitLabProvider) CreatePR(ctx context.Context, input CreatePullRequestInput) (PullRequest, error) {
	glInput := gitlab.CreateMergeRequestInput{
		ProjectID:    input.RepositoryID,
		Title:        input.Title,
		Description:  input.Body,
		SourceBranch: input.HeadRefName,
		TargetBranch: input.BaseRefName,
		Draft:        &input.Draft,
	}
	
	mr, err := g.client.CreateMergeRequest(ctx, glInput)
	if err != nil {
		return nil, errors.Wrap(err, "failed to create merge request")
	}
	
	return &gitlabPullRequest{mr: mr}, nil
}

func (g *GitLabProvider) GetPR(ctx context.Context, id string) (PullRequest, error) {
	mr, err := g.client.MergeRequest(ctx, id)
	if err != nil {
		return nil, errors.Wrap(err, "failed to get merge request")
	}
	
	return &gitlabPullRequest{mr: mr}, nil
}

func (g *GitLabProvider) UpdatePR(ctx context.Context, input UpdatePullRequestInput) (PullRequest, error) {
	// Parse project ID and merge request IID from the ID
	// GitLab IDs are typically in format "projectID:iid" or just the global ID
	projectID, iid, err := parseGitLabMergeRequestID(input.ID)
	if err != nil {
		return nil, errors.Wrap(err, "failed to parse merge request ID")
	}
	
	glInput := gitlab.UpdateMergeRequestInput{
		ProjectID: projectID,
		IID:       iid,
	}
	
	if input.Title != nil {
		glInput.Title = input.Title
	}
	if input.Body != nil {
		glInput.Description = input.Body
	}
	if input.Draft != nil {
		glInput.Draft = input.Draft
	}
	
	mr, err := g.client.UpdateMergeRequest(ctx, glInput)
	if err != nil {
		return nil, errors.Wrap(err, "failed to update merge request")
	}
	
	return &gitlabPullRequest{mr: mr}, nil
}

func (g *GitLabProvider) GetPRs(ctx context.Context, input GetPullRequestsInput) (*GetPullRequestsPage, error) {
	var states []gitlab.MergeRequestState
	for _, state := range input.States {
		switch strings.ToLower(state) {
		case "open":
			states = append(states, gitlab.MergeRequestStateOpened)
		case "closed":
			states = append(states, gitlab.MergeRequestStateClosed)
		case "merged":
			states = append(states, gitlab.MergeRequestStateMerged)
		}
	}
	
	// Build project ID from owner/repo
	projectID := input.Owner + "/" + input.Repo
	
	glInput := gitlab.GetMergeRequestsInput{
		ProjectID:    projectID,
		SourceBranch: input.HeadRefName,
		TargetBranch: input.BaseRefName,
		State:        states,
		First:        input.First,
		After:        input.After,
	}
	
	result, err := g.client.GetMergeRequests(ctx, glInput)
	if err != nil {
		return nil, errors.Wrap(err, "failed to get merge requests")
	}
	
	var prs []PullRequest
	for i := range result.MergeRequests {
		prs = append(prs, &gitlabPullRequest{mr: &result.MergeRequests[i]})
	}
	
	return &GetPullRequestsPage{
		PullRequests: prs,
		HasNextPage:  result.HasNextPage,
		EndCursor:    result.EndCursor,
	}, nil
}

func (g *GitLabProvider) RequestReviews(ctx context.Context, input RequestReviewsInput) (PullRequest, error) {
	// Parse project ID and merge request IID from the PullRequestID
	projectID, iid, err := parseGitLabMergeRequestID(input.PullRequestID)
	if err != nil {
		return nil, errors.Wrap(err, "failed to parse merge request ID")
	}
	
	// Convert user ID strings to int64 for GitLab API
	var reviewerIDs []int64
	for _, userID := range input.UserIDs {
		if id, parseErr := strconv.ParseInt(userID, 10, 64); parseErr == nil {
			reviewerIDs = append(reviewerIDs, id)
		}
	}
	
	// GitLab doesn't have team reviews in the same way as GitHub
	// Teams would need to be handled differently, possibly through group mentions or similar
	if len(input.TeamIDs) > 0 {
		// For now, we'll skip team assignments and just log a warning
		// This could be enhanced to handle GitLab groups or similar functionality
	}
	
	glInput := gitlab.RequestReviewsInput{
		ProjectID:   projectID,
		IID:         iid,
		ReviewerIDs: reviewerIDs,
	}
	
	mr, err := g.client.RequestReviews(ctx, glInput)
	if err != nil {
		return nil, errors.Wrap(err, "failed to request reviews")
	}
	
	return &gitlabPullRequest{mr: mr}, nil
}

func (g *GitLabProvider) ConvertToDraft(ctx context.Context, id string) (PullRequest, error) {
	projectID, iid, err := parseGitLabMergeRequestID(id)
	if err != nil {
		return nil, errors.Wrap(err, "failed to parse merge request ID")
	}
	
	mr, err := g.client.ConvertMergeRequestToDraft(ctx, projectID, iid)
	if err != nil {
		return nil, errors.Wrap(err, "failed to convert merge request to draft")
	}
	
	return &gitlabPullRequest{mr: mr}, nil
}

func (g *GitLabProvider) MarkReadyForReview(ctx context.Context, id string) (PullRequest, error) {
	projectID, iid, err := parseGitLabMergeRequestID(id)
	if err != nil {
		return nil, errors.Wrap(err, "failed to parse merge request ID")
	}
	
	mr, err := g.client.MarkMergeRequestReadyForReview(ctx, projectID, iid)
	if err != nil {
		return nil, errors.Wrap(err, "failed to mark merge request ready for review")
	}
	
	return &gitlabPullRequest{mr: mr}, nil
}

func (g *GitLabProvider) GetRepository(ctx context.Context, owner, repo string) (Repository, error) {
	slug := owner + "/" + repo
	glRepo, err := g.client.GetRepositoryBySlug(ctx, slug)
	if err != nil {
		return nil, errors.Wrap(err, "failed to get repository")
	}
	
	return &gitlabRepository{repo: glRepo}, nil
}

func (g *GitLabProvider) GetUser(ctx context.Context, login string) (User, error) {
	user, err := g.client.GetUser(ctx, login)
	if err != nil {
		return nil, errors.Wrap(err, "failed to get user")
	}
	
	return &gitlabUser{user: user}, nil
}

func (g *GitLabProvider) GetViewer(ctx context.Context) (User, error) {
	user, err := g.client.GetViewer(ctx)
	if err != nil {
		return nil, errors.Wrap(err, "failed to get viewer")
	}
	
	return &gitlabUser{user: user}, nil
}

func (g *GitLabProvider) GetOrganizationTeam(ctx context.Context, org, team string) (Team, error) {
	// In GitLab, teams are groups and the org/team pattern maps to group paths
	// We build the group path as "org/team" to match GitLab's hierarchical structure
	groupPath := org + "/" + team
	
	glTeam, err := g.client.GetTeam(ctx, groupPath)
	if err != nil {
		return nil, errors.Wrap(err, "failed to get organization team")
	}
	
	return &gitlabTeam{team: glTeam}, nil
}

// parseGitLabMergeRequestID parses a GitLab merge request ID to extract project ID and IID
// This is a helper function to handle the different ID formats GitLab might use
func parseGitLabMergeRequestID(id string) (projectID string, iid int64, err error) {
	if id == "" {
		return "", 0, errors.New("merge request ID cannot be empty")
	}
	
	// Try to parse different formats:
	// 1. "projectID:iid" format
	if strings.Contains(id, ":") {
		parts := strings.Split(id, ":")
		if len(parts) == 2 && parts[0] != "" && parts[1] != "" {
			projectID = parts[0]
			iid, err = strconv.ParseInt(parts[1], 10, 64)
			if err != nil {
				return "", 0, errors.Wrapf(err, "invalid IID format in ID %s", id)
			}
			return projectID, iid, nil
		} else {
			return "", 0, errors.Errorf("invalid merge request ID format: %s (expected projectID:iid)", id)
		}
	}
	
	// 2. If it's just a numeric ID, we need additional context to determine project
	// This is a limitation - we might need to store project context differently
	// For now, return an error indicating we need more context
	if _, parseErr := strconv.ParseInt(id, 10, 64); parseErr == nil {
		return "", 0, errors.Errorf("merge request ID %s lacks project context - expected format: projectID:iid", id)
	}
	
	// 3. If it's a string ID, assume it might be a global ID or path-based identifier
	return id, 0, errors.Errorf("unsupported merge request ID format: %s", id)
}

func init() {
	NewGitLabProvider = func(ctx context.Context, token, baseURL string) (Provider, error) {
		client, err := gitlab.NewClient(ctx, token, baseURL)
		if err != nil {
			return nil, errors.Wrap(err, "failed to create GitLab client")
		}
		return &GitLabProvider{client: client}, nil
	}
}