package git_test

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"github.com/aviator-co/av/internal/git"
	"github.com/aviator-co/av/internal/git/gittest"
	"github.com/stretchr/testify/require"
)

func TestParseWorktreeListPorcelain(t *testing.T) {
	tests := []struct {
		name    string
		input   string
		want    []git.WorktreeInfo
		wantErr bool
	}{
		{
			name: "single worktree with branch",
			input: `worktree /path/to/repo
HEAD 1234567890abcdef1234567890abcdef12345678
branch refs/heads/main`,
			want: []git.WorktreeInfo{
				{
					Path:   "/path/to/repo",
					Head:   "1234567890abcdef1234567890abcdef12345678",
					Branch: "main",
				},
			},
		},
		{
			name: "multiple worktrees",
			input: `worktree /path/to/repo
HEAD 1234567890abcdef1234567890abcdef12345678
branch refs/heads/main

worktree /path/to/worktree1
HEAD abcdef1234567890abcdef1234567890abcdef12
branch refs/heads/feature-a

worktree /path/to/worktree2
HEAD fedcba0987654321fedcba0987654321fedcba09
branch refs/heads/feature-b`,
			want: []git.WorktreeInfo{
				{
					Path:   "/path/to/repo",
					Head:   "1234567890abcdef1234567890abcdef12345678",
					Branch: "main",
				},
				{
					Path:   "/path/to/worktree1",
					Head:   "abcdef1234567890abcdef1234567890abcdef12",
					Branch: "feature-a",
				},
				{
					Path:   "/path/to/worktree2",
					Head:   "fedcba0987654321fedcba0987654321fedcba09",
					Branch: "feature-b",
				},
			},
		},
		{
			name: "detached HEAD worktree",
			input: `worktree /path/to/repo
HEAD 1234567890abcdef1234567890abcdef12345678
branch refs/heads/main

worktree /path/to/detached
HEAD abcdef1234567890abcdef1234567890abcdef12
detached`,
			want: []git.WorktreeInfo{
				{
					Path:   "/path/to/repo",
					Head:   "1234567890abcdef1234567890abcdef12345678",
					Branch: "main",
				},
				{
					Path:       "/path/to/detached",
					Head:       "abcdef1234567890abcdef1234567890abcdef12",
					IsDetached: true,
				},
			},
		},
		{
			name: "bare repository",
			input: `worktree /path/to/bare.git
HEAD 1234567890abcdef1234567890abcdef12345678
bare`,
			want: []git.WorktreeInfo{
				{
					Path:   "/path/to/bare.git",
					Head:   "1234567890abcdef1234567890abcdef12345678",
					IsBare: true,
				},
			},
		},
		{
			name: "locked worktree with reason",
			input: `worktree /path/to/locked
HEAD 1234567890abcdef1234567890abcdef12345678
branch refs/heads/feature
locked working on important feature`,
			want: []git.WorktreeInfo{
				{
					Path:       "/path/to/locked",
					Head:       "1234567890abcdef1234567890abcdef12345678",
					Branch:     "feature",
					Locked:     true,
					LockReason: "working on important feature",
				},
			},
		},
		{
			name: "locked worktree without reason",
			input: `worktree /path/to/locked
HEAD 1234567890abcdef1234567890abcdef12345678
branch refs/heads/feature
locked`,
			want: []git.WorktreeInfo{
				{
					Path:   "/path/to/locked",
					Head:   "1234567890abcdef1234567890abcdef12345678",
					Branch: "feature",
					Locked: true,
				},
			},
		},
		{
			name: "prunable worktree with reason",
			input: `worktree /path/to/prunable
HEAD 1234567890abcdef1234567890abcdef12345678
branch refs/heads/old-feature
prunable gitdir file points to non-existent location`,
			want: []git.WorktreeInfo{
				{
					Path:           "/path/to/prunable",
					Head:           "1234567890abcdef1234567890abcdef12345678",
					Branch:         "old-feature",
					Prunable:       true,
					PrunableReason: "gitdir file points to non-existent location",
				},
			},
		},
		{
			name: "prunable worktree without reason",
			input: `worktree /path/to/prunable
HEAD 1234567890abcdef1234567890abcdef12345678
branch refs/heads/old-feature
prunable`,
			want: []git.WorktreeInfo{
				{
					Path:     "/path/to/prunable",
					Head:     "1234567890abcdef1234567890abcdef12345678",
					Branch:   "old-feature",
					Prunable: true,
				},
			},
		},
		{
			name: "complex worktree with multiple attributes",
			input: `worktree /path/to/repo
HEAD 1234567890abcdef1234567890abcdef12345678
branch refs/heads/main

worktree /path/to/complex
HEAD abcdef1234567890abcdef1234567890abcdef12
branch refs/heads/feature
locked
prunable`,
			want: []git.WorktreeInfo{
				{
					Path:   "/path/to/repo",
					Head:   "1234567890abcdef1234567890abcdef12345678",
					Branch: "main",
				},
				{
					Path:     "/path/to/complex",
					Head:     "abcdef1234567890abcdef1234567890abcdef12",
					Branch:   "feature",
					Locked:   true,
					Prunable: true,
				},
			},
		},
		{
			name: "no trailing blank line",
			input: `worktree /path/to/repo
HEAD 1234567890abcdef1234567890abcdef12345678
branch refs/heads/main`,
			want: []git.WorktreeInfo{
				{
					Path:   "/path/to/repo",
					Head:   "1234567890abcdef1234567890abcdef12345678",
					Branch: "main",
				},
			},
		},
		{
			name:    "empty output",
			input:   "",
			want:    nil,
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := git.ParseWorktreeListPorcelain(tt.input)
			if tt.wantErr {
				require.Error(t, err)
				return
			}
			require.NoError(t, err)
			require.Equal(t, tt.want, got)
		})
	}
}

func TestListWorktrees(t *testing.T) {
	repo := gittest.NewTempRepo(t)
	avRepo := repo.AsAvGitRepo()
	ctx := context.Background()

	// Create a branch
	repo.Git(t, "checkout", "-b", "feature-a")
	repo.Git(t, "commit", "--allow-empty", "-m", "commit 1")

	// List worktrees (should just be the main repo)
	worktrees, err := avRepo.ListWorktrees(ctx)
	require.NoError(t, err)
	require.Len(t, worktrees, 1)
	require.Equal(t, repo.RepoDir, worktrees[0].Path)
	require.NotEmpty(t, worktrees[0].Head)
	require.Equal(t, "feature-a", worktrees[0].Branch)

	// Create an additional worktree
	worktreePath := filepath.Join(t.TempDir(), "worktree1")
	repo.Git(t, "worktree", "add", worktreePath, "main")

	// List worktrees again
	worktrees, err = avRepo.ListWorktrees(ctx)
	require.NoError(t, err)
	require.Len(t, worktrees, 2)

	// Find the new worktree
	var foundNewWorktree bool
	for _, wt := range worktrees {
		if wt.Path == worktreePath {
			foundNewWorktree = true
			require.Equal(t, "main", wt.Branch)
			require.NotEmpty(t, wt.Head)
		}
	}
	require.True(t, foundNewWorktree, "new worktree not found in list")
}

func TestGetWorktreeForBranch(t *testing.T) {
	repo := gittest.NewTempRepo(t)
	avRepo := repo.AsAvGitRepo()
	ctx := context.Background()

	// Create some branches
	repo.Git(t, "checkout", "-b", "feature-a")
	repo.Git(t, "commit", "--allow-empty", "-m", "commit 1")
	repo.Git(t, "checkout", "main")
	repo.Git(t, "checkout", "-b", "feature-b")
	repo.Git(t, "commit", "--allow-empty", "-m", "commit 2")

	// Create worktrees
	worktree1Path := filepath.Join(t.TempDir(), "worktree1")
	repo.Git(t, "worktree", "add", worktree1Path, "feature-a")

	// Test finding worktree by short branch name
	wt, err := avRepo.GetWorktreeForBranch(ctx, "feature-a")
	require.NoError(t, err)
	require.NotNil(t, wt)
	require.Equal(t, worktree1Path, wt.Path)
	require.Equal(t, "feature-a", wt.Branch)

	// Test finding worktree by full branch name
	wt, err = avRepo.GetWorktreeForBranch(ctx, "refs/heads/feature-a")
	require.NoError(t, err)
	require.NotNil(t, wt)
	require.Equal(t, worktree1Path, wt.Path)
	require.Equal(t, "feature-a", wt.Branch)

	// Test branch that exists and is checked out in the main worktree
	wt, err = avRepo.GetWorktreeForBranch(ctx, "feature-b")
	require.NoError(t, err)
	require.NotNil(t, wt)
	require.Equal(t, "feature-b", wt.Branch)

	// Test branch that doesn't exist
	wt, err = avRepo.GetWorktreeForBranch(ctx, "nonexistent")
	require.NoError(t, err)
	require.Nil(t, wt)
}

func TestIsBranchCheckedOut(t *testing.T) {
	repo := gittest.NewTempRepo(t)
	avRepo := repo.AsAvGitRepo()
	ctx := context.Background()

	// Create some branches
	repo.Git(t, "checkout", "-b", "feature-a")
	repo.Git(t, "commit", "--allow-empty", "-m", "commit 1")
	repo.Git(t, "checkout", "main")
	repo.Git(t, "checkout", "-b", "feature-b")
	repo.Git(t, "commit", "--allow-empty", "-m", "commit 2")

	// Create a worktree
	worktreePath := filepath.Join(t.TempDir(), "worktree1")
	repo.Git(t, "worktree", "add", worktreePath, "feature-a")

	// Test branch checked out in worktree (short name)
	isCheckedOut, path, err := avRepo.IsBranchCheckedOut(ctx, "feature-a")
	require.NoError(t, err)
	require.True(t, isCheckedOut)
	require.Equal(t, worktreePath, path)

	// Test branch checked out in worktree (full name)
	isCheckedOut, path, err = avRepo.IsBranchCheckedOut(ctx, "refs/heads/feature-a")
	require.NoError(t, err)
	require.True(t, isCheckedOut)
	require.Equal(t, worktreePath, path)

	// Test branch checked out in main repo
	isCheckedOut, path, err = avRepo.IsBranchCheckedOut(ctx, "feature-b")
	require.NoError(t, err)
	require.True(t, isCheckedOut)
	require.Equal(t, repo.RepoDir, path)

	// Test branch that exists but is not checked out
	repo.Git(t, "branch", "feature-c")
	isCheckedOut, path, err = avRepo.IsBranchCheckedOut(ctx, "feature-c")
	require.NoError(t, err)
	require.False(t, isCheckedOut)
	require.Empty(t, path)

	// Test branch that doesn't exist
	isCheckedOut, path, err = avRepo.IsBranchCheckedOut(ctx, "nonexistent")
	require.NoError(t, err)
	require.False(t, isCheckedOut)
	require.Empty(t, path)
}

func TestIsWorktreeLockError(t *testing.T) {
	tests := []struct {
		name string
		err  error
		want bool
	}{
		{
			name: "nil error",
			err:  nil,
			want: false,
		},
		{
			name: "worktree lock error - checked out at",
			err:  os.ErrExist, // placeholder for demonstration
			want: false,
		},
		{
			name: "worktree lock error with checked out message",
			err:  &testError{msg: "fatal: 'feature-branch' is already checked out at '/path/to/worktree'"},
			want: true,
		},
		{
			name: "worktree lock error with used by message",
			err:  &testError{msg: "error: Cannot delete branch 'my-branch' used by worktree at '/some/path'"},
			want: true,
		},
		{
			name: "other git error",
			err:  &testError{msg: "fatal: branch 'feature' not found"},
			want: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := git.IsWorktreeLockError(tt.err)
			require.Equal(t, tt.want, got)
		})
	}
}

func TestParseWorktreeLockError(t *testing.T) {
	tests := []struct {
		name       string
		err        error
		wantBranch string
		wantPath   string
	}{
		{
			name:       "nil error",
			err:        nil,
			wantBranch: "",
			wantPath:   "",
		},
		{
			name:       "checked out at pattern",
			err:        &testError{msg: "fatal: 'feature-branch' is already checked out at '/path/to/worktree'"},
			wantBranch: "feature-branch",
			wantPath:   "/path/to/worktree",
		},
		{
			name:       "checked out at pattern with different quotes",
			err:        &testError{msg: "fatal: 'my-feature' is already checked out at '/home/user/workspace'"},
			wantBranch: "my-feature",
			wantPath:   "/home/user/workspace",
		},
		{
			name:       "used by worktree pattern",
			err:        &testError{msg: "error: Cannot delete branch 'test-branch' used by worktree at '/tmp/test'"},
			wantBranch: "test-branch",
			wantPath:   "",
		},
		{
			name:       "non-worktree error",
			err:        &testError{msg: "fatal: branch 'feature' not found"},
			wantBranch: "",
			wantPath:   "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gotBranch, gotPath := git.ParseWorktreeLockError(tt.err)
			require.Equal(t, tt.wantBranch, gotBranch)
			require.Equal(t, tt.wantPath, gotPath)
		})
	}
}

// testError is a simple error type for testing
type testError struct {
	msg string
}

func (e *testError) Error() string {
	return e.msg
}
