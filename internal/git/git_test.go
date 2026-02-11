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

	// Initially, there should be only the main worktree
	worktrees, err := avGitRepo.ListWorktrees(t.Context())
	require.NoError(t, err)
	require.Len(t, worktrees, 1)
	require.Equal(t, repo.RepoDir, worktrees[0].Path)
	require.Equal(t, "main", worktrees[0].Branch)
	require.False(t, worktrees[0].IsBare)
	require.False(t, worktrees[0].IsDetached)
	require.NotEmpty(t, worktrees[0].Head)
	require.NotEmpty(t, worktrees[0].GitDir)

	// Create a new branch
	repo.Git(t, "checkout", "-b", "feature-branch")
	repo.Git(t, "checkout", "main")

	// Add a linked worktree
	worktreePath := t.TempDir() + "/feature-worktree"
	repo.Git(t, "worktree", "add", worktreePath, "feature-branch")

	// Now we should have two worktrees
	worktrees, err = avGitRepo.ListWorktrees(t.Context())
	require.NoError(t, err)
	require.Len(t, worktrees, 2)

	// Find the main and linked worktrees
	var mainWT, linkedWT *git.Worktree
	for i := range worktrees {
		if worktrees[i].Path == repo.RepoDir {
			mainWT = &worktrees[i]
		} else if worktrees[i].Path == worktreePath {
			linkedWT = &worktrees[i]
		}
	}

	require.NotNil(t, mainWT, "main worktree should be found")
	require.NotNil(t, linkedWT, "linked worktree should be found")

	require.Equal(t, "main", mainWT.Branch)
	require.Equal(t, "feature-branch", linkedWT.Branch)
}

func TestListWorktrees_DetachedHead(t *testing.T) {
	repo := gittest.NewTempRepo(t)
	avGitRepo := repo.AsAvGitRepo()

	// Create a commit to get a valid hash
	hash := repo.CommitFile(t, "test.txt", "test content")

	// Create a worktree with detached HEAD
	worktreePath := t.TempDir() + "/detached-worktree"
	repo.Git(t, "worktree", "add", "--detach", worktreePath, hash.String())

	worktrees, err := avGitRepo.ListWorktrees(t.Context())
	require.NoError(t, err)
	require.Len(t, worktrees, 2)

	// Find the detached worktree
	var detachedWT *git.Worktree
	for i := range worktrees {
		if worktrees[i].Path == worktreePath {
			detachedWT = &worktrees[i]
			break
		}
	}

	require.NotNil(t, detachedWT, "detached worktree should be found")
	require.True(t, detachedWT.IsDetached)
	require.Empty(t, detachedWT.Branch)
	require.Equal(t, hash.String(), detachedWT.Head)
}

func TestFindWorktreeForBranch(t *testing.T) {
	repo := gittest.NewTempRepo(t)
	avGitRepo := repo.AsAvGitRepo()

	// Create a new branch
	repo.Git(t, "checkout", "-b", "feature-branch")
	repo.Git(t, "checkout", "main")

	// Add a linked worktree for feature-branch
	worktreePath := t.TempDir() + "/feature-worktree"
	repo.Git(t, "worktree", "add", worktreePath, "feature-branch")

	// Find worktree for feature-branch
	wt, err := avGitRepo.FindWorktreeForBranch(t.Context(), "feature-branch")
	require.NoError(t, err)
	require.NotNil(t, wt)
	require.Equal(t, worktreePath, wt.Path)
	require.Equal(t, "feature-branch", wt.Branch)

	// Find worktree for main
	wt, err = avGitRepo.FindWorktreeForBranch(t.Context(), "main")
	require.NoError(t, err)
	require.NotNil(t, wt)
	require.Equal(t, repo.RepoDir, wt.Path)
	require.Equal(t, "main", wt.Branch)

	// Find worktree for non-existent branch
	wt, err = avGitRepo.FindWorktreeForBranch(t.Context(), "non-existent-branch")
	require.NoError(t, err)
	require.Nil(t, wt)
}

func TestWorktreeGitDirResolution(t *testing.T) {
	repo := gittest.NewTempRepo(t)
	avGitRepo := repo.AsAvGitRepo()

	// Create a new branch and worktree
	repo.Git(t, "checkout", "-b", "feature-branch")
	repo.Git(t, "checkout", "main")

	worktreePath := t.TempDir() + "/feature-worktree"
	repo.Git(t, "worktree", "add", worktreePath, "feature-branch")

	worktrees, err := avGitRepo.ListWorktrees(t.Context())
	require.NoError(t, err)
	require.Len(t, worktrees, 2)

	for _, wt := range worktrees {
		// Verify GitDir is set and is an absolute path
		require.NotEmpty(t, wt.GitDir, "GitDir should be set for worktree at %s", wt.Path)
		require.True(t, len(wt.GitDir) > 0 && wt.GitDir[0] == '/', "GitDir should be absolute path, got: %s", wt.GitDir)

		// For the main worktree, GitDir should be the repository's .git directory
		if wt.Path == repo.RepoDir {
			require.Contains(t, wt.GitDir, ".git", "Main worktree GitDir should contain .git")
		} else {
			// For linked worktrees, GitDir should be in .git/worktrees/
			require.Contains(t, wt.GitDir, "worktrees", "Linked worktree GitDir should contain 'worktrees'")
		}
	}
}
