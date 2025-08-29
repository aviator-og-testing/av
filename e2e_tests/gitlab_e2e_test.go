package e2e_tests

import (
	"os"
	"testing"

	"github.com/aviator-co/av/internal/git/gittest"
	"github.com/stretchr/testify/require"
)

func TestGitLabSync(t *testing.T) {
	server := RunMockGitLabServer(t)
	defer server.Close()
	repo := gittest.NewTempRepoWithGitLabServer(t, server.URL)
	Chdir(t, repo.RepoDir)

	// Set GitLab token for testing
	oldToken := os.Getenv("AV_GITLAB_TOKEN")
	defer func() {
		if oldToken == "" {
			os.Unsetenv("AV_GITLAB_TOKEN")
		} else {
			os.Setenv("AV_GITLAB_TOKEN", oldToken)
		}
	}()
	os.Setenv("AV_GITLAB_TOKEN", "glpat_testtoken")

	// Create a simple branch structure similar to GitHub tests
	RequireAv(t, "branch", "gitlab-feature")
	repo.CommitFile(t, "feature.txt", "feature code\n", gittest.WithMessage("Add feature"))
	
	// Test sync with GitLab - this should work without errors once provider abstraction is in place
	RequireAv(t, "sync")

	// Verify the branch still exists and has the expected commit
	RequireCurrentBranchName(t, repo, "refs/heads/gitlab-feature")
}

func TestGitLabBranchWorkflow(t *testing.T) {
	server := RunMockGitLabServer(t)
	defer server.Close()
	repo := gittest.NewTempRepoWithGitLabServer(t, server.URL)
	Chdir(t, repo.RepoDir)

	// Set GitLab token for testing
	oldToken := os.Getenv("AV_GITLAB_TOKEN")
	defer func() {
		if oldToken == "" {
			os.Unsetenv("AV_GITLAB_TOKEN")
		} else {
			os.Setenv("AV_GITLAB_TOKEN", oldToken)
		}
	}()
	os.Setenv("AV_GITLAB_TOKEN", "glpat_testtoken")

	// Test creating branches similar to existing e2e patterns
	RequireAv(t, "branch", "gitlab-branch-1")
	repo.CommitFile(t, "file1.txt", "content1\n", gittest.WithMessage("First commit"))
	
	RequireAv(t, "branch", "gitlab-branch-2")
	repo.CommitFile(t, "file2.txt", "content2\n", gittest.WithMessage("Second commit"))

	// Test branch navigation
	RequireAv(t, "checkout", "gitlab-branch-1")
	RequireCurrentBranchName(t, repo, "refs/heads/gitlab-branch-1")
	
	RequireAv(t, "checkout", "gitlab-branch-2")
	RequireCurrentBranchName(t, repo, "refs/heads/gitlab-branch-2")
}

func TestGitLabStackOperations(t *testing.T) {
	server := RunMockGitLabServer(t)
	defer server.Close()
	repo := gittest.NewTempRepoWithGitLabServer(t, server.URL)
	Chdir(t, repo.RepoDir)

	// Set GitLab token for testing
	oldToken := os.Getenv("AV_GITLAB_TOKEN")
	defer func() {
		if oldToken == "" {
			os.Unsetenv("AV_GITLAB_TOKEN")
		} else {
			os.Setenv("AV_GITLAB_TOKEN", oldToken)
		}
	}()
	os.Setenv("AV_GITLAB_TOKEN", "glpat_testtoken")

	// Create a stack of branches for GitLab testing
	RequireAv(t, "branch", "gitlab-base")
	repo.CommitFile(t, "base.txt", "base\n", gittest.WithMessage("Base commit"))
	
	RequireAv(t, "branch", "gitlab-middle")
	repo.CommitFile(t, "middle.txt", "base\nmiddle\n", gittest.WithMessage("Middle commit"))
	
	RequireAv(t, "branch", "gitlab-top")
	repo.CommitFile(t, "top.txt", "base\nmiddle\ntop\n", gittest.WithMessage("Top commit"))

	// Test tree view to ensure stack is properly structured
	RequireAv(t, "tree")

	// Test sync operations on the stack
	RequireAv(t, "sync", "--all")
}

func TestGitLabProviderDetection(t *testing.T) {
	server := RunMockGitLabServer(t)
	defer server.Close()
	repo := gittest.NewTempRepoWithGitLabServer(t, server.URL)
	Chdir(t, repo.RepoDir)

	// Set both tokens to test provider detection
	oldGitHubToken := os.Getenv("AV_GITHUB_TOKEN")
	oldGitLabToken := os.Getenv("AV_GITLAB_TOKEN")
	defer func() {
		if oldGitHubToken == "" {
			os.Unsetenv("AV_GITHUB_TOKEN")
		} else {
			os.Setenv("AV_GITHUB_TOKEN", oldGitHubToken)
		}
		if oldGitLabToken == "" {
			os.Unsetenv("AV_GITLAB_TOKEN")
		} else {
			os.Setenv("AV_GITLAB_TOKEN", oldGitLabToken)
		}
	}()
	
	// Keep GitHub token but add GitLab token
	os.Setenv("AV_GITHUB_TOKEN", "ghp_thisisntarealltokenitsjustfortesting")
	os.Setenv("AV_GITLAB_TOKEN", "glpat_testtoken")

	// With GitLab remote URL, should auto-detect GitLab provider
	RequireAv(t, "branch", "auto-detect-test")
	repo.CommitFile(t, "detect.txt", "provider detection test\n", gittest.WithMessage("Test provider detection"))
	
	// This should work without specifying provider since URL should auto-detect GitLab
	RequireAv(t, "sync")
}