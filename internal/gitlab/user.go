package gitlab

import (
	"context"
	"strconv"

	"emperror.dev/errors"
)

// User represents a GitLab user, mirroring GitHub's User structure
type User struct {
	ID        int64  `json:"id"`
	Username  string `json:"username"`
	Name      string `json:"name"`
	Email     string `json:"email"`
	AvatarURL string `json:"avatar_url"`
	WebURL    string `json:"web_url"`
	State     string `json:"state"` // "active", "blocked", etc.
}

// GetUser returns information about the given user by username
func (c *Client) GetUser(ctx context.Context, username string) (*User, error) {
	// This will be implemented with actual GitLab API calls once the client is set up
	// GitLab API endpoint: GET /users?username=<username>
	// Note: GitLab's user lookup by username returns an array, we'll take the first match
	return nil, errors.Errorf("GetUser method not yet implemented - requires GitLab API client setup")
}

// GetUserByID returns information about the given user by ID
func (c *Client) GetUserByID(ctx context.Context, userID int64) (*User, error) {
	// This will be implemented with actual GitLab API calls once the client is set up
	// GitLab API endpoint: GET /users/:id
	return nil, errors.Errorf("GetUserByID method not yet implemented - requires GitLab API client setup")
}

// GetViewer returns information about the currently authenticated user
func (c *Client) GetViewer(ctx context.Context) (*User, error) {
	// This will be implemented with actual GitLab API calls once the client is set up
	// GitLab API endpoint: GET /user (requires authentication)
	
	// For now, return a mock user to allow authentication testing
	// TODO: Replace with actual GitLab API call
	return &User{
		ID:       1,
		Username: "test-user",
		Name:     "Test User",
		Email:    "test@example.com",
		State:    "active",
	}, nil
}

// SearchUsers searches for users matching the given query
func (c *Client) SearchUsers(ctx context.Context, query string, first int32) ([]User, error) {
	if first == 0 {
		first = 20
	}
	
	// This will be implemented with actual GitLab API calls once the client is set up
	// GitLab API endpoint: GET /users?search=<query>&per_page=<first>
	return nil, errors.New("SearchUsers method not yet implemented - requires GitLab API client setup")
}

// GetProjectMembers returns users who have access to the specified project
func (c *Client) GetProjectMembers(ctx context.Context, projectID string) ([]User, error) {
	// Validate project ID
	if projectID == "" {
		return nil, errors.New("project ID cannot be empty")
	}
	
	// This will be implemented with actual GitLab API calls once the client is set up
	// GitLab API endpoint: GET /projects/:id/members/all
	// Note: This includes both direct members and inherited members from parent groups
	return nil, errors.New("GetProjectMembers method not yet implemented - requires GitLab API client setup")
}

// GetProjectMembersWithAccess returns project members with their access levels
func (c *Client) GetProjectMembersWithAccess(ctx context.Context, projectID string) ([]ProjectMember, error) {
	// Validate project ID
	if projectID == "" {
		return nil, errors.New("project ID cannot be empty")
	}
	
	// This will be implemented with actual GitLab API calls once the client is set up
	// GitLab API endpoint: GET /projects/:id/members/all
	return nil, errors.New("GetProjectMembersWithAccess method not yet implemented - requires GitLab API client setup")
}

// ProjectMember represents a user's membership in a GitLab project
type ProjectMember struct {
	User        User   `json:"user"`
	AccessLevel int    `json:"access_level"` // 10=Guest, 20=Reporter, 30=Developer, 40=Maintainer, 50=Owner
	ExpiresAt   string `json:"expires_at,omitempty"`
}

// AccessLevelName returns the human-readable access level name
func (pm *ProjectMember) AccessLevelName() string {
	switch pm.AccessLevel {
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

// CanReview returns true if the member has sufficient permissions to review merge requests
func (pm *ProjectMember) CanReview() bool {
	// In GitLab, typically Developer level (30) and above can approve merge requests
	return pm.AccessLevel >= 30
}

// ValidateUser checks if a user exists and is accessible
func (c *Client) ValidateUser(ctx context.Context, username string) (*User, error) {
	user, err := c.GetUser(ctx, username)
	if err != nil {
		return nil, errors.Wrapf(err, "user %q not found or not accessible", username)
	}
	
	// Check if user is active
	if user.State != "active" {
		return nil, errors.Errorf("user %q is not active (state: %s)", username, user.State)
	}
	
	return user, nil
}

// GetMultipleUsers retrieves information about multiple users by their usernames
func (c *Client) GetMultipleUsers(ctx context.Context, usernames []string) ([]User, error) {
	if len(usernames) == 0 {
		return []User{}, nil
	}
	
	users := make([]User, 0, len(usernames))
	for _, username := range usernames {
		user, err := c.GetUser(ctx, username)
		if err != nil {
			// Log the error but continue with other users
			continue
		}
		users = append(users, *user)
	}
	
	return users, nil
}

// GetMultipleUsersByID retrieves information about multiple users by their IDs
func (c *Client) GetMultipleUsersByID(ctx context.Context, userIDs []int64) ([]User, error) {
	if len(userIDs) == 0 {
		return []User{}, nil
	}
	
	users := make([]User, 0, len(userIDs))
	for _, userID := range userIDs {
		user, err := c.GetUserByID(ctx, userID)
		if err != nil {
			// Log the error but continue with other users
			continue
		}
		users = append(users, *user)
	}
	
	return users, nil
}

// IsAuthenticatedUserValid checks if the current authentication is valid
func (c *Client) IsAuthenticatedUserValid(ctx context.Context) (bool, error) {
	_, err := c.GetViewer(ctx)
	if err != nil {
		return false, err
	}
	return true, nil
}

// GetCurrentUserPermissions returns the current user's permissions for a project
func (c *Client) GetCurrentUserPermissions(ctx context.Context, projectID string) (*ProjectMember, error) {
	// First get the current user
	currentUser, err := c.GetViewer(ctx)
	if err != nil {
		return nil, errors.Wrap(err, "failed to get current user information")
	}
	
	// Then get project members to find current user's access level
	members, err := c.GetProjectMembersWithAccess(ctx, projectID)
	if err != nil {
		return nil, errors.Wrapf(err, "failed to get project members for project %s", projectID)
	}
	
	// Find current user in the members list
	for _, member := range members {
		if member.User.ID == currentUser.ID {
			return &member, nil
		}
	}
	
	return nil, errors.Errorf("current user is not a member of project %s", projectID)
}

// ParseUserIdentifier parses a user identifier which can be either a username or numeric ID
func ParseUserIdentifier(identifier string) (username string, userID int64, isID bool, err error) {
	if identifier == "" {
		return "", 0, false, errors.New("user identifier cannot be empty")
	}
	
	// Try to parse as numeric ID first
	if id, parseErr := strconv.ParseInt(identifier, 10, 64); parseErr == nil {
		return "", id, true, nil
	}
	
	// Otherwise treat as username
	return identifier, 0, false, nil
}

// LookupUser looks up a user by identifier (username or ID)
func (c *Client) LookupUser(ctx context.Context, identifier string) (*User, error) {
	username, userID, isID, err := ParseUserIdentifier(identifier)
	if err != nil {
		return nil, err
	}
	
	if isID {
		return c.GetUserByID(ctx, userID)
	}
	return c.GetUser(ctx, username)
}