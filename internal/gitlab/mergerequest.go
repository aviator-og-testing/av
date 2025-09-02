package gitlab

import (
	"context"
	"strconv"
	"strings"

	"emperror.dev/errors"
	"github.com/xanzy/go-gitlab"
)

// MergeRequest represents a GitLab merge request with fields equivalent to GitHub's PullRequest
type MergeRequest struct {
	ID              int
	IID             int
	Number          int64 // Alias for IID to match GitHub's Number field
	SourceBranch    string
	TargetBranch    string
	State           string
	Title           string
	Description     string
	WebURL          string
	IsDraft         bool
	MergeCommitSHA  string
	MergedBy        *gitlab.BasicUser
	MergeStatus     string
	Author          *gitlab.BasicUser
	CreatedAt       *gitlab.Time
	UpdatedAt       *gitlab.Time
	MergedAt        *gitlab.Time
}

// HeadBranchName returns the source branch name with refs/heads/ prefix removed
func (mr *MergeRequest) HeadBranchName() string {
	return strings.TrimPrefix(mr.SourceBranch, "refs/heads/")
}

// BaseBranchName returns the target branch name with refs/heads/ prefix removed  
func (mr *MergeRequest) BaseBranchName() string {
	return strings.TrimPrefix(mr.TargetBranch, "refs/heads/")
}

// GetMergeCommit returns the merge commit SHA if the merge request is merged
func (mr *MergeRequest) GetMergeCommit() string {
	if mr.State != "merged" {
		return ""
	}
	return mr.MergeCommitSHA
}

// convertFromGitLabMR converts a GitLab MergeRequest to our internal MergeRequest type
func convertFromGitLabMR(glMR *gitlab.MergeRequest) *MergeRequest {
	mr := &MergeRequest{
		ID:              glMR.ID,
		IID:             glMR.IID,
		Number:          int64(glMR.IID),
		SourceBranch:    glMR.SourceBranch,
		TargetBranch:    glMR.TargetBranch,
		State:           glMR.State,
		Title:           glMR.Title,
		Description:     glMR.Description,
		WebURL:          glMR.WebURL,
		MergeCommitSHA:  glMR.MergeCommitSHA,
		MergedBy:        glMR.MergedBy,
		MergeStatus:     glMR.MergeStatus,
		Author:          glMR.Author,
		CreatedAt:       glMR.CreatedAt,
		UpdatedAt:       glMR.UpdatedAt,
		MergedAt:        glMR.MergedAt,
	}
	
	// GitLab uses "draft" prefix in title or work_in_progress flag
	mr.IsDraft = glMR.WorkInProgress || strings.HasPrefix(strings.ToLower(glMR.Title), "draft:")
	
	return mr
}

// MergeRequestOpts represents options for fetching a specific merge request
type MergeRequestOpts struct {
	ProjectID string
	IID       int
}

// MergeRequest fetches a specific merge request by project and IID
func (c *Client) MergeRequest(ctx context.Context, projectID string, iid int) (*MergeRequest, error) {
	result, err := c.query(ctx, "GetMergeRequest", func() (interface{}, *gitlab.Response, error) {
		return c.gl.MergeRequests.GetMergeRequest(projectID, iid, nil, gitlab.WithContext(ctx))
	})
	if err != nil {
		return nil, errors.Wrapf(err, "failed to get merge request %d in project %s", iid, projectID)
	}
	
	glMR, ok := result.(*gitlab.MergeRequest)
	if !ok {
		return nil, errors.Errorf("unexpected result type from GitLab API")
	}
	
	return convertFromGitLabMR(glMR), nil
}

// GetMergeRequestsInput represents input parameters for listing merge requests
type GetMergeRequestsInput struct {
	ProjectID    string
	SourceBranch string
	TargetBranch string
	State        string
	PerPage      int
	Page         int
}

// GetMergeRequestsPage represents a page of merge requests with pagination info
type GetMergeRequestsPage struct {
	MergeRequests []MergeRequest
	NextPage      int
	PrevPage      int
	TotalPages    int
	TotalItems    int
}

// GetMergeRequests fetches a list of merge requests with optional filtering
func (c *Client) GetMergeRequests(ctx context.Context, input GetMergeRequestsInput) (*GetMergeRequestsPage, error) {
	opts := &gitlab.ListProjectMergeRequestsOptions{
		ListOptions: gitlab.ListOptions{
			PerPage: input.PerPage,
			Page:    input.Page,
		},
	}
	
	if input.SourceBranch != "" {
		opts.SourceBranch = &input.SourceBranch
	}
	if input.TargetBranch != "" {
		opts.TargetBranch = &input.TargetBranch  
	}
	if input.State != "" {
		opts.State = &input.State
	}
	
	if opts.PerPage == 0 {
		opts.PerPage = 50
	}
	
	result, err := c.query(ctx, "ListProjectMergeRequests", func() (interface{}, *gitlab.Response, error) {
		return c.gl.MergeRequests.ListProjectMergeRequests(input.ProjectID, opts, gitlab.WithContext(ctx))
	})
	if err != nil {
		return nil, errors.Wrapf(err, "failed to list merge requests for project %s", input.ProjectID)
	}
	
	glMRs, ok := result.([]*gitlab.MergeRequest)
	if !ok {
		return nil, errors.Errorf("unexpected result type from GitLab API")
	}
	
	mrs := make([]MergeRequest, len(glMRs))
	for i, glMR := range glMRs {
		mrs[i] = *convertFromGitLabMR(glMR)
	}
	
	return &GetMergeRequestsPage{
		MergeRequests: mrs,
		// Note: GitLab response headers contain pagination info
		// This would need to be extracted from the response in a real implementation
		NextPage:   0, // TODO: Extract from response headers
		PrevPage:   0, // TODO: Extract from response headers  
		TotalPages: 0, // TODO: Extract from response headers
		TotalItems: 0, // TODO: Extract from response headers
	}, nil
}

// CreateMergeRequestInput represents input for creating a merge request
type CreateMergeRequestInput struct {
	ProjectID            string
	Title                string
	Description          string
	SourceBranch         string
	TargetBranch         string
	AssigneeID           *int
	AssigneeIDs          []int
	ReviewerIDs          []int
	TargetProjectID      *int
	Labels               []string
	MilestoneID          *int
	RemoveSourceBranch   *bool
	AllowCollaboration   *bool
	Squash               *bool
}

// CreateMergeRequest creates a new merge request
func (c *Client) CreateMergeRequest(ctx context.Context, input CreateMergeRequestInput) (*MergeRequest, error) {
	opts := &gitlab.CreateMergeRequestOptions{
		Title:        &input.Title,
		Description:  &input.Description,  
		SourceBranch: &input.SourceBranch,
		TargetBranch: &input.TargetBranch,
	}
	
	if input.AssigneeID != nil {
		opts.AssigneeID = input.AssigneeID
	}
	if len(input.AssigneeIDs) > 0 {
		opts.AssigneeIDs = &input.AssigneeIDs
	}
	if len(input.ReviewerIDs) > 0 {
		opts.ReviewerIDs = &input.ReviewerIDs
	}
	if input.TargetProjectID != nil {
		opts.TargetProjectID = input.TargetProjectID
	}
	if len(input.Labels) > 0 {
		labels := gitlab.LabelOptions(input.Labels)
		opts.Labels = &labels
	}
	if input.MilestoneID != nil {
		opts.MilestoneID = input.MilestoneID
	}
	if input.RemoveSourceBranch != nil {
		opts.RemoveSourceBranch = input.RemoveSourceBranch
	}
	if input.AllowCollaboration != nil {
		opts.AllowCollaboration = input.AllowCollaboration
	}
	if input.Squash != nil {
		opts.Squash = input.Squash
	}
	
	result, err := c.mutate(ctx, "CreateMergeRequest", func() (interface{}, *gitlab.Response, error) {
		return c.gl.MergeRequests.CreateMergeRequest(input.ProjectID, opts, gitlab.WithContext(ctx))
	})
	if err != nil {
		return nil, errors.Wrapf(err, "failed to create merge request in project %s", input.ProjectID)
	}
	
	glMR, ok := result.(*gitlab.MergeRequest)
	if !ok {
		return nil, errors.Errorf("unexpected result type from GitLab API")
	}
	
	return convertFromGitLabMR(glMR), nil
}

// UpdateMergeRequestInput represents input for updating a merge request
type UpdateMergeRequestInput struct {
	ProjectID           string
	IID                 int
	Title               *string
	Description         *string  
	TargetBranch        *string
	AssigneeID          *int
	AssigneeIDs         *[]int
	ReviewerIDs         *[]int
	Labels              *[]string
	MilestoneID         *int
	StateEvent          *string
	RemoveSourceBranch  *bool
	Squash              *bool
	DiscussionLocked    *bool
	AllowCollaboration  *bool
}

// UpdateMergeRequest updates an existing merge request
func (c *Client) UpdateMergeRequest(ctx context.Context, input UpdateMergeRequestInput) (*MergeRequest, error) {
	opts := &gitlab.UpdateMergeRequestOptions{}
	
	if input.Title != nil {
		opts.Title = input.Title
	}
	if input.Description != nil {
		opts.Description = input.Description
	}
	if input.TargetBranch != nil {
		opts.TargetBranch = input.TargetBranch
	}
	if input.AssigneeID != nil {
		opts.AssigneeID = input.AssigneeID
	}
	if input.AssigneeIDs != nil {
		opts.AssigneeIDs = input.AssigneeIDs
	}
	if input.ReviewerIDs != nil {
		opts.ReviewerIDs = input.ReviewerIDs
	}
	if input.Labels != nil {
		labels := gitlab.LabelOptions(*input.Labels)
		opts.Labels = &labels
	}
	if input.MilestoneID != nil {
		opts.MilestoneID = input.MilestoneID
	}
	if input.StateEvent != nil {
		opts.StateEvent = input.StateEvent
	}
	if input.RemoveSourceBranch != nil {
		opts.RemoveSourceBranch = input.RemoveSourceBranch
	}
	if input.Squash != nil {
		opts.Squash = input.Squash
	}
	if input.DiscussionLocked != nil {
		opts.DiscussionLocked = input.DiscussionLocked
	}
	if input.AllowCollaboration != nil {
		opts.AllowCollaboration = input.AllowCollaboration
	}
	
	result, err := c.mutate(ctx, "UpdateMergeRequest", func() (interface{}, *gitlab.Response, error) {
		return c.gl.MergeRequests.UpdateMergeRequest(input.ProjectID, input.IID, opts, gitlab.WithContext(ctx))
	})
	if err != nil {
		return nil, errors.Wrapf(err, "failed to update merge request %d in project %s", input.IID, input.ProjectID)
	}
	
	glMR, ok := result.(*gitlab.MergeRequest)
	if !ok {
		return nil, errors.Errorf("unexpected result type from GitLab API")
	}
	
	return convertFromGitLabMR(glMR), nil
}