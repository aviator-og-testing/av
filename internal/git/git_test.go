package git_test

import (
	"testing"

	"github.com/aviator-co/av/internal/config"
	"github.com/aviator-co/av/internal/git"
	"github.com/aviator-co/av/internal/git/gittest"
	"github.com/stretchr/testify/require"
)

func TestOrigin(t *testing.T) {
	repo := gittest.NewTempRepo(t)
	repo.Git(t, "remote", "set-url", "origin", "https://github.com/aviator-co/av.git")
	origin, err := repo.AsAvGitRepo().Origin(t.Context())
	require.NoError(t, err)
	require.Equal(t, "aviator-co/av", origin.RepoSlug)

	repo.Git(t, "remote", "set-url", "origin", "git@github.com:aviator-co/av.git")
	require.NoError(t, err)
	origin, err = repo.AsAvGitRepo().Origin(t.Context())
	require.NoError(t, err)
	require.Equal(t, "aviator-co/av", origin.RepoSlug)
}

func TestTrunkBranches(t *testing.T) {
	repo := gittest.NewTempRepo(t)

	branches := repo.AsAvGitRepo().TrunkBranches()
	require.Equal(t, branches, []string{"main"})

	// add some branches to AdditionalTrunkBranches
	config.Av.AdditionalTrunkBranches = []string{"develop", "staging"}
	branches = repo.AsAvGitRepo().TrunkBranches()
	require.Equal(t, branches, []string{"main", "develop", "staging"})
}

func TestGetRemoteName(t *testing.T) {
	repo := gittest.NewTempRepo(t)
	avGitRepo := repo.AsAvGitRepo()
	require.Equal(t, avGitRepo.GetRemoteName(), git.DEFAULT_REMOTE_NAME)

	// This is a global config, so changing it here affects other tests. Be
	// sure to reset it.
	config.Av.Remote = "new-remote"
	require.Equal(t, avGitRepo.GetRemoteName(), "new-remote")
	config.Av.Remote = ""
}

func TestListWorktrees(t *testing.T) {
	repo := gittest.NewTempRepo(t)
	avGitRepo := repo.AsAvGitRepo()

	worktrees, err := avGitRepo.ListWorktrees(t.Context())
	require.NoError(t, err)
	require.Len(t, worktrees, 1)
	require.Equal(t, repo.RepoDir(), worktrees[0].Path)
	require.Equal(t, "main", worktrees[0].Branch)
	require.False(t, worktrees[0].IsDetached)
	require.NotEmpty(t, worktrees[0].Head)

	repo.Git(t, "checkout", "-b", "feature")
	repo.Git(t, "worktree", "add", "../other-worktree", "-b", "other-branch")

	worktrees, err = avGitRepo.ListWorktrees(t.Context())
	require.NoError(t, err)
	require.Len(t, worktrees, 2)

	var mainWorktree, otherWorktree git.Worktree
	for _, wt := range worktrees {
		if wt.Branch == "feature" {
			mainWorktree = wt
		} else if wt.Branch == "other-branch" {
			otherWorktree = wt
		}
	}

	require.Equal(t, repo.RepoDir(), mainWorktree.Path)
	require.Equal(t, "feature", mainWorktree.Branch)
	require.False(t, mainWorktree.IsDetached)

	require.Contains(t, otherWorktree.Path, "other-worktree")
	require.Equal(t, "other-branch", otherWorktree.Branch)
	require.False(t, otherWorktree.IsDetached)
}

func TestIsBranchCheckedOut(t *testing.T) {
	repo := gittest.NewTempRepo(t)
	avGitRepo := repo.AsAvGitRepo()

	// Initially on main branch
	isCheckedOut, path, err := avGitRepo.IsBranchCheckedOut(t.Context(), "main")
	require.NoError(t, err)
	require.True(t, isCheckedOut)
	require.Equal(t, repo.RepoDir(), path)

	// Test with full ref format
	isCheckedOut, path, err = avGitRepo.IsBranchCheckedOut(t.Context(), "refs/heads/main")
	require.NoError(t, err)
	require.True(t, isCheckedOut)
	require.Equal(t, repo.RepoDir(), path)

	// Create a new branch and worktree
	repo.Git(t, "checkout", "-b", "feature")
	repo.Git(t, "worktree", "add", "../other-worktree", "-b", "other-branch")

	// Check feature branch (current branch)
	isCheckedOut, path, err = avGitRepo.IsBranchCheckedOut(t.Context(), "feature")
	require.NoError(t, err)
	require.True(t, isCheckedOut)
	require.Equal(t, repo.RepoDir(), path)

	// Check other-branch (in other worktree)
	isCheckedOut, path, err = avGitRepo.IsBranchCheckedOut(t.Context(), "other-branch")
	require.NoError(t, err)
	require.True(t, isCheckedOut)
	require.Contains(t, path, "other-worktree")

	// Check non-checked-out branch
	repo.Git(t, "branch", "unused-branch")
	isCheckedOut, path, err = avGitRepo.IsBranchCheckedOut(t.Context(), "unused-branch")
	require.NoError(t, err)
	require.False(t, isCheckedOut)
	require.Empty(t, path)

	// Check non-existent branch (should return false without error)
	isCheckedOut, path, err = avGitRepo.IsBranchCheckedOut(t.Context(), "non-existent")
	require.NoError(t, err)
	require.False(t, isCheckedOut)
	require.Empty(t, path)
}
