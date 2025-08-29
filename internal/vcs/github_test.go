package vcs

import (
	"testing"

	"github.com/aviator-co/av/internal/gh"
	"github.com/shurcooL/githubv4"
	"github.com/stretchr/testify/assert"
)

func TestGitHubPullRequest_WrapperMethods(t *testing.T) {
	testPR := &gh.PullRequest{
		ID:          "pr-123",
		Number:      42,
		HeadRefName: "feature-branch",
		BaseRefName: "main",
		Title:       "Test PR",
		Body:        "Test PR body",
		Permalink:   "https://github.com/owner/repo/pull/42",
		IsDraft:     false,
		State:       githubv4.PullRequestStateOpen,
	}
	
	wrapper := &githubPullRequest{pr: testPR}
	
	assert.Equal(t, "pr-123", wrapper.GetID())
	assert.Equal(t, int64(42), wrapper.GetNumber())
	assert.Equal(t, "feature-branch", wrapper.HeadBranchName())
	assert.Equal(t, "main", wrapper.BaseBranchName())
	assert.Equal(t, "Test PR", wrapper.GetTitle())
	assert.Equal(t, "Test PR body", wrapper.GetBody())
	assert.Equal(t, "https://github.com/owner/repo/pull/42", wrapper.GetPermalink())
	assert.False(t, wrapper.IsDraft())
	assert.True(t, wrapper.IsOpen())
	assert.False(t, wrapper.IsMerged())
}

func TestGitHubRepository_WrapperMethods(t *testing.T) {
	testRepo := &gh.Repository{
		ID:   "repo-123",
		Name: "test-repo",
		Owner: struct{ Login string }{
			Login: "test-owner",
		},
	}
	
	wrapper := &githubRepository{repo: testRepo}
	
	assert.Equal(t, "repo-123", wrapper.GetID())
	assert.Equal(t, "test-owner", wrapper.GetOwner())
	assert.Equal(t, "test-repo", wrapper.GetName())
	assert.Equal(t, "test-owner/test-repo", wrapper.GetFullName())
}

func TestGitHubUser_WrapperMethods(t *testing.T) {
	testUser := &gh.User{
		ID:    githubv4.ID("user-123"),
		Login: "test-user",
	}
	
	wrapper := &githubUser{user: testUser}
	
	assert.Equal(t, "user-123", wrapper.GetID())
	assert.Equal(t, "test-user", wrapper.GetLogin())
}

func TestGitHubViewer_WrapperMethods(t *testing.T) {
	testViewer := &gh.Viewer{
		Name:  "Test User",
		Login: "test-user",
	}
	
	wrapper := &githubViewer{viewer: testViewer}
	
	assert.Equal(t, "", wrapper.GetID())
	assert.Equal(t, "test-user", wrapper.GetLogin())
}

func TestGitHubPullRequest_StateChecks(t *testing.T) {
	tests := []struct {
		name     string
		state    githubv4.PullRequestState
		isOpen   bool
		isMerged bool
	}{
		{
			name:     "Open PR",
			state:    githubv4.PullRequestStateOpen,
			isOpen:   true,
			isMerged: false,
		},
		{
			name:     "Merged PR",
			state:    githubv4.PullRequestStateMerged,
			isOpen:   false,
			isMerged: true,
		},
		{
			name:     "Closed PR",
			state:    githubv4.PullRequestStateClosed,
			isOpen:   false,
			isMerged: false,
		},
	}
	
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			pr := &githubPullRequest{
				pr: &gh.PullRequest{
					State: tt.state,
				},
			}
			
			assert.Equal(t, tt.isOpen, pr.IsOpen())
			assert.Equal(t, tt.isMerged, pr.IsMerged())
		})
	}
}