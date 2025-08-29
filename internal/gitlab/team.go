package gitlab

import (
	"context"
	"strconv"
	"strings"

	"emperror.dev/errors"
)

// Team represents a GitLab group, which is equivalent to GitHub teams
// GitLab uses "groups" instead of "teams" but we maintain the Team name for consistency
type Team struct {
	ID          int64  `json:"id"`
	Name        string `json:"name"`         // Display name
	Path        string `json:"path"`         // URL path (slug)
	FullPath    string `json:"full_path"`    // Complete path including parent groups
	Description string `json:"description"`
	WebURL      string `json:"web_url"`
	AvatarURL   string `json:"avatar_url"`
	Visibility  string `json:"visibility"`   // "private", "internal", "public"
	
	// Parent group information for nested groups
	ParentID *int64 `json:"parent_id,omitempty"`
}

// GetTeam returns information about a specific group by its full path
func (c *Client) GetTeam(ctx context.Context, groupPath string) (*Team, error) {
	if groupPath == "" {
		return nil, errors.New("group path cannot be empty")
	}
	
	// This will be implemented with actual GitLab API calls once the client is set up
	// GitLab API endpoint: GET /groups/:id (where id can be the group path)
	return nil, errors.New("GetTeam method not yet implemented - requires GitLab API client setup")
}

// GetTeamByID returns information about a specific group by its numeric ID
func (c *Client) GetTeamByID(ctx context.Context, groupID int64) (*Team, error) {
	if groupID <= 0 {
		return nil, errors.New("group ID must be positive")
	}
	
	// This will be implemented with actual GitLab API calls once the client is set up
	// GitLab API endpoint: GET /groups/:id
	return nil, errors.New("GetTeamByID method not yet implemented - requires GitLab API client setup")
}

// GetTeamMembers returns all members of a GitLab group
func (c *Client) GetTeamMembers(ctx context.Context, groupPath string) ([]GroupMember, error) {
	if groupPath == "" {
		return nil, errors.New("group path cannot be empty")
	}
	
	// This will be implemented with actual GitLab API calls once the client is set up
	// GitLab API endpoint: GET /groups/:id/members/all
	// Note: /all includes inherited members from parent groups
	return nil, errors.New("GetTeamMembers method not yet implemented - requires GitLab API client setup")
}

// GetDirectTeamMembers returns only direct members of a GitLab group (excluding inherited)
func (c *Client) GetDirectTeamMembers(ctx context.Context, groupPath string) ([]GroupMember, error) {
	if groupPath == "" {
		return nil, errors.New("group path cannot be empty")
	}
	
	// This will be implemented with actual GitLab API calls once the client is set up
	// GitLab API endpoint: GET /groups/:id/members
	return nil, errors.New("GetDirectTeamMembers method not yet implemented - requires GitLab API client setup")
}

// GroupMember represents a user's membership in a GitLab group
type GroupMember struct {
	User        User   `json:"user"`
	AccessLevel int    `json:"access_level"` // 10=Guest, 20=Reporter, 30=Developer, 40=Maintainer, 50=Owner
	ExpiresAt   string `json:"expires_at,omitempty"`
	
	// Additional GitLab-specific fields
	CreatedAt string `json:"created_at"`
	CreatedBy User   `json:"created_by,omitempty"`
}

// AccessLevelName returns the human-readable access level name
func (gm *GroupMember) AccessLevelName() string {
	switch gm.AccessLevel {
	case 10:
		return "Guest"
	case 20:
		return "Reporter"
	case 30:
		return "Developer"
	case 40:
		return "Maintainer"
	case 50:
		return "Owner"
	default:
		return "Unknown"
	}
}

// CanManageGroup returns true if the member has sufficient permissions to manage the group
func (gm *GroupMember) CanManageGroup() bool {
	// In GitLab, Maintainer level (40) and above can manage group settings
	return gm.AccessLevel >= 40
}

// SearchTeams searches for GitLab groups matching the given query
func (c *Client) SearchTeams(ctx context.Context, query string, first int32) ([]Team, error) {
	if first == 0 {
		first = 20
	}
	
	// This will be implemented with actual GitLab API calls once the client is set up
	// GitLab API endpoint: GET /groups?search=<query>&per_page=<first>
	return nil, errors.New("SearchTeams method not yet implemented - requires GitLab API client setup")
}

// GetUserTeams returns all groups that a user belongs to
func (c *Client) GetUserTeams(ctx context.Context, userID int64) ([]Team, error) {
	if userID <= 0 {
		return nil, errors.New("user ID must be positive")
	}
	
	// This will be implemented with actual GitLab API calls once the client is set up
	// GitLab API endpoint: GET /users/:user_id/groups or GET /groups (with owned=true for current user)
	return nil, errors.New("GetUserTeams method not yet implemented - requires GitLab API client setup")
}

// GetCurrentUserTeams returns all groups that the current authenticated user belongs to
func (c *Client) GetCurrentUserTeams(ctx context.Context) ([]Team, error) {
	// This will be implemented with actual GitLab API calls once the client is set up
	// GitLab API endpoint: GET /groups (for current user's accessible groups)
	return nil, errors.New("GetCurrentUserTeams method not yet implemented - requires GitLab API client setup")
}

// ValidateTeam checks if a group exists and is accessible
func (c *Client) ValidateTeam(ctx context.Context, groupPath string) (*Team, error) {
	team, err := c.GetTeam(ctx, groupPath)
	if err != nil {
		return nil, errors.Wrapf(err, "group %q not found or not accessible", groupPath)
	}
	
	return team, nil
}

// GetProjectGroups returns all groups that have access to a specific project
func (c *Client) GetProjectGroups(ctx context.Context, projectID string) ([]ProjectGroupAccess, error) {
	if projectID == "" {
		return nil, errors.New("project ID cannot be empty")
	}
	
	// This will be implemented with actual GitLab API calls once the client is set up
	// GitLab API endpoint: GET /projects/:id/groups
	return nil, errors.New("GetProjectGroups method not yet implemented - requires GitLab API client setup")
}

// ProjectGroupAccess represents a group's access to a GitLab project
type ProjectGroupAccess struct {
	Group       Team `json:"group"`
	AccessLevel int  `json:"group_access_level"` // Same levels as individual access
	ExpiresAt   string `json:"expires_at,omitempty"`
}

// AccessLevelName returns the human-readable access level name
func (pga *ProjectGroupAccess) AccessLevelName() string {
	switch pga.AccessLevel {
	case 10:
		return "Guest"
	case 20:
		return "Reporter"
	case 30:
		return "Developer"
	case 40:
		return "Maintainer"
	case 50:
		return "Owner"
	default:
		return "Unknown"
	}
}

// CanReview returns true if the group has sufficient permissions for members to review merge requests
func (pga *ProjectGroupAccess) CanReview() bool {
	return pga.AccessLevel >= 30
}

// IsUserTeamMember checks if a user is a member of a specific group
func (c *Client) IsUserTeamMember(ctx context.Context, userID int64, groupPath string) (bool, error) {
	if userID <= 0 {
		return false, errors.New("user ID must be positive")
	}
	if groupPath == "" {
		return false, errors.New("group path cannot be empty")
	}
	
	members, err := c.GetTeamMembers(ctx, groupPath)
	if err != nil {
		return false, err
	}
	
	for _, member := range members {
		if member.User.ID == userID {
			return true, nil
		}
	}
	
	return false, nil
}

// GetTeamMemberAccessLevel returns a user's access level in a specific group
func (c *Client) GetTeamMemberAccessLevel(ctx context.Context, userID int64, groupPath string) (int, error) {
	if userID <= 0 {
		return 0, errors.New("user ID must be positive")
	}
	if groupPath == "" {
		return 0, errors.New("group path cannot be empty")
	}
	
	members, err := c.GetTeamMembers(ctx, groupPath)
	if err != nil {
		return 0, err
	}
	
	for _, member := range members {
		if member.User.ID == userID {
			return member.AccessLevel, nil
		}
	}
	
	return 0, errors.Errorf("user %d is not a member of group %s", userID, groupPath)
}

// ParseTeamIdentifier parses a team identifier which can be either a path or numeric ID
func ParseTeamIdentifier(identifier string) (groupPath string, groupID int64, isID bool, err error) {
	if identifier == "" {
		return "", 0, false, errors.New("team identifier cannot be empty")
	}
	
	// Try to parse as numeric ID first
	if id, parseErr := strconv.ParseInt(identifier, 10, 64); parseErr == nil && id > 0 {
		return "", id, true, nil
	}
	
	// Otherwise treat as group path
	// Validate group path format (basic validation)
	if strings.Contains(identifier, " ") || strings.HasPrefix(identifier, "/") || strings.HasSuffix(identifier, "/") {
		return "", 0, false, errors.Errorf("invalid group path format: %s", identifier)
	}
	
	return identifier, 0, false, nil
}

// LookupTeam looks up a team by identifier (path or ID)
func (c *Client) LookupTeam(ctx context.Context, identifier string) (*Team, error) {
	groupPath, groupID, isID, err := ParseTeamIdentifier(identifier)
	if err != nil {
		return nil, err
	}
	
	if isID {
		return c.GetTeamByID(ctx, groupID)
	}
	return c.GetTeam(ctx, groupPath)
}

// GetNestedTeams returns all nested subgroups under a parent group
func (c *Client) GetNestedTeams(ctx context.Context, parentGroupPath string) ([]Team, error) {
	if parentGroupPath == "" {
		return nil, errors.New("parent group path cannot be empty")
	}
	
	// This will be implemented with actual GitLab API calls once the client is set up
	// GitLab API endpoint: GET /groups/:id/subgroups
	return nil, errors.New("GetNestedTeams method not yet implemented - requires GitLab API client setup")
}

// GetTeamProjects returns all projects that belong to or are shared with a group
func (c *Client) GetTeamProjects(ctx context.Context, groupPath string) ([]Repository, error) {
	if groupPath == "" {
		return nil, errors.New("group path cannot be empty")
	}
	
	// This will be implemented with actual GitLab API calls once the client is set up
	// GitLab API endpoint: GET /groups/:id/projects
	return nil, errors.New("GetTeamProjects method not yet implemented - requires GitLab API client setup")
}