package vcs

import (
	"testing"

	"github.com/aviator-co/av/internal/gitlab"
	"github.com/stretchr/testify/assert"
)

func TestGitLabPullRequest_WrapperMethods(t *testing.T) {
	testMR := &gitlab.MergeRequest{
		ID:           "mr-123",
		IID:          42,
		Title:        "Test MR",
		Description:  "Test MR body",
		SourceBranch: "feature-branch",
		TargetBranch: "main",
		Draft:        false,
		State:        gitlab.MergeRequestStateOpened,
		WebURL:       "https://gitlab.com/owner/repo/-/merge_requests/42",
	}
	testMR.Body = testMR.Description // Set compatibility alias
	
	wrapper := &gitlabPullRequest{mr: testMR}
	
	assert.Equal(t, "mr-123", wrapper.GetID())
	assert.Equal(t, int64(42), wrapper.GetNumber())
	assert.Equal(t, "feature-branch", wrapper.HeadBranchName())
	assert.Equal(t, "main", wrapper.BaseBranchName())
	assert.Equal(t, "Test MR", wrapper.GetTitle())
	assert.Equal(t, "Test MR body", wrapper.GetBody())
	assert.Equal(t, "https://gitlab.com/owner/repo/-/merge_requests/42", wrapper.GetPermalink())
	assert.False(t, wrapper.IsDraft())
	assert.True(t, wrapper.IsOpen())
	assert.False(t, wrapper.IsMerged())
}

func TestGitLabPullRequest_HeadBaseBranchNormalization(t *testing.T) {
	testMR := &gitlab.MergeRequest{
		SourceBranch: "refs/heads/feature-branch",
		TargetBranch: "refs/heads/main",
	}
	
	wrapper := &gitlabPullRequest{mr: testMR}
	
	assert.Equal(t, "feature-branch", wrapper.HeadBranchName())
	assert.Equal(t, "main", wrapper.BaseBranchName())
}

func TestGitLabPullRequest_States(t *testing.T) {
	tests := []struct {
		name     string
		state    gitlab.MergeRequestState
		isOpen   bool
		isMerged bool
	}{
		{"opened", gitlab.MergeRequestStateOpened, true, false},
		{"closed", gitlab.MergeRequestStateClosed, false, false},
		{"merged", gitlab.MergeRequestStateMerged, false, true},
	}
	
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			testMR := &gitlab.MergeRequest{State: tt.state}
			wrapper := &gitlabPullRequest{mr: testMR}
			
			assert.Equal(t, tt.isOpen, wrapper.IsOpen())
			assert.Equal(t, tt.isMerged, wrapper.IsMerged())
		})
	}
}

func TestGitLabRepository_WrapperMethods(t *testing.T) {
	testRepo := &gitlab.Repository{
		ID:                123,
		Name:              "test-repo",
		Path:              "test-repo",
		PathWithNamespace: "test-owner/test-repo",
		WebURL:            "https://gitlab.com/test-owner/test-repo",
		Namespace: struct {
			ID       int64  `json:"id"`
			Name     string `json:"name"`
			Path     string `json:"path"`
			Kind     string `json:"kind"`
			FullPath string `json:"full_path"`
		}{
			ID:       456,
			Name:     "Test Owner",
			Path:     "test-owner",
			Kind:     "user",
			FullPath: "test-owner",
		},
	}
	
	wrapper := &gitlabRepository{repo: testRepo}
	
	assert.Equal(t, "123", wrapper.GetID())
	assert.Equal(t, "test-owner", wrapper.GetOwner())
	assert.Equal(t, "test-repo", wrapper.GetName())
	assert.Equal(t, "test-owner/test-repo", wrapper.GetFullName())
}

func TestGitLabUser_WrapperMethods(t *testing.T) {
	testUser := &gitlab.User{
		ID:        123,
		Username:  "test-user",
		Name:      "Test User",
		Email:     "test@example.com",
		AvatarURL: "https://gitlab.com/uploads/user/avatar/123/avatar.png",
		WebURL:    "https://gitlab.com/test-user",
		State:     "active",
	}
	
	wrapper := &gitlabUser{user: testUser}
	
	assert.Equal(t, "123", wrapper.GetID())
	assert.Equal(t, "test-user", wrapper.GetLogin())
}

func TestGitLabUser_NilSafety(t *testing.T) {
	wrapper := &gitlabUser{user: nil}
	
	assert.Equal(t, "", wrapper.GetID())
	assert.Equal(t, "", wrapper.GetLogin())
}

func TestParseGitLabMergeRequestID(t *testing.T) {
	tests := []struct {
		name        string
		id          string
		expectedPID string
		expectedIID int64
		expectError bool
	}{
		{
			name:        "valid project:iid format",
			id:          "123:45",
			expectedPID: "123",
			expectedIID: 45,
			expectError: false,
		},
		{
			name:        "valid string project:iid format",
			id:          "owner/repo:1",
			expectedPID: "owner/repo",
			expectedIID: 1,
			expectError: false,
		},
		{
			name:        "numeric ID without project context",
			id:          "789",
			expectedPID: "",
			expectedIID: 0,
			expectError: true,
		},
		{
			name:        "invalid format",
			id:          "invalid-format",
			expectedPID: "",
			expectedIID: 0,
			expectError: true,
		},
		{
			name:        "invalid IID in project:iid format",
			id:          "123:abc",
			expectedPID: "",
			expectedIID: 0,
			expectError: true,
		},
	}
	
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			projectID, iid, err := parseGitLabMergeRequestID(tt.id)
			
			if tt.expectError {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
				assert.Equal(t, tt.expectedPID, projectID)
				assert.Equal(t, tt.expectedIID, iid)
			}
		})
	}
}

func TestGitLabMergeRequest_GetMergeCommit(t *testing.T) {
	// Test with merge commit SHA
	testMR := &gitlab.MergeRequest{
		State: gitlab.MergeRequestStateMerged,
	}
	testMR.PRIVATE_MergeCommit.ID = "abc123"
	
	wrapper := &gitlabPullRequest{mr: testMR}
	assert.Equal(t, "abc123", wrapper.GetMergeCommit())
	
	// Test open merge request (should return empty string)
	testMR.State = gitlab.MergeRequestStateOpened
	assert.Equal(t, "", wrapper.GetMergeCommit())
}

func TestGitLabProvider_parseGitLabMergeRequestID_EdgeCases(t *testing.T) {
	// Test multiple colons
	projectID, iid, err := parseGitLabMergeRequestID("group/subgroup:project:123")
	assert.Error(t, err)
	assert.Equal(t, "", projectID)
	assert.Equal(t, int64(0), iid)
	
	// Test empty string
	projectID, iid, err = parseGitLabMergeRequestID("")
	assert.Error(t, err)
	assert.Equal(t, "", projectID)
	assert.Equal(t, int64(0), iid)
}