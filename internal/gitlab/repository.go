package gitlab

import (
	"context"
	"fmt"
	"net/url"
	"regexp"
	"strconv"
	"strings"

	"emperror.dev/errors"
)

// Client represents the GitLab API client (will be implemented in client.go)
type Client struct {
	// Implementation will be added when client.go is created
}

// User represents a GitLab user (will be implemented in user.go)  
type User struct {
	ID       int64  `json:"id"`
	Username string `json:"username"`
	Name     string `json:"name"`
	Email    string `json:"email"`
}

// Repository represents a GitLab project/repository, mirroring GitHub's Repository structure
type Repository struct {
	ID          int64  `json:"id"`           // GitLab uses numeric IDs
	Name        string `json:"name"`         // Project name
	Path        string `json:"path"`         // URL path (slug)
	PathWithNamespace string `json:"path_with_namespace"` // Full path including namespace
	WebURL      string `json:"web_url"`      // Full web URL to the project
	SSHURL      string `json:"ssh_url_to_repo"`    // SSH clone URL
	HTTPURL     string `json:"http_url_to_repo"`   // HTTP clone URL
	DefaultBranch string `json:"default_branch"`    // Default branch name
	Description string `json:"description"`  // Project description
	
	// Namespace information (similar to GitHub Owner)
	Namespace struct {
		ID       int64  `json:"id"`
		Name     string `json:"name"`     // Display name
		Path     string `json:"path"`     // URL path (username/group name)
		Kind     string `json:"kind"`     // "user" or "group"
		FullPath string `json:"full_path"`
	} `json:"namespace"`
	
	// Fork information
	ForkedFromProject *Repository `json:"forked_from_project,omitempty"`
}

// Owner returns the namespace path, similar to GitHub's Owner.Login
func (r *Repository) Owner() string {
	return r.Namespace.Path
}

// FullName returns the full project path (namespace/project)
func (r *Repository) FullName() string {
	return r.PathWithNamespace
}

// IsFork returns true if this repository is a fork
func (r *Repository) IsFork() bool {
	return r.ForkedFromProject != nil
}

// ParentRepository returns the parent repository if this is a fork
func (r *Repository) ParentRepository() *Repository {
	return r.ForkedFromProject
}

// GitLabURL represents a parsed GitLab repository URL
type GitLabURL struct {
	Host      string // GitLab instance hostname
	Namespace string // User or group name  
	Project   string // Project name
	IsSSH     bool   // True if SSH URL, false if HTTP(S)
}

// ParseGitLabURL parses various GitLab URL formats and extracts repository information
func ParseGitLabURL(rawURL string) (*GitLabURL, error) {
	// Handle SSH URLs: git@gitlab.com:namespace/project.git
	sshPattern := regexp.MustCompile(`^git@([^:]+):([^/]+)/(.+?)(?:\.git)?/?$`)
	if matches := sshPattern.FindStringSubmatch(rawURL); matches != nil {
		return &GitLabURL{
			Host:      matches[1],
			Namespace: matches[2],
			Project:   matches[3],
			IsSSH:     true,
		}, nil
	}
	
	// Handle HTTP(S) URLs
	parsedURL, err := url.Parse(rawURL)
	if err != nil {
		return nil, errors.Wrapf(err, "invalid URL format: %s", rawURL)
	}
	
	// Extract namespace and project from path
	pathParts := strings.Split(strings.Trim(parsedURL.Path, "/"), "/")
	if len(pathParts) < 2 {
		return nil, errors.Errorf("invalid GitLab URL format: expected /namespace/project, got %s", parsedURL.Path)
	}
	
	// Remove .git suffix if present
	project := pathParts[len(pathParts)-1]
	project = strings.TrimSuffix(project, ".git")
	
	// For URLs like /group/subgroup/project, we need to join all but the last part as namespace
	namespace := strings.Join(pathParts[:len(pathParts)-1], "/")
	
	return &GitLabURL{
		Host:      parsedURL.Host,
		Namespace: namespace,
		Project:   project,
		IsSSH:     false,
	}, nil
}

// IsGitLabURL checks if a URL appears to be a GitLab repository URL
func IsGitLabURL(rawURL string) bool {
	// Check for SSH format
	if matched, _ := regexp.MatchString(`^git@[^:]+:[^/]+/.+`, rawURL); matched {
		return true
	}
	
	// Check for HTTP(S) format
	parsedURL, err := url.Parse(rawURL)
	if err != nil {
		return false
	}
	
	// Must have a path with at least namespace/project
	pathParts := strings.Split(strings.Trim(parsedURL.Path, "/"), "/")
	return len(pathParts) >= 2
}

// DetectGitLabRepository detects if a git remote URL points to a GitLab repository
func DetectGitLabRepository(remoteURL string) (*GitLabURL, bool) {
	if !IsGitLabURL(remoteURL) {
		return nil, false
	}
	
	gitlabURL, err := ParseGitLabURL(remoteURL)
	if err != nil {
		return nil, false
	}
	
	return gitlabURL, true
}

// GetRepository retrieves project details from GitLab
func (c *Client) GetRepository(ctx context.Context, projectID string) (*Repository, error) {
	// This will be implemented with actual GitLab API calls once the client is set up
	// For now, returning a placeholder error
	return nil, errors.New("GetRepository method not yet implemented - requires GitLab API client setup")
}

// GetRepositoryBySlug retrieves a GitLab project by its namespace/project path
func (c *Client) GetRepositoryBySlug(ctx context.Context, slug string) (*Repository, error) {
	// Parse the slug to ensure it's valid
	parts := strings.Split(slug, "/")
	if len(parts) < 2 {
		return nil, errors.Errorf(
			"unable to parse repository slug (expected <namespace>/<project>): %q",
			slug,
		)
	}
	
	// GitLab allows nested groups, so we need to handle cases like group/subgroup/project
	namespace := strings.Join(parts[:len(parts)-1], "/")
	project := parts[len(parts)-1]
	
	if namespace == "" || project == "" {
		return nil, errors.Errorf(
			"invalid repository slug format: %q",
			slug,
		)
	}
	
	// This will be implemented with actual GitLab API calls once the client is set up
	// The GitLab API endpoint would be: GET /projects/:id where id can be namespace%2Fproject
	return nil, errors.New("GetRepositoryBySlug method not yet implemented - requires GitLab API client setup")
}

// GetRepositoryByURL retrieves a GitLab project by parsing its URL
func (c *Client) GetRepositoryByURL(ctx context.Context, repoURL string) (*Repository, error) {
	gitlabURL, err := ParseGitLabURL(repoURL)
	if err != nil {
		return nil, errors.Wrapf(err, "failed to parse GitLab URL: %s", repoURL)
	}
	
	// Convert to slug format for API call
	slug := fmt.Sprintf("%s/%s", gitlabURL.Namespace, gitlabURL.Project)
	return c.GetRepositoryBySlug(ctx, slug)
}

// GetRepositoryForkParent retrieves the parent repository if this is a fork
func (c *Client) GetRepositoryForkParent(ctx context.Context, repo *Repository) (*Repository, error) {
	if !repo.IsFork() {
		return nil, errors.New("repository is not a fork")
	}
	
	// If fork information is already populated, return it
	if repo.ForkedFromProject != nil {
		return repo.ForkedFromProject, nil
	}
	
	// Otherwise, fetch the parent repository information
	// This would require an additional API call to get full parent details
	return nil, errors.New("GetRepositoryForkParent method not yet implemented - requires GitLab API client setup")
}

// ValidateRepository checks if a repository exists and is accessible
func (c *Client) ValidateRepository(ctx context.Context, slug string) error {
	_, err := c.GetRepositoryBySlug(ctx, slug)
	return err
}

// GetRepositoryByNumericID retrieves a GitLab project by its numeric ID
func (c *Client) GetRepositoryByNumericID(ctx context.Context, id int64) (*Repository, error) {
	// Convert numeric ID to string for API call
	projectID := strconv.FormatInt(id, 10)
	return c.GetRepository(ctx, projectID)
}

// RepositorySearchOpts represents options for searching repositories
type RepositorySearchOpts struct {
	Query       string   // Search query
	Namespace   string   // Limit to specific namespace
	Visibility  string   // "private", "internal", "public"
	Archived    *bool    // Include/exclude archived projects
	OrderBy     string   // "id", "name", "path", "created_at", "updated_at", "last_activity_at"
	Sort        string   // "asc", "desc"
	First       int32    // Pagination limit
	After       string   // Pagination cursor
}

// SearchRepositoriesResponse represents the response from repository search
type SearchRepositoriesResponse struct {
	PageInfo
	TotalCount   int64
	Repositories []Repository
}

// SearchRepositories searches for repositories matching the given criteria
func (c *Client) SearchRepositories(
	ctx context.Context,
	opts RepositorySearchOpts,
) (*SearchRepositoriesResponse, error) {
	if opts.First == 0 {
		opts.First = 20
	}
	
	// This will be implemented with actual GitLab API calls once the client is set up
	// GitLab API endpoint: GET /projects with search parameters
	return nil, errors.New("SearchRepositories method not yet implemented - requires GitLab API client setup")
}

// GetRepositoryCollaborators retrieves the list of users who have access to the repository
func (c *Client) GetRepositoryCollaborators(ctx context.Context, projectID string) ([]User, error) {
	// This will be implemented with actual GitLab API calls once the client is set up
	// GitLab API endpoint: GET /projects/:id/members
	return nil, errors.New("GetRepositoryCollaborators method not yet implemented - requires GitLab API client setup")
}