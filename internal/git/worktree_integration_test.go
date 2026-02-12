package git_test

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"github.com/aviator-co/av/internal/git"
	"github.com/aviator-co/av/internal/git/gittest"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestListWorktrees_SingleWorktree(t *testing.T) {
	repo := gittest.NewTempRepo(t)
	defer repo.Remove(t)

	avRepo, err := git.OpenRepo(repo.Dir, repo.GitDir)
	require.NoError(t, err)

	ctx := context.Background()
	worktrees, err := avRepo.ListWorktrees(ctx)
	require.NoError(t, err)

	// Should have exactly one worktree (the main one)
	assert.Len(t, worktrees, 1)
	assert.Equal(t, repo.Dir, worktrees[0].Path)
	assert.Equal(t, "main", worktrees[0].Branch)
	assert.False(t, worktrees[0].IsBare)
	assert.False(t, worktrees[0].IsDetached)
	assert.False(t, worktrees[0].Locked)
	assert.NotEmpty(t, worktrees[0].Head)
}

func TestListWorktrees_MultipleWorktrees(t *testing.T) {
	repo := gittest.NewTempRepo(t)
	defer repo.Remove(t)

	avRepo, err := git.OpenRepo(repo.Dir, repo.GitDir)
	require.NoError(t, err)

	// Create a feature branch
	repo.Git(t, "checkout", "-b", "feature-branch")
	repo.Git(t, "checkout", "main")

	// Create a second worktree
	worktreeDir := filepath.Join(t.TempDir(), "feature-worktree")
	repo.Git(t, "worktree", "add", worktreeDir, "feature-branch")

	ctx := context.Background()
	worktrees, err := avRepo.ListWorktrees(ctx)
	require.NoError(t, err)

	// Should have two worktrees
	assert.Len(t, worktrees, 2)

	// Find the main worktree
	var mainWt *git.WorktreeInfo
	var featureWt *git.WorktreeInfo
	for i := range worktrees {
		if worktrees[i].Branch == "main" {
			mainWt = &worktrees[i]
		} else if worktrees[i].Branch == "feature-branch" {
			featureWt = &worktrees[i]
		}
	}

	require.NotNil(t, mainWt, "should find main worktree")
	require.NotNil(t, featureWt, "should find feature worktree")

	assert.Equal(t, repo.Dir, mainWt.Path)
	assert.Equal(t, "main", mainWt.Branch)
	assert.Equal(t, worktreeDir, featureWt.Path)
	assert.Equal(t, "feature-branch", featureWt.Branch)
}

func TestListWorktrees_DetachedHead(t *testing.T) {
	repo := gittest.NewTempRepo(t)
	defer repo.Remove(t)

	// Get the current commit hash
	headHash := repo.Git(t, "rev-parse", "HEAD")

	// Detach HEAD
	repo.Git(t, "checkout", "--detach")

	avRepo, err := git.OpenRepo(repo.Dir, repo.GitDir)
	require.NoError(t, err)

	ctx := context.Background()
	worktrees, err := avRepo.ListWorktrees(ctx)
	require.NoError(t, err)

	assert.Len(t, worktrees, 1)
	assert.Equal(t, repo.Dir, worktrees[0].Path)
	assert.Empty(t, worktrees[0].Branch, "detached HEAD should have no branch")
	assert.True(t, worktrees[0].IsDetached)
	assert.Equal(t, headHash, worktrees[0].Head)
}

func TestGetWorktreeForBranch_Found(t *testing.T) {
	repo := gittest.NewTempRepo(t)
	defer repo.Remove(t)

	avRepo, err := git.OpenRepo(repo.Dir, repo.GitDir)
	require.NoError(t, err)

	// Create a feature branch
	repo.Git(t, "checkout", "-b", "feature-branch")
	repo.Git(t, "checkout", "main")

	// Create a worktree for the feature branch
	worktreeDir := filepath.Join(t.TempDir(), "feature-worktree")
	repo.Git(t, "worktree", "add", worktreeDir, "feature-branch")

	ctx := context.Background()
	wt, err := avRepo.GetWorktreeForBranch(ctx, "feature-branch")
	require.NoError(t, err)
	require.NotNil(t, wt)

	assert.Equal(t, "feature-branch", wt.Branch)
	assert.Equal(t, worktreeDir, wt.Path)
}

func TestGetWorktreeForBranch_NotFound(t *testing.T) {
	repo := gittest.NewTempRepo(t)
	defer repo.Remove(t)

	avRepo, err := git.OpenRepo(repo.Dir, repo.GitDir)
	require.NoError(t, err)

	// Create a branch but don't check it out anywhere
	repo.Git(t, "branch", "unused-branch")

	ctx := context.Background()
	wt, err := avRepo.GetWorktreeForBranch(ctx, "unused-branch")
	require.NoError(t, err)
	assert.Nil(t, wt, "should return nil for branch not checked out in any worktree")
}

func TestGetWorktreeForBranch_NonExistentBranch(t *testing.T) {
	repo := gittest.NewTempRepo(t)
	defer repo.Remove(t)

	avRepo, err := git.OpenRepo(repo.Dir, repo.GitDir)
	require.NoError(t, err)

	ctx := context.Background()
	wt, err := avRepo.GetWorktreeForBranch(ctx, "does-not-exist")
	require.NoError(t, err)
	assert.Nil(t, wt, "should return nil for non-existent branch")
}

func TestWorktreeLockError_RealCheckout(t *testing.T) {
	repo := gittest.NewTempRepo(t)
	defer repo.Remove(t)

	avRepo, err := git.OpenRepo(repo.Dir, repo.GitDir)
	require.NoError(t, err)

	// Create a feature branch
	repo.Git(t, "checkout", "-b", "feature-branch")

	// Create a worktree for the feature branch
	worktreeDir := filepath.Join(t.TempDir(), "feature-worktree")
	repo.Git(t, "worktree", "add", worktreeDir, "feature-branch")

	// Try to checkout the same branch in the main worktree (should fail)
	ctx := context.Background()
	_, err = avRepo.CheckoutBranch(ctx, &git.CheckoutBranch{
		Name: "feature-branch",
	})

	// Should get an error
	require.Error(t, err)

	// Should be detected as a worktree lock error
	assert.True(t, git.IsWorktreeLockError(err))

	// Should be able to parse the error
	branch, path := git.ParseWorktreeLockError(err)
	assert.Equal(t, "feature-branch", branch)
	assert.Equal(t, worktreeDir, path)

	// Should be able to wrap the error
	wrappedErr := git.WrapWorktreeLockError(err)
	var wtErr *git.ErrWorktreeLock
	require.ErrorAs(t, wrappedErr, &wtErr)
	assert.Equal(t, "feature-branch", wtErr.Branch)
	assert.Equal(t, worktreeDir, wtErr.WorktreePath)
}

func TestListWorktrees_LockedWorktree(t *testing.T) {
	repo := gittest.NewTempRepo(t)
	defer repo.Remove(t)

	// Create a feature branch
	repo.Git(t, "checkout", "-b", "feature-branch")
	repo.Git(t, "checkout", "main")

	// Create a worktree
	worktreeDir := filepath.Join(t.TempDir(), "feature-worktree")
	repo.Git(t, "worktree", "add", worktreeDir, "feature-branch")

	// Lock the worktree
	repo.Git(t, "worktree", "lock", worktreeDir, "--reason", "testing")

	avRepo, err := git.OpenRepo(repo.Dir, repo.GitDir)
	require.NoError(t, err)

	ctx := context.Background()
	worktrees, err := avRepo.ListWorktrees(ctx)
	require.NoError(t, err)

	// Find the locked worktree
	var lockedWt *git.WorktreeInfo
	for i := range worktrees {
		if worktrees[i].Branch == "feature-branch" {
			lockedWt = &worktrees[i]
			break
		}
	}

	require.NotNil(t, lockedWt)
	assert.True(t, lockedWt.Locked)
	assert.Equal(t, "testing", lockedWt.LockReason)
}

func TestListWorktrees_PrunableWorktree(t *testing.T) {
	repo := gittest.NewTempRepo(t)
	defer repo.Remove(t)

	// Create a feature branch
	repo.Git(t, "checkout", "-b", "feature-branch")
	repo.Git(t, "checkout", "main")

	// Create a worktree
	worktreeDir := filepath.Join(t.TempDir(), "feature-worktree")
	repo.Git(t, "worktree", "add", worktreeDir, "feature-branch")

	// Remove the worktree directory (but not the Git metadata)
	// This makes it prunable
	err := os.RemoveAll(worktreeDir)
	require.NoError(t, err)

	avRepo, err := git.OpenRepo(repo.Dir, repo.GitDir)
	require.NoError(t, err)

	ctx := context.Background()
	worktrees, err := avRepo.ListWorktrees(ctx)
	require.NoError(t, err)

	// Find the prunable worktree
	var prunableWt *git.WorktreeInfo
	for i := range worktrees {
		if worktrees[i].Branch == "feature-branch" {
			prunableWt = &worktrees[i]
			break
		}
	}

	require.NotNil(t, prunableWt)
	assert.True(t, prunableWt.Prunable)
	// Note: PrunableReason might be empty or contain a message depending on Git version
}
