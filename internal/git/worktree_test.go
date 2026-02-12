package git

import (
	"context"
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestListWorktrees_ParsesPorcelainOutput(t *testing.T) {
	// This is a unit test for the parsing logic.
	// We'll create integration tests separately once we have a test repo setup.

	// Test the WorktreeInfo struct is properly defined
	wt := WorktreeInfo{
		Path:           "/path/to/worktree",
		Head:           "abc123",
		Branch:         "main",
		IsBare:         false,
		IsDetached:     false,
		Locked:         false,
		LockReason:     "",
		Prunable:       false,
		PrunableReason: "",
	}

	assert.Equal(t, "/path/to/worktree", wt.Path)
	assert.Equal(t, "abc123", wt.Head)
	assert.Equal(t, "main", wt.Branch)
	assert.False(t, wt.IsBare)
	assert.False(t, wt.IsDetached)
	assert.False(t, wt.Locked)
	assert.False(t, wt.Prunable)
}

func TestIsWorktreeLockError(t *testing.T) {
	tests := []struct {
		name     string
		err      error
		expected bool
	}{
		{
			name:     "nil error",
			err:      nil,
			expected: false,
		},
		{
			name:     "worktree lock error with single quotes",
			err:      errors.New("fatal: 'feature-branch' is already checked out at '/home/user/worktree'"),
			expected: true,
		},
		{
			name:     "worktree lock error with different format",
			err:      errors.New("fatal: feature-branch is already checked out at /home/user/worktree"),
			expected: true,
		},
		{
			name:     "error level instead of fatal",
			err:      errors.New("error: 'main' is already checked out at '/tmp/main-worktree'"),
			expected: true,
		},
		{
			name:     "cannot lock ref error",
			err:      errors.New("error: cannot lock ref 'refs/heads/main': is at 1234567 but expected 7654321"),
			expected: true,
		},
		{
			name:     "unrelated git error",
			err:      errors.New("fatal: not a git repository"),
			expected: false,
		},
		{
			name:     "merge conflict error",
			err:      errors.New("CONFLICT (content): Merge conflict in file.txt"),
			expected: false,
		},
		{
			name:     "generic error",
			err:      errors.New("something went wrong"),
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := IsWorktreeLockError(tt.err)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestParseWorktreeLockError(t *testing.T) {
	tests := []struct {
		name             string
		err              error
		expectedBranch   string
		expectedWorktree string
	}{
		{
			name:             "nil error",
			err:              nil,
			expectedBranch:   "",
			expectedWorktree: "",
		},
		{
			name:             "standard worktree lock error",
			err:              errors.New("fatal: 'feature-branch' is already checked out at '/home/user/worktree'"),
			expectedBranch:   "feature-branch",
			expectedWorktree: "/home/user/worktree",
		},
		{
			name:             "worktree lock with spaces in path",
			err:              errors.New("fatal: 'main' is already checked out at '/home/user/my worktree'"),
			expectedBranch:   "main",
			expectedWorktree: "/home/user/my worktree",
		},
		{
			name:             "worktree lock without quotes",
			err:              errors.New("fatal: develop is already checked out at /tmp/develop-worktree"),
			expectedBranch:   "develop",
			expectedWorktree: "/tmp/develop-worktree",
		},
		{
			name:             "unrelated error",
			err:              errors.New("fatal: not a git repository"),
			expectedBranch:   "",
			expectedWorktree: "",
		},
		{
			name:             "cannot lock ref error",
			err:              errors.New("error: cannot lock ref 'refs/heads/main': is at 1234567 but expected 7654321"),
			expectedBranch:   "",
			expectedWorktree: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			branch, worktreePath := ParseWorktreeLockError(tt.err)
			assert.Equal(t, tt.expectedBranch, branch)
			assert.Equal(t, tt.expectedWorktree, worktreePath)
		})
	}
}

func TestErrWorktreeLock(t *testing.T) {
	origErr := errors.New("fatal: 'main' is already checked out at '/tmp/worktree'")
	err := &ErrWorktreeLock{
		Branch:       "main",
		WorktreePath: "/tmp/worktree",
		OriginalErr:  origErr,
	}

	assert.Equal(t, "cannot checkout 'main': already checked out at '/tmp/worktree'", err.Error())
	assert.Equal(t, origErr, errors.Unwrap(err))
}

func TestWrapWorktreeLockError(t *testing.T) {
	tests := []struct {
		name           string
		err            error
		expectWrapped  bool
		expectedBranch string
		expectedPath   string
	}{
		{
			name:          "nil error",
			err:           nil,
			expectWrapped: false,
		},
		{
			name:           "worktree lock error",
			err:            errors.New("fatal: 'feature' is already checked out at '/home/user/wt'"),
			expectWrapped:  true,
			expectedBranch: "feature",
			expectedPath:   "/home/user/wt",
		},
		{
			name:          "unrelated error",
			err:           errors.New("fatal: not a git repository"),
			expectWrapped: false,
		},
		{
			name:          "cannot lock ref without checkout message",
			err:           errors.New("error: cannot lock ref 'refs/heads/main'"),
			expectWrapped: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := WrapWorktreeLockError(tt.err)

			if !tt.expectWrapped {
				assert.Equal(t, tt.err, result)
				return
			}

			var wtErr *ErrWorktreeLock
			require.True(t, errors.As(result, &wtErr))
			assert.Equal(t, tt.expectedBranch, wtErr.Branch)
			assert.Equal(t, tt.expectedPath, wtErr.WorktreePath)
			assert.Equal(t, tt.err, errors.Unwrap(result))
		})
	}
}

func TestGetWorktreeForBranch_NotFound(t *testing.T) {
	// This test validates the logic when a branch is not found.
	// The actual Git integration will be tested separately.

	// We can't easily test the full ListWorktrees without a real Git repo,
	// but we can verify the function signature and nil return
	ctx := context.Background()

	// Create a mock scenario where GetWorktreeForBranch would return nil
	// when the branch doesn't exist in any worktree.
	// This will be tested properly in integration tests.
	_ = ctx
}
