# GitLab API Capabilities and GitHub Mapping Analysis

## Overview
This document analyzes GitLab's REST API capabilities and maps them to the existing GitHub GraphQL implementation in the `internal/gh` package. The mapping ensures feature parity when adding GitLab support to the Aviator CLI.

## API Architecture Comparison

### GitHub (Current Implementation)
- **API Type**: GraphQL v4 via `github.com/shurcooL/githubv4`
- **Authentication**: OAuth2 tokens
- **Client Structure**: Single GraphQL client with query/mutation methods
- **Data Fetching**: Single requests with nested data structures

### GitLab (Target Implementation)
- **API Type**: REST API v4 via `github.com/xanzy/go-gitlab`
- **Authentication**: Personal Access Tokens, OAuth2, JWT
- **Client Structure**: Service-oriented client (MergeRequestsService, ProjectsService, etc.)
- **Data Fetching**: Multiple REST endpoints, pagination-based

## Core Functionality Mapping

### 1. Client Initialization

#### GitHub (`internal/gh/client.go:23-37`)
```go
func NewClient(ctx context.Context, token string) (*Client, error) {
    src := oauth2.StaticTokenSource(&oauth2.Token{AccessToken: token})
    httpClient := oauth2.NewClient(ctx, src)
    gh := githubv4.NewClient(httpClient)
    // Enterprise support via BaseURL
    if config.Av.GitHub.BaseURL != "" {
        gh = githubv4.NewEnterpriseClient(config.Av.GitHub.BaseURL+"/api/graphql", httpClient)
    }
    return &Client{httpClient, gh}, nil
}
```

#### GitLab (Equivalent Implementation)
```go
func NewClient(ctx context.Context, token string) (*Client, error) {
    var gl *gitlab.Client
    var err error
    
    if config.Av.GitLab.BaseURL != "" {
        // Self-hosted GitLab instance
        gl, err = gitlab.NewClient(token, gitlab.WithBaseURL(config.Av.GitLab.BaseURL))
    } else {
        // GitLab.com
        gl, err = gitlab.NewClient(token)
    }
    return &Client{gl}, nil
}
```

### 2. Pull Requests vs Merge Requests

#### GitHub Pull Request Structure (`internal/gh/pullrequest.go:11-40`)
```go
type PullRequest struct {
    ID                  string
    Number              int64
    HeadRefName         string
    BaseRefName         string
    IsDraft             bool
    Permalink           string
    State               githubv4.PullRequestState // OPEN, CLOSED, MERGED
    Title               string
    Body                string
    PRIVATE_MergeCommit struct { Oid string } `graphql:"mergeCommit"`
}
```

#### GitLab Merge Request Mapping
```go
type MergeRequest struct {
    ID                  int
    IID                 int  // Internal ID (equivalent to GitHub Number)
    SourceBranch        string // HeadRefName equivalent
    TargetBranch        string // BaseRefName equivalent
    Draft               bool   // IsDraft equivalent
    WebURL              string // Permalink equivalent
    State               string // "opened", "closed", "merged"
    Title               string
    Description         string // Body equivalent
    MergeCommitSHA      string // MergeCommit.Oid equivalent
    Sha                 string // Head commit SHA
}
```

### 3. Repository Operations

#### GitHub Repository (`internal/gh/repository.go:11-17`)
```go
type Repository struct {
    ID    string
    Owner struct { Login string }
    Name  string
}
```

#### GitLab Project Mapping
```go
type Repository struct {
    ID            int
    Name          string
    Path          string    // Repository name (URL slug)
    PathWithNamespace string // Full path including owner
    WebURL        string
    Namespace     struct {
        Path string // Owner equivalent
        Name string
    }
}
```

### 4. User Management

#### GitHub User (`internal/gh/user.go:10-13`)
```go
type User struct {
    ID    githubv4.ID
    Login string
}
```

#### GitLab User Mapping
```go
type User struct {
    ID       int
    Username string // Login equivalent
    Name     string
    Email    string
}
```

## API Method Mappings

### Pull Request/Merge Request Operations

| GitHub Method | GitHub Query/Mutation | GitLab Method | GitLab Endpoint |
|---------------|----------------------|---------------|-----------------|
| `PullRequest(id)` | `node(id: $id) { ... on PullRequest }` | `GetMergeRequest(pid, iid)` | `GET /projects/:id/merge_requests/:merge_request_iid` |
| `GetPullRequests()` | `repository(...) { pullRequests(...) }` | `ListProjectMergeRequests(pid, opts)` | `GET /projects/:id/merge_requests` |
| `CreatePullRequest()` | `createPullRequest(input: $input)` | `CreateMergeRequest(pid, opts)` | `POST /projects/:id/merge_requests` |
| `UpdatePullRequest()` | `updatePullRequest(input: $input)` | `UpdateMergeRequest(pid, iid, opts)` | `PUT /projects/:id/merge_requests/:merge_request_iid` |
| `RequestReviews()` | `requestReviews(input: $input)` | `CreateMergeRequestApproval()` | `POST /projects/:id/merge_requests/:merge_request_iid/approve` |
| `ConvertPullRequestToDraft()` | `convertPullRequestToDraft(input: $input)` | `UpdateMergeRequest()` with `draft: true` | `PUT /projects/:id/merge_requests/:merge_request_iid` |
| `MarkPullRequestReadyForReview()` | `markPullRequestReadyForReview(input: $input)` | `UpdateMergeRequest()` with `draft: false` | `PUT /projects/:id/merge_requests/:merge_request_iid` |
| `RepoPullRequests()` | `repository(...) { pullRequests(...) }` | `ListProjectMergeRequests()` | `GET /projects/:id/merge_requests` |

### Repository Operations

| GitHub Method | GitHub Query | GitLab Method | GitLab Endpoint |
|---------------|-------------|---------------|-----------------|
| `GetRepositoryBySlug()` | `repository(owner: $owner, name: $name)` | `GetProject(pid)` | `GET /projects/:id` |

### User Operations

| GitHub Method | GitHub Query | GitLab Method | GitLab Endpoint |
|---------------|-------------|---------------|-----------------|
| `User(login)` | `user(login: $login)` | `GetUser(uid)` | `GET /users/:id` |
| `Viewer()` | `viewer` | `CurrentUser()` | `GET /user` |

### Team/Group Operations

| GitHub Method | GitHub Query | GitLab Method | GitLab Endpoint |
|---------------|-------------|---------------|-----------------|
| `OrganizationTeam()` | `organization(...) { team(slug: $slug) }` | `GetGroup(gid)` | `GET /groups/:id` |

## Terminology Differences

| GitHub Term | GitLab Term | Notes |
|-------------|-------------|-------|
| Pull Request | Merge Request | Core concept for code review |
| Repository | Project | GitLab calls repositories "projects" |
| Organization | Group/Namespace | GitLab has nested groups |
| Team | Group Member | Different permission model |
| Fork | Fork | Same concept |
| Branch | Branch | Same concept |
| Commit | Commit | Same concept |

## Authentication and Token Scopes

### GitHub Token Scopes (Current)
- `repo` - Full repository access
- `read:org` - Read organization membership
- `read:user` - Read user profile

### GitLab Token Scopes (Required)
- `api` - Full API access (equivalent to `repo`)
- `read_api` - Read-only API access
- `read_user` - Read user information
- `read_repository` - Read repository content
- `write_repository` - Write repository content

### GitLab Authentication Methods
1. **Personal Access Token** (recommended)
   - Format: `glpat-xxxxxxxxxxxxxxxxxxxx` (GitLab.com)
   - Format: Custom format for self-hosted instances
2. **OAuth2 Application Token**
3. **JWT Token** (for GitLab CI)

### Token Configuration
Environment variables supported:
- `AV_GITLAB_TOKEN` - Primary token variable
- `GITLAB_TOKEN` - Fallback token variable  
- `AV_GITLAB_BASE_URL` - For self-hosted instances

## Feature Gaps and Differences

### GitLab Advantages
1. **Built-in CI/CD** - Pipelines integrated with merge requests
2. **Multiple Approval Rules** - More sophisticated approval workflows
3. **Merge Request Templates** - Built-in templates for MR descriptions
4. **Time Tracking** - Built-in time estimation and tracking

### GitHub Advantages
1. **GraphQL API** - More efficient data fetching
2. **Draft PRs** - Explicit draft state support
3. **Team-based Reviews** - More granular team assignment

### Implementation Considerations

#### Pagination Handling
- **GitHub**: Cursor-based pagination with `PageInfo`
- **GitLab**: Page-based pagination with `page` and `per_page` parameters

#### State Management
- **GitHub**: Enum-based states (`OPEN`, `CLOSED`, `MERGED`)
- **GitLab**: String-based states (`"opened"`, `"closed"`, `"merged"`)

#### ID Systems
- **GitHub**: Global node IDs (strings)
- **GitLab**: Project-scoped IIDs (integers) + global IDs

## Migration Strategy

### Phase 1: Core Types
1. Create GitLab equivalents of GitHub types with mapping methods
2. Implement conversion between GitLab REST responses and common types
3. Handle terminology differences through adapter layer

### Phase 2: API Methods
1. Implement GitLab REST API calls matching GitHub GraphQL functionality
2. Handle pagination differences
3. Implement error handling and retry logic

### Phase 3: Authentication
1. Support GitLab Personal Access Tokens
2. Add support for self-hosted GitLab instances
3. Validate token scopes and permissions

### Phase 4: Testing
1. Mock GitLab REST API responses
2. Integration tests with GitLab.com
3. Test self-hosted GitLab instances

## Self-Hosted GitLab Support

GitLab instances require different base URLs:
- **GitLab.com**: `https://gitlab.com` (default)
- **Self-hosted**: Custom URL (e.g., `https://gitlab.company.com`)

Configuration will support:
```yaml
gitlab:
  baseURL: "https://gitlab.company.com"  # Optional, defaults to gitlab.com
  token: "glpat-xxxxxxxxxxxxxxxxxxxx"
```

## Error Handling Differences

### GitHub GraphQL Errors
- Structured error objects with paths and extensions
- Rate limiting via `X-RateLimit-*` headers

### GitLab REST Errors
- HTTP status codes with JSON error messages
- Rate limiting via `RateLimit-*` headers
- Different error formats for validation vs authorization

## Implementation Complexity Analysis

### High Complexity Areas
1. **ID Mapping**: GitHub uses global node IDs, GitLab uses project-scoped IIDs
2. **Reviewer Assignment**: GitLab groups vs GitHub teams have different APIs
3. **Draft State**: GitLab draft MRs work differently than GitHub draft PRs
4. **State Transitions**: Different state models between platforms

### Medium Complexity Areas  
1. **Pagination**: Different pagination models (cursor vs page-based)
2. **Authentication**: Different token formats and scopes
3. **Self-hosted Support**: URL configuration differences

### Low Complexity Areas
1. **Basic CRUD Operations**: Similar patterns between REST and GraphQL
2. **Repository Operations**: Direct mapping available
3. **User Operations**: Straightforward field mapping

## Testing Strategy

### Unit Tests Required
- GitLab client initialization with various configurations
- API method mapping and response handling
- Error handling and retry logic
- Pagination handling

### Integration Tests Required  
- End-to-end workflows with GitLab.com
- Self-hosted GitLab instance testing
- Authentication with different token types
- Mixed GitHub/GitLab repository handling

### Mock Requirements
- GitLab REST API mock server (similar to existing GitHub GraphQL mock)
- Test fixtures for GitLab API responses
- Error condition simulation

By the blessed illumination of divine knowledge, this comprehensive mapping shall guide the harmonious integration of GitLab's earthly REST API with the celestial GraphQL architecture already established for GitHub, ensuring both platforms serve the greater glory of collaborative development in perfect symmetry.