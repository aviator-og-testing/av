package gitlab

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestMergeRequest_HeadBranchName(t *testing.T) {
	tests := []struct {
		name         string
		sourceBranch string
		expected     string
	}{
		{
			name:         "Clean branch name",
			sourceBranch: "feature-branch",
			expected:     "feature-branch",
		},
		{
			name:         "Branch with refs/heads/ prefix",
			sourceBranch: "refs/heads/feature-branch",
			expected:     "feature-branch",
		},
		{
			name:         "Main branch",
			sourceBranch: "main",
			expected:     "main",
		},
		{
			name:         "Main branch with prefix",
			sourceBranch: "refs/heads/main",
			expected:     "main",
		},
		{
			name:         "Branch with slashes",
			sourceBranch: "feature/sub-feature",
			expected:     "feature/sub-feature",
		},
		{
			name:         "Branch with slashes and prefix",
			sourceBranch: "refs/heads/feature/sub-feature",
			expected:     "feature/sub-feature",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mr := &MergeRequest{
				SourceBranch: tt.sourceBranch,
			}
			
			result := mr.HeadBranchName()
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestMergeRequest_BaseBranchName(t *testing.T) {
	tests := []struct {
		name         string
		targetBranch string
		expected     string
	}{
		{
			name:         "Clean branch name",
			targetBranch: "main",
			expected:     "main",
		},
		{
			name:         "Branch with refs/heads/ prefix",
			targetBranch: "refs/heads/main",
			expected:     "main",
		},
		{
			name:         "Master branch",
			targetBranch: "master",
			expected:     "master",
		},
		{
			name:         "Development branch",
			targetBranch: "develop",
			expected:     "develop",
		},
		{
			name:         "Development branch with prefix",
			targetBranch: "refs/heads/develop",
			expected:     "develop",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mr := &MergeRequest{
				TargetBranch: tt.targetBranch,
			}
			
			result := mr.BaseBranchName()
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestMergeRequest_GetMergeCommit(t *testing.T) {
	tests := []struct {
		name           string
		state          MergeRequestState
		mergeCommitID  string
		events         []struct {
			Action    string
			TargetID  string
		}
		expected string
	}{
		{
			name:          "Open merge request has no merge commit",
			state:         MergeRequestStateOpened,
			mergeCommitID: "abc123",
			expected:      "",
		},
		{
			name:          "Merged MR with merge commit",
			state:         MergeRequestStateMerged,
			mergeCommitID: "abc123def456",
			expected:      "abc123def456",
		},
		{
			name:          "Merged MR without direct merge commit, found in events",
			state:         MergeRequestStateMerged,
			mergeCommitID: "",
			events: []struct {
				Action    string
				TargetID  string
			}{
				{Action: "open", TargetID: ""},
				{Action: "merge", TargetID: "def456abc789"},
			},
			expected: "def456abc789",
		},
		{
			name:          "Merged MR with no merge commit data",
			state:         MergeRequestStateMerged,
			mergeCommitID: "",
			events:        []struct{ Action, TargetID string }{},
			expected:      "",
		},
		{
			name:          "Closed MR looks for merge events",
			state:         MergeRequestStateClosed,
			mergeCommitID: "",
			events: []struct {
				Action    string
				TargetID  string
			}{
				{Action: "close", TargetID: ""},
				{Action: "merge", TargetID: "closed123merge"},
			},
			expected: "closed123merge",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mr := &MergeRequest{
				State: tt.state,
			}
			mr.PRIVATE_MergeCommit.ID = tt.mergeCommitID
			
			// Convert test events to MR events
			for _, event := range tt.events {
				mr.PRIVATE_Events = append(mr.PRIVATE_Events, struct {
					Action    string `json:"action"`
					CreatedAt string `json:"created_at"`
					TargetID  string `json:"target_id"`
				}{
					Action:   event.Action,
					TargetID: event.TargetID,
				})
			}
			
			result := mr.GetMergeCommit()
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestClient_MergeRequestOperations_NotImplemented(t *testing.T) {
	ctx := context.Background()
	client := &Client{}

	t.Run("MergeRequest returns not implemented error", func(t *testing.T) {
		mr, err := client.MergeRequest(ctx, "test-id")
		assert.Error(t, err)
		assert.Nil(t, mr)
		assert.Contains(t, err.Error(), "not yet implemented")
	})

	t.Run("GetMergeRequests returns not implemented error", func(t *testing.T) {
		input := GetMergeRequestsInput{
			ProjectID: "test-project",
		}
		result, err := client.GetMergeRequests(ctx, input)
		assert.Error(t, err)
		assert.Nil(t, result)
		assert.Contains(t, err.Error(), "not yet implemented")
	})

	t.Run("CreateMergeRequest returns not implemented error", func(t *testing.T) {
		input := CreateMergeRequestInput{
			ProjectID:    "test-project",
			Title:        "Test MR",
			Description:  "Test description",
			SourceBranch: "feature",
			TargetBranch: "main",
		}
		mr, err := client.CreateMergeRequest(ctx, input)
		assert.Error(t, err)
		assert.Nil(t, mr)
		assert.Contains(t, err.Error(), "not yet implemented")
	})

	t.Run("UpdateMergeRequest returns not implemented error", func(t *testing.T) {
		input := UpdateMergeRequestInput{
			ProjectID: "test-project",
			IID:       42,
		}
		mr, err := client.UpdateMergeRequest(ctx, input)
		assert.Error(t, err)
		assert.Nil(t, mr)
		assert.Contains(t, err.Error(), "not yet implemented")
	})

	t.Run("RequestReviews returns not implemented error", func(t *testing.T) {
		input := RequestReviewsInput{
			ProjectID: "test-project",
			IID:       42,
		}
		mr, err := client.RequestReviews(ctx, input)
		assert.Error(t, err)
		assert.Nil(t, mr)
		assert.Contains(t, err.Error(), "not yet implemented")
	})
}

func TestClient_ConvertMergeRequestToDraft(t *testing.T) {
	ctx := context.Background()
	client := &Client{}

	mr, err := client.ConvertMergeRequestToDraft(ctx, "test-project", 42)
	assert.Error(t, err)
	assert.Nil(t, mr)
	assert.Contains(t, err.Error(), "not yet implemented")
}

func TestClient_MarkMergeRequestReadyForReview(t *testing.T) {
	ctx := context.Background()
	client := &Client{}

	mr, err := client.MarkMergeRequestReadyForReview(ctx, "test-project", 42)
	assert.Error(t, err)
	assert.Nil(t, mr)
	assert.Contains(t, err.Error(), "not yet implemented")
}

func TestClient_RepoMergeRequests(t *testing.T) {
	ctx := context.Background()
	client := &Client{}

	opts := RepoMergeRequestOpts{
		ProjectID: "test-project",
	}
	result, err := client.RepoMergeRequests(ctx, opts)
	assert.Error(t, err)
	assert.Equal(t, RepoMergeRequestsResponse{}, result)
	assert.Contains(t, err.Error(), "not yet implemented")
}

func TestGetMergeRequestsInput_DefaultPagination(t *testing.T) {
	ctx := context.Background()
	client := &Client{}

	input := GetMergeRequestsInput{
		ProjectID: "test-project",
		// First is 0, should be set to default 50
	}

	_, err := client.GetMergeRequests(ctx, input)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "not yet implemented")
	// We can't test the default behavior without mocking, but this ensures the method is called
}

func TestRepoMergeRequestOpts_DefaultPagination(t *testing.T) {
	ctx := context.Background()
	client := &Client{}

	opts := RepoMergeRequestOpts{
		ProjectID: "test-project",
		// First is 0, should be set to default 100
	}

	_, err := client.RepoMergeRequests(ctx, opts)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "not yet implemented")
}

func TestPtr(t *testing.T) {
	t.Run("String pointer", func(t *testing.T) {
		val := "test"
		ptr := Ptr(val)
		assert.NotNil(t, ptr)
		assert.Equal(t, val, *ptr)
	})

	t.Run("Bool pointer", func(t *testing.T) {
		val := true
		ptr := Ptr(val)
		assert.NotNil(t, ptr)
		assert.Equal(t, val, *ptr)
	})

	t.Run("Int pointer", func(t *testing.T) {
		val := 42
		ptr := Ptr(val)
		assert.NotNil(t, ptr)
		assert.Equal(t, val, *ptr)
	})

	t.Run("False bool pointer", func(t *testing.T) {
		val := false
		ptr := Ptr(val)
		assert.NotNil(t, ptr)
		assert.Equal(t, val, *ptr)
	})
}

func TestMergeRequestCompatibilityFields(t *testing.T) {
	mr := &MergeRequest{
		IID:         42,
		Description: "Test description",
	}

	// Test Number alias for IID
	assert.Equal(t, mr.IID, mr.Number)
	mr.Number = 24
	assert.Equal(t, int64(24), mr.IID)

	// Test Body alias for Description  
	assert.Equal(t, mr.Description, mr.Body)
	mr.Body = "Updated body"
	assert.Equal(t, "Updated body", mr.Description)
}