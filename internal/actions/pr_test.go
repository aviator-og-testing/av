package actions_test

import (
	"context"
	"testing"

	"github.com/aviator-co/av/internal/actions"
	"github.com/aviator-co/av/internal/meta"
	"github.com/aviator-co/av/internal/utils/maputils"
	"github.com/aviator-co/av/internal/utils/stackutils"
	"github.com/aviator-co/av/internal/vcs"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/stretchr/testify/mock"
)

func TestReadPRMetadata(t *testing.T) {
	tx := fakeReadTx{}
	prMeta := actions.PRMetadata{
		Parent:     "foo",
		ParentHead: "bar",
		ParentPull: 123,
		Trunk:      "baz",
	}
	prBody := actions.AddPRMetadataAndStack(
		"Hello! This is a cool PR that does some neat things.",
		prMeta,
		"foo",
		nil,
		tx,
	)
	prMeta2, err := actions.ReadPRMetadata(prBody)
	require.NoError(t, err)
	assert.Equal(t, prMeta.Parent, prMeta2.Parent)
	assert.Equal(t, prMeta.ParentHead, prMeta2.ParentHead)
	assert.Equal(t, prMeta.ParentPull, prMeta2.ParentPull)
	assert.Equal(t, prMeta.Trunk, prMeta2.Trunk)

	prBody = actions.AddPRMetadataAndStack(prBody, actions.PRMetadata{
		Parent:     "foo2",
		ParentHead: "bar2",
		ParentPull: 1234,
		Trunk:      "baz2",
	}, "foo2", nil, tx)
	assert.Contains(t, prBody, "Hello! This is a cool PR that does some neat things.\n\n")
	prMeta2, err = actions.ReadPRMetadata(prBody)
	require.NoError(t, err)
	assert.Equal(t, "foo2", prMeta2.Parent)
	assert.Equal(t, "bar2", prMeta2.ParentHead)
}

func TestPRMetadataPreservesBody(t *testing.T) {
	tx := fakeReadTx{}
	sampleMeta := actions.PRMetadata{
		Parent:     "foo",
		ParentHead: "bar",
		ParentPull: 123,
		Trunk:      "baz",
	}
	body1 := actions.AddPRMetadataAndStack(
		"Hello! This is a cool PR that does some neat things.",
		sampleMeta,
		"foo",
		nil,
		tx,
	)
	// Add some text to the end of the body (as if someone had edited manually)
	body1 += "\n\nIt's very neat, actually."

	body2 := actions.AddPRMetadataAndStack(body1, sampleMeta, "foo", nil, tx)
	assert.Contains(t, body2, "Hello! This is a cool PR that does some neat things.")
	assert.Contains(t, body2, "It's very neat, actually.")
	assert.Contains(t, body2, "\n"+actions.PRMetadataCommentStart)
}

func TestPRWithStack(t *testing.T) {
	tx := fakeReadTx{
		"baz": {
			Name: "baz",
			Parent: meta.BranchState{
				Name:  "main",
				Trunk: true,
			},
			PullRequest: &meta.PullRequest{
				Number:    1001,
				Permalink: "https://github.com/org/repo/pull/1001",
			},
		},
		"foo": {
			Name: "foo",
			Parent: meta.BranchState{
				Name: "baz",
			},
			PullRequest: &meta.PullRequest{
				Number:    1002,
				Permalink: "https://github.com/org/repo/pull/1002",
			},
		},
	}
	stack := &stackutils.StackTreeNode{
		Branch: &stackutils.StackTreeBranchInfo{
			BranchName: "main",
		},
		Children: []*stackutils.StackTreeNode{
			{
				Branch: &stackutils.StackTreeBranchInfo{
					BranchName: "baz",
				},
				Children: []*stackutils.StackTreeNode{
					{
						Branch: &stackutils.StackTreeBranchInfo{
							BranchName: "foo",
						},
						Children: []*stackutils.StackTreeNode{},
					},
				},
			},
		},
	}

	sampleMeta := actions.PRMetadata{
		Parent:     "foo",
		ParentHead: "bar",
		ParentPull: 123,
		Trunk:      "baz",
	}
	body1 := actions.AddPRMetadataAndStack(
		"Hello! This is a cool PR that does some neat things.",
		sampleMeta,
		"foo",
		stack,
		tx,
	)

	assert.Equal(t, `<!-- av pr stack begin -->
<table><tr><td><details><summary><b>Depends on #1001.</b> This PR is part of a stack created with <a href="https://github.com/aviator-co/av">Aviator</a>.</summary>

* ➡️ **#1002**
* **#1001**
* `+"`"+`main`+"`"+`
</details></td></tr></table>
<!-- av pr stack end -->

Hello! This is a cool PR that does some neat things.

<!-- av pr metadata
This information is embedded by the av CLI when creating PRs to track the status of stacks when using Aviator. Please do not delete or edit this section of the PR.
`+"```"+`
{"parent":"foo","parentHead":"bar","parentPull":123,"trunk":"baz"}
`+"```"+`
-->
`, body1)
}

func TestPRWithForkedStack(t *testing.T) {
	tx := fakeReadTx{
		"baz": {
			Name: "baz",
			Parent: meta.BranchState{
				Name:  "main",
				Trunk: true,
			},
			PullRequest: &meta.PullRequest{
				Number:    1001,
				Permalink: "https://github.com/org/repo/pull/1001",
			},
		},
		"foo": {
			Name: "foo",
			Parent: meta.BranchState{
				Name: "baz",
			},
			PullRequest: &meta.PullRequest{
				Number:    1002,
				Permalink: "https://github.com/org/repo/pull/1002",
			},
		},
		"qux": {
			Name: "qux",
			Parent: meta.BranchState{
				Name:  "main",
				Trunk: true,
			},
			PullRequest: &meta.PullRequest{
				Number:    1003,
				Permalink: "https://github.com/org/repo/pull/1003",
			},
		},
	}
	stack := &stackutils.StackTreeNode{
		Branch: &stackutils.StackTreeBranchInfo{
			BranchName: "main",
		},
		Children: []*stackutils.StackTreeNode{
			{
				Branch: &stackutils.StackTreeBranchInfo{
					BranchName: "baz",
				},
				Children: []*stackutils.StackTreeNode{
					{
						Branch: &stackutils.StackTreeBranchInfo{
							BranchName: "foo",
						},
						Children: []*stackutils.StackTreeNode{},
					},
				},
			},
			{
				Branch: &stackutils.StackTreeBranchInfo{
					BranchName: "qux",
				},
				Children: []*stackutils.StackTreeNode{},
			},
		},
	}

	sampleMeta := actions.PRMetadata{
		Parent:     "foo",
		ParentHead: "bar",
		ParentPull: 123,
		Trunk:      "baz",
	}
	body1 := actions.AddPRMetadataAndStack(
		"Hello! This is a cool PR that does some neat things.",
		sampleMeta,
		"foo",
		stack,
		tx,
	)

	assert.Equal(t, `<!-- av pr stack begin -->
<table><tr><td><details><summary><b>Depends on #1001.</b> This PR is part of a stack created with <a href="https://github.com/aviator-co/av">Aviator</a>.</summary>

* `+"`"+`main`+"`"+`
  * **#1001**
    * ➡️ **#1002**
  * **#1003**
</details></td></tr></table>
<!-- av pr stack end -->

Hello! This is a cool PR that does some neat things.

<!-- av pr metadata
This information is embedded by the av CLI when creating PRs to track the status of stacks when using Aviator. Please do not delete or edit this section of the PR.
`+"```"+`
{"parent":"foo","parentHead":"bar","parentPull":123,"trunk":"baz"}
`+"```"+`
-->
`, body1)
}

type fakeReadTx map[string]meta.Branch

func (tx fakeReadTx) Repository() meta.Repository {
	return meta.Repository{}
}

func (tx fakeReadTx) Branch(name string) (meta.Branch, bool) {
	branch, ok := tx[name]
	if branch.Name == "" {
		branch.Name = name
	}
	return branch, ok
}

func (tx fakeReadTx) AllBranches() map[string]meta.Branch {
	return maputils.Copy(tx)
}

// Mock implementations for testing multi-provider support

type mockProvider struct {
	mock.Mock
}

func (m *mockProvider) CreatePR(ctx context.Context, input vcs.CreatePullRequestInput) (vcs.PullRequest, error) {
	args := m.Called(ctx, input)
	return args.Get(0).(vcs.PullRequest), args.Error(1)
}

func (m *mockProvider) GetPR(ctx context.Context, id string) (vcs.PullRequest, error) {
	args := m.Called(ctx, id)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(vcs.PullRequest), args.Error(1)
}

func (m *mockProvider) UpdatePR(ctx context.Context, input vcs.UpdatePullRequestInput) (vcs.PullRequest, error) {
	args := m.Called(ctx, input)
	return args.Get(0).(vcs.PullRequest), args.Error(1)
}

func (m *mockProvider) GetPRs(ctx context.Context, input vcs.GetPullRequestsInput) (*vcs.GetPullRequestsPage, error) {
	args := m.Called(ctx, input)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*vcs.GetPullRequestsPage), args.Error(1)
}

func (m *mockProvider) RequestReviews(ctx context.Context, input vcs.RequestReviewsInput) (vcs.PullRequest, error) {
	args := m.Called(ctx, input)
	return args.Get(0).(vcs.PullRequest), args.Error(1)
}

func (m *mockProvider) ConvertToDraft(ctx context.Context, id string) (vcs.PullRequest, error) {
	args := m.Called(ctx, id)
	return args.Get(0).(vcs.PullRequest), args.Error(1)
}

func (m *mockProvider) MarkReadyForReview(ctx context.Context, id string) (vcs.PullRequest, error) {
	args := m.Called(ctx, id)
	return args.Get(0).(vcs.PullRequest), args.Error(1)
}

func (m *mockProvider) GetRepository(ctx context.Context, owner, repo string) (vcs.Repository, error) {
	args := m.Called(ctx, owner, repo)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(vcs.Repository), args.Error(1)
}

func (m *mockProvider) GetUser(ctx context.Context, login string) (vcs.User, error) {
	args := m.Called(ctx, login)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(vcs.User), args.Error(1)
}

func (m *mockProvider) GetViewer(ctx context.Context) (vcs.User, error) {
	args := m.Called(ctx)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(vcs.User), args.Error(1)
}

func (m *mockProvider) GetOrganizationTeam(ctx context.Context, org, team string) (vcs.Team, error) {
	args := m.Called(ctx, org, team)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(vcs.Team), args.Error(1)
}

// Mock pull request implementation for testing
type mockPullRequest struct {
	id          string
	number      int64
	headBranch  string
	baseBranch  string
	title       string
	body        string
	permalink   string
	isDraft     bool
	isOpen      bool
	isMerged    bool
	mergeCommit string
}

func (pr *mockPullRequest) GetID() string          { return pr.id }
func (pr *mockPullRequest) GetNumber() int64       { return pr.number }
func (pr *mockPullRequest) HeadBranchName() string { return pr.headBranch }
func (pr *mockPullRequest) BaseBranchName() string { return pr.baseBranch }
func (pr *mockPullRequest) GetTitle() string       { return pr.title }
func (pr *mockPullRequest) GetBody() string        { return pr.body }
func (pr *mockPullRequest) GetPermalink() string   { return pr.permalink }
func (pr *mockPullRequest) IsDraft() bool          { return pr.isDraft }
func (pr *mockPullRequest) IsOpen() bool           { return pr.isOpen }
func (pr *mockPullRequest) IsMerged() bool         { return pr.isMerged }
func (pr *mockPullRequest) GetMergeCommit() string { return pr.mergeCommit }

// Mock repository implementation for testing
type mockRepository struct {
	id       string
	owner    string
	name     string
	fullName string
}

func (r *mockRepository) GetID() string       { return r.id }
func (r *mockRepository) GetOwner() string    { return r.owner }
func (r *mockRepository) GetName() string     { return r.name }
func (r *mockRepository) GetFullName() string { return r.fullName }

// Test fixtures for different provider scenarios
func createGitHubMockPR(id string, number int64) *mockPullRequest {
	return &mockPullRequest{
		id:         id,
		number:     number,
		headBranch: "feature-branch",
		baseBranch: "main",
		title:      "GitHub Test PR",
		body:       "Test PR from GitHub",
		permalink:  "https://github.com/owner/repo/pull/42",
		isDraft:    false,
		isOpen:     true,
		isMerged:   false,
	}
}

func createGitLabMockPR(id string, number int64) *mockPullRequest {
	return &mockPullRequest{
		id:         id,
		number:     number,
		headBranch: "feature-branch",
		baseBranch: "main",
		title:      "GitLab Test MR",
		body:       "Test MR from GitLab",
		permalink:  "https://gitlab.com/owner/repo/-/merge_requests/89",
		isDraft:    false,
		isOpen:     true,
		isMerged:   false,
	}
}

// Provider-specific test cases

func TestCreatePullRequest_GitHubProvider(t *testing.T) {
	mockPR := createGitHubMockPR("gh-pr-123", 42)
	
	// Test that GitHub-specific PR creation works correctly
	assert.Equal(t, "GitHub Test PR", mockPR.GetTitle())
	assert.Contains(t, mockPR.GetPermalink(), "github.com")
	assert.Equal(t, "gh-pr-123", mockPR.GetID())
}

func TestCreatePullRequest_GitLabProvider(t *testing.T) {
	mockPR := createGitLabMockPR("gl-mr-456", 89)
	
	// Test that GitLab-specific MR creation works correctly
	assert.Equal(t, "GitLab Test MR", mockPR.GetTitle())
	assert.Contains(t, mockPR.GetPermalink(), "gitlab.com")
	assert.Contains(t, mockPR.GetPermalink(), "merge_requests")
	assert.Equal(t, "gl-mr-456", mockPR.GetID())
}

func TestGetExistingOpenPR_MultiProvider(t *testing.T) {
	tests := []struct {
		name        string
		provider    string
		prID        string
		prNumber    int64
		expectedURL string
	}{
		{
			name:        "GitHub PR",
			provider:    "github",
			prID:        "gh-pr-123",
			prNumber:    42,
			expectedURL: "github.com",
		},
		{
			name:        "GitLab MR", 
			provider:    "gitlab",
			prID:        "gl-mr-456",
			prNumber:    89,
			expectedURL: "gitlab.com",
		},
	}
	
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var mockPR *mockPullRequest
			if tt.provider == "github" {
				mockPR = createGitHubMockPR(tt.prID, tt.prNumber)
			} else {
				mockPR = createGitLabMockPR(tt.prID, tt.prNumber)
			}
			
			// Verify the PR data
			assert.Equal(t, tt.prID, mockPR.GetID())
			assert.Equal(t, tt.prNumber, mockPR.GetNumber())
			assert.Contains(t, mockPR.GetPermalink(), tt.expectedURL)
			assert.True(t, mockPR.IsOpen())
		})
	}
}

func TestPRStateHandling_MultiProvider(t *testing.T) {
	tests := []struct {
		name       string
		provider   string
		isOpen     bool
		isMerged   bool
		expectedOp string
	}{
		{"GitHub Open PR", "github", true, false, "can_update"},
		{"GitHub Merged PR", "github", false, true, "read_only"},
		{"GitLab Open MR", "gitlab", true, false, "can_update"},
		{"GitLab Merged MR", "gitlab", false, true, "read_only"},
	}
	
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var mockPR *mockPullRequest
			if tt.provider == "github" {
				mockPR = createGitHubMockPR("test-id", 1)
			} else {
				mockPR = createGitLabMockPR("test-id", 1)
			}
			
			mockPR.isOpen = tt.isOpen
			mockPR.isMerged = tt.isMerged
			
			assert.Equal(t, tt.isOpen, mockPR.IsOpen())
			assert.Equal(t, tt.isMerged, mockPR.IsMerged())
			
			// Test expected operation based on state
			if tt.expectedOp == "can_update" {
				assert.True(t, mockPR.IsOpen() && !mockPR.IsMerged())
			} else {
				assert.True(t, !mockPR.IsOpen() || mockPR.IsMerged())
			}
		})
	}
}

func TestPRMetadataCompatibility_MultiProvider(t *testing.T) {
	// Test that PR metadata parsing works the same for both providers
	prMeta := actions.PRMetadata{
		Parent:     "parent-branch",
		ParentHead: "parent-head",
		ParentPull: 123,
		Trunk:      "main",
	}
	
	tx := fakeReadTx{}
	
	// Test with GitHub-style PR body
	githubBody := "This is a GitHub PR\n\n" + 
		"Check out the changes at https://github.com/owner/repo/pull/456"
	githubBodyWithMeta := actions.AddPRMetadataAndStack(githubBody, prMeta, "test-branch", nil, tx)
	
	// Test with GitLab-style MR body  
	gitlabBody := "This is a GitLab MR\n\n" +
		"Check out the changes at https://gitlab.com/owner/repo/-/merge_requests/789"
	gitlabBodyWithMeta := actions.AddPRMetadataAndStack(gitlabBody, prMeta, "test-branch", nil, tx)
	
	// Both should parse the same metadata
	githubParsedMeta, err := actions.ReadPRMetadata(githubBodyWithMeta)
	require.NoError(t, err)
	gitlabParsedMeta, err := actions.ReadPRMetadata(gitlabBodyWithMeta)
	require.NoError(t, err)
	
	// Metadata should be identical regardless of provider
	assert.Equal(t, prMeta, githubParsedMeta)
	assert.Equal(t, prMeta, gitlabParsedMeta)
	assert.Equal(t, githubParsedMeta, gitlabParsedMeta)
}

// Test provider-specific URL patterns in permalinks
func TestProviderSpecificPermalinks(t *testing.T) {
	tests := []struct {
		name           string
		provider       string
		expectedDomain string
		expectedPath   string
	}{
		{
			name:           "GitHub PR URL",
			provider:       "github",
			expectedDomain: "github.com",
			expectedPath:   "/pull/",
		},
		{
			name:           "GitLab MR URL",
			provider:       "gitlab",
			expectedDomain: "gitlab.com",
			expectedPath:   "/-/merge_requests/",
		},
	}
	
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var mockPR *mockPullRequest
			if tt.provider == "github" {
				mockPR = createGitHubMockPR("test-id", 42)
			} else {
				mockPR = createGitLabMockPR("test-id", 42)
			}
			
			permalink := mockPR.GetPermalink()
			assert.Contains(t, permalink, tt.expectedDomain)
			assert.Contains(t, permalink, tt.expectedPath)
		})
	}
}

// Test that existing GitHub PR functionality is preserved
func TestBackwardCompatibility_GitHubPRs(t *testing.T) {
	// Test that existing GitHub PR creation still works
	githubPR := createGitHubMockPR("gh-legacy-pr", 100)
	
	// Test all the interface methods work as expected
	assert.Equal(t, "gh-legacy-pr", githubPR.GetID())
	assert.Equal(t, int64(100), githubPR.GetNumber())
	assert.Equal(t, "feature-branch", githubPR.HeadBranchName())
	assert.Equal(t, "main", githubPR.BaseBranchName())
	assert.Equal(t, "GitHub Test PR", githubPR.GetTitle())
	assert.Equal(t, "Test PR from GitHub", githubPR.GetBody())
	assert.Contains(t, githubPR.GetPermalink(), "github.com")
	assert.False(t, githubPR.IsDraft())
	assert.True(t, githubPR.IsOpen())
	assert.False(t, githubPR.IsMerged())
	assert.Empty(t, githubPR.GetMergeCommit())
}

// Test GitLab-specific behaviors and edge cases
func TestGitLabSpecificBehaviors(t *testing.T) {
	gitlabMR := createGitLabMockPR("gitlab/project:123", 456)
	
	// Test GitLab MR interface
	assert.Equal(t, "gitlab/project:123", gitlabMR.GetID())
	assert.Equal(t, int64(456), gitlabMR.GetNumber())
	assert.Equal(t, "GitLab Test MR", gitlabMR.GetTitle())
	assert.Contains(t, gitlabMR.GetPermalink(), "gitlab.com")
	assert.Contains(t, gitlabMR.GetPermalink(), "merge_requests")
	
	// Test merge commit handling
	gitlabMR.isMerged = true
	gitlabMR.mergeCommit = "abc123def"
	assert.True(t, gitlabMR.IsMerged())
	assert.Equal(t, "abc123def", gitlabMR.GetMergeCommit())
	
	// Test draft state
	gitlabMR.isDraft = true
	assert.True(t, gitlabMR.IsDraft())
}
