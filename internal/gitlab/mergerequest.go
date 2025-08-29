package gitlab

import (
	"context"
	"strings"

	"emperror.dev/errors"
)

// MergeRequestState represents the possible states of a GitLab merge request
type MergeRequestState string

const (
	MergeRequestStateOpened MergeRequestState = "opened"
	MergeRequestStateClosed MergeRequestState = "closed"
	MergeRequestStateMerged MergeRequestState = "merged"
	MergeRequestStateLocked MergeRequestState = "locked"
)

// MergeRequest represents a GitLab merge request, mirroring GitHub's PullRequest structure
type MergeRequest struct {
	ID                  string
	IID                 int64  // GitLab uses IID (internal ID) similar to GitHub's Number
	Number              int64  // Alias for IID to maintain compatibility
	SourceBranch        string // GitLab equivalent of HeadRefName
	TargetBranch        string // GitLab equivalent of BaseRefName
	State               MergeRequestState
	Draft               bool   // GitLab equivalent of IsDraft
	WebURL              string // GitLab equivalent of Permalink
	Title               string
	Description         string // GitLab uses Description instead of Body
	Body                string // Alias for Description to maintain compatibility
	PRIVATE_MergeCommit struct {
		ID string `json:"id"`
	} `json:"merge_commit_sha"`
	// GitLab doesn't have timeline events in the same way as GitHub,
	// so we'll need to adapt this for GitLab's API structure
	PRIVATE_Events []struct {
		Action    string `json:"action"`
		CreatedAt string `json:"created_at"`
		TargetID  string `json:"target_id"`
	} `json:"resource_state_events"`
}

// HeadBranchName returns the source branch name, normalizing any "refs/heads/" prefix
func (mr *MergeRequest) HeadBranchName() string {
	// GitLab typically returns clean branch names without refs/heads/ prefix,
	// but we'll handle it consistently with GitHub implementation
	return strings.TrimPrefix(mr.SourceBranch, "refs/heads/")
}

// BaseBranchName returns the target branch name, normalizing any "refs/heads/" prefix
func (mr *MergeRequest) BaseBranchName() string {
	// GitLab typically returns clean branch names without refs/heads/ prefix,
	// but we'll handle it consistently with GitHub implementation
	return strings.TrimPrefix(mr.TargetBranch, "refs/heads/")
}

// GetMergeCommit returns the merge commit SHA if the merge request is merged
func (mr *MergeRequest) GetMergeCommit() string {
	if mr.State == MergeRequestStateOpened {
		return ""
	}
	
	if mr.State == MergeRequestStateMerged && mr.PRIVATE_MergeCommit.ID != "" {
		return mr.PRIVATE_MergeCommit.ID
	}

	// For GitLab, we may need to look at resource state events to find merge commits
	// This is a simplified implementation that can be enhanced based on actual GitLab API responses
	for i := len(mr.PRIVATE_Events) - 1; i >= 0; i-- {
		event := mr.PRIVATE_Events[i]
		if event.Action == "merge" && event.TargetID != "" {
			return event.TargetID
		}
	}
	
	return ""
}

// MergeRequestOpts represents options for querying a specific merge request
type MergeRequestOpts struct {
	ProjectID string // GitLab uses project ID instead of owner/repo
	IID       int64  // GitLab's internal ID
}

// MergeRequest retrieves a specific merge request by ID
func (c *Client) MergeRequest(ctx context.Context, id string) (*MergeRequest, error) {
	// This will be implemented with actual GitLab API calls
	// For now, returning a placeholder error
	return nil, errors.New("MergeRequest method not yet implemented - requires GitLab API client setup")
}

// GetMergeRequestsInput represents parameters for querying multiple merge requests
type GetMergeRequestsInput struct {
	// REQUIRED
	ProjectID string // GitLab project ID or path
	// OPTIONAL  
	SourceBranch string                // GitLab equivalent of HeadRefName
	TargetBranch string                // GitLab equivalent of BaseRefName
	State        []MergeRequestState    // Filter by state(s)
	First        int32                  // Pagination limit
	After        string                 // Pagination cursor
}

// GetMergeRequestsPage represents a paginated response of merge requests
type GetMergeRequestsPage struct {
	PageInfo
	MergeRequests []MergeRequest
}

// PageInfo contains pagination information for GitLab API responses
type PageInfo struct {
	EndCursor       string
	HasNextPage     bool
	HasPreviousPage bool
	StartCursor     string
}

// GetMergeRequests retrieves multiple merge requests based on filter criteria
func (c *Client) GetMergeRequests(
	ctx context.Context,
	input GetMergeRequestsInput,
) (*GetMergeRequestsPage, error) {
	if input.First == 0 {
		input.First = 50
	}
	
	// This will be implemented with actual GitLab API calls
	// For now, returning a placeholder error
	return nil, errors.New("GetMergeRequests method not yet implemented - requires GitLab API client setup")
}

// CreateMergeRequestInput represents parameters for creating a new merge request
type CreateMergeRequestInput struct {
	ProjectID    string `json:"id"`
	Title        string `json:"title"`
	Description  string `json:"description"`
	SourceBranch string `json:"source_branch"`
	TargetBranch string `json:"target_branch"`
	Draft        *bool  `json:"draft,omitempty"`
}

// CreateMergeRequest creates a new merge request in GitLab
func (c *Client) CreateMergeRequest(
	ctx context.Context,
	input CreateMergeRequestInput,
) (*MergeRequest, error) {
	// This will be implemented with actual GitLab API calls
	// For now, returning a placeholder error
	return nil, errors.New("CreateMergeRequest method not yet implemented - requires GitLab API client setup")
}

// UpdateMergeRequestInput represents parameters for updating an existing merge request
type UpdateMergeRequestInput struct {
	ProjectID   string  `json:"id"`
	IID         int64   `json:"merge_request_iid"`
	Title       *string `json:"title,omitempty"`
	Description *string `json:"description,omitempty"`
	State       *string `json:"state_event,omitempty"` // close, reopen
	Draft       *bool   `json:"draft,omitempty"`
}

// UpdateMergeRequest updates an existing merge request
func (c *Client) UpdateMergeRequest(
	ctx context.Context,
	input UpdateMergeRequestInput,
) (*MergeRequest, error) {
	// This will be implemented with actual GitLab API calls
	// For now, returning a placeholder error
	return nil, errors.New("UpdateMergeRequest method not yet implemented - requires GitLab API client setup")
}

// RequestReviewsInput represents parameters for requesting reviews on a merge request
type RequestReviewsInput struct {
	ProjectID        string   `json:"id"`
	IID              int64    `json:"merge_request_iid"`
	ReviewerIDs      []int64  `json:"reviewer_ids,omitempty"`
	ReviewerUsernames []string `json:"reviewer_usernames,omitempty"`
}

// RequestReviews requests reviews from specified users on a merge request
func (c *Client) RequestReviews(
	ctx context.Context,
	input RequestReviewsInput,
) (*MergeRequest, error) {
	// This will be implemented with actual GitLab API calls
	// For now, returning a placeholder error
	return nil, errors.New("RequestReviews method not yet implemented - requires GitLab API client setup")
}

// ConvertMergeRequestToDraft marks a merge request as draft
func (c *Client) ConvertMergeRequestToDraft(ctx context.Context, projectID string, iid int64) (*MergeRequest, error) {
	input := UpdateMergeRequestInput{
		ProjectID: projectID,
		IID:       iid,
		Draft:     Ptr(true),
	}
	return c.UpdateMergeRequest(ctx, input)
}

// MarkMergeRequestReadyForReview removes draft status from a merge request
func (c *Client) MarkMergeRequestReadyForReview(
	ctx context.Context,
	projectID string,
	iid int64,
) (*MergeRequest, error) {
	input := UpdateMergeRequestInput{
		ProjectID: projectID,
		IID:       iid,
		Draft:     Ptr(false),
	}
	return c.UpdateMergeRequest(ctx, input)
}

// RepoMergeRequestOpts represents options for querying repository merge requests
type RepoMergeRequestOpts struct {
	ProjectID string
	First     int32
	After     string
	State     []MergeRequestState
}

// RepoMergeRequestsResponse represents the response from querying repository merge requests
type RepoMergeRequestsResponse struct {
	PageInfo
	TotalCount     int64
	MergeRequests  []MergeRequest
}

// RepoMergeRequests retrieves merge requests for a specific repository/project
func (c *Client) RepoMergeRequests(
	ctx context.Context,
	opts RepoMergeRequestOpts,
) (RepoMergeRequestsResponse, error) {
	if opts.First == 0 {
		opts.First = 100
	}
	
	// This will be implemented with actual GitLab API calls
	// For now, returning a placeholder error
	return RepoMergeRequestsResponse{}, errors.New("RepoMergeRequests method not yet implemented - requires GitLab API client setup")
}

// Ptr returns a pointer to the argument - utility function for optional fields
func Ptr[T any](v T) *T {
	return &v
}