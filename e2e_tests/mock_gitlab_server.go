package e2e_tests

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strconv"
	"strings"
	"testing"
)

// GitLab API endpoints and responses
const (
	gitLabAPIPrefix = "/api/v4"
)

func RunMockGitLabServer(t *testing.T) *mockGitLabServer {
	t.Helper()
	s := &mockGitLabServer{t: t, Server: nil}
	s.Server = httptest.NewServer(s)
	return s
}

type mockGitLabServer struct {
	t *testing.T

	mergeRequests []mockMR
	projects      []mockProject
	users         []mockUser

	*httptest.Server
}

type mockMR struct {
	ID              int64
	IID             int64
	ProjectID       int64
	Title           string
	Description     string
	SourceBranch    string
	TargetBranch    string
	State           string
	Draft           bool
	WebURL          string
	MergeCommitSHA  string
	ClosedCommitSHA string
}

type mockProject struct {
	ID                int64
	Name              string
	Path              string
	PathWithNamespace string
	WebURL            string
	HTTPURLToRepo     string
	SSHURLToRepo      string
	DefaultBranch     string
	Namespace         mockNamespace
}

type mockNamespace struct {
	ID       int64
	Name     string
	Path     string
	Kind     string
	FullPath string
}

type mockUser struct {
	ID       int64
	Username string
	Name     string
	Email    string
	State    string
	WebURL   string
}

// GitLab API response structures
type gitLabMRResponse struct {
	ID              int64  `json:"id"`
	IID             int64  `json:"iid"`
	ProjectID       int64  `json:"project_id"`
	Title           string `json:"title"`
	Description     string `json:"description"`
	SourceBranch    string `json:"source_branch"`
	TargetBranch    string `json:"target_branch"`
	State           string `json:"state"`
	Draft           bool   `json:"draft"`
	WebURL          string `json:"web_url"`
	MergeCommitSHA  string `json:"merge_commit_sha,omitempty"`
	ClosedCommitSHA string `json:"closed_commit_sha,omitempty"`
}

type gitLabProjectResponse struct {
	ID                int64           `json:"id"`
	Name              string          `json:"name"`
	Path              string          `json:"path"`
	PathWithNamespace string          `json:"path_with_namespace"`
	WebURL            string          `json:"web_url"`
	HTTPURLToRepo     string          `json:"http_url_to_repo"`
	SSHURLToRepo      string          `json:"ssh_url_to_repo"`
	DefaultBranch     string          `json:"default_branch"`
	Namespace         mockNamespace   `json:"namespace"`
}

func (s *mockGitLabServer) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	// Set GitLab API headers
	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("X-GitLab-Feature-Category", "continuous_integration")
	
	path := strings.TrimPrefix(r.URL.Path, gitLabAPIPrefix)
	
	// Route requests to appropriate handlers
	switch {
	case strings.HasPrefix(path, "/projects/") && strings.Contains(path, "/merge_requests"):
		s.handleMergeRequestsAPI(w, r, path)
	case strings.HasPrefix(path, "/projects/"):
		s.handleProjectsAPI(w, r, path)
	case strings.HasPrefix(path, "/user"):
		s.handleUserAPI(w, r, path)
	default:
		s.t.Logf("Unhandled GitLab API request: %s %s", r.Method, r.URL.Path)
		w.WriteHeader(http.StatusNotFound)
		json.NewEncoder(w).Encode(map[string]string{"message": "404 Not Found"})
	}
}

func (s *mockGitLabServer) handleMergeRequestsAPI(w http.ResponseWriter, r *http.Request, path string) {
	// Extract project ID from path: /projects/:id/merge_requests
	pathParts := strings.Split(strings.Trim(path, "/"), "/")
	if len(pathParts) < 3 {
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	projectIDStr := pathParts[1]
	projectID, err := strconv.ParseInt(projectIDStr, 10, 64)
	if err != nil {
		// Handle project path format (namespace/project)
		projectID = 1 // Default for testing
	}

	switch r.Method {
	case http.MethodGet:
		// GET /projects/:id/merge_requests or GET /projects/:id/merge_requests/:merge_request_iid
		if len(pathParts) > 3 {
			// Get specific merge request
			iidStr := pathParts[3]
			iid, err := strconv.ParseInt(iidStr, 10, 64)
			if err != nil {
				w.WriteHeader(http.StatusBadRequest)
				return
			}
			s.getMergeRequest(w, projectID, iid)
		} else {
			// List merge requests
			s.listMergeRequests(w, projectID, r)
		}
	case http.MethodPost:
		// POST /projects/:id/merge_requests
		s.createMergeRequest(w, r, projectID)
	case http.MethodPut:
		// PUT /projects/:id/merge_requests/:merge_request_iid
		if len(pathParts) > 3 {
			iidStr := pathParts[3]
			iid, err := strconv.ParseInt(iidStr, 10, 64)
			if err != nil {
				w.WriteHeader(http.StatusBadRequest)
				return
			}
			s.updateMergeRequest(w, r, projectID, iid)
		}
	default:
		w.WriteHeader(http.StatusMethodNotAllowed)
	}
}

func (s *mockGitLabServer) handleProjectsAPI(w http.ResponseWriter, r *http.Request, path string) {
	if r.Method != http.MethodGet {
		w.WriteHeader(http.StatusMethodNotAllowed)
		return
	}

	// GET /projects/:id
	pathParts := strings.Split(strings.Trim(path, "/"), "/")
	if len(pathParts) < 2 {
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	projectIDStr := pathParts[1]
	projectID, err := strconv.ParseInt(projectIDStr, 10, 64)
	if err != nil {
		// Handle namespace/project format
		projectID = 1 // Default for testing
	}

	s.getProject(w, projectID)
}

func (s *mockGitLabServer) handleUserAPI(w http.ResponseWriter, r *http.Request, path string) {
	if r.Method != http.MethodGet {
		w.WriteHeader(http.StatusMethodNotAllowed)
		return
	}

	// Return current user info
	user := mockUser{
		ID:       100,
		Username: "test-user",
		Name:     "Test User",
		Email:    "test@example.com",
		State:    "active",
		WebURL:   fmt.Sprintf("%s/test-user", strings.TrimSuffix(s.URL, gitLabAPIPrefix)),
	}

	if err := json.NewEncoder(w).Encode(user); err != nil {
		s.t.Logf("Failed to encode user response: %v", err)
		w.WriteHeader(http.StatusInternalServerError)
	}
}

func (s *mockGitLabServer) getMergeRequest(w http.ResponseWriter, projectID, iid int64) {
	for _, mr := range s.mergeRequests {
		if mr.ProjectID == projectID && mr.IID == iid {
			response := gitLabMRResponse{
				ID:             mr.ID,
				IID:            mr.IID,
				ProjectID:      mr.ProjectID,
				Title:          mr.Title,
				Description:    mr.Description,
				SourceBranch:   mr.SourceBranch,
				TargetBranch:   mr.TargetBranch,
				State:          mr.State,
				Draft:          mr.Draft,
				WebURL:         mr.WebURL,
				MergeCommitSHA: mr.MergeCommitSHA,
			}
			if err := json.NewEncoder(w).Encode(response); err != nil {
				s.t.Logf("Failed to encode merge request response: %v", err)
				w.WriteHeader(http.StatusInternalServerError)
			}
			return
		}
	}

	w.WriteHeader(http.StatusNotFound)
	json.NewEncoder(w).Encode(map[string]string{"message": "404 Merge Request Not Found"})
}

func (s *mockGitLabServer) listMergeRequests(w http.ResponseWriter, projectID int64, r *http.Request) {
	var mrs []gitLabMRResponse
	sourceBranch := r.URL.Query().Get("source_branch")
	targetBranch := r.URL.Query().Get("target_branch")
	state := r.URL.Query().Get("state")

	for _, mr := range s.mergeRequests {
		if mr.ProjectID != projectID {
			continue
		}
		if sourceBranch != "" && mr.SourceBranch != sourceBranch {
			continue
		}
		if targetBranch != "" && mr.TargetBranch != targetBranch {
			continue
		}
		if state != "" && mr.State != state {
			continue
		}

		mrs = append(mrs, gitLabMRResponse{
			ID:           mr.ID,
			IID:          mr.IID,
			ProjectID:    mr.ProjectID,
			Title:        mr.Title,
			Description:  mr.Description,
			SourceBranch: mr.SourceBranch,
			TargetBranch: mr.TargetBranch,
			State:        mr.State,
			Draft:        mr.Draft,
			WebURL:       mr.WebURL,
		})
	}

	if err := json.NewEncoder(w).Encode(mrs); err != nil {
		s.t.Logf("Failed to encode merge requests response: %v", err)
		w.WriteHeader(http.StatusInternalServerError)
	}
}

func (s *mockGitLabServer) createMergeRequest(w http.ResponseWriter, r *http.Request, projectID int64) {
	var req gitLabMRResponse
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	// Generate new MR
	newMR := mockMR{
		ID:           int64(len(s.mergeRequests) + 1000),
		IID:          int64(len(s.mergeRequests) + 1),
		ProjectID:    projectID,
		Title:        req.Title,
		Description:  req.Description,
		SourceBranch: req.SourceBranch,
		TargetBranch: req.TargetBranch,
		State:        "opened",
		Draft:        req.Draft,
		WebURL:       fmt.Sprintf("%s/test/repo/-/merge_requests/%d", strings.TrimSuffix(s.URL, gitLabAPIPrefix), len(s.mergeRequests)+1),
	}

	s.mergeRequests = append(s.mergeRequests, newMR)

	response := gitLabMRResponse{
		ID:           newMR.ID,
		IID:          newMR.IID,
		ProjectID:    newMR.ProjectID,
		Title:        newMR.Title,
		Description:  newMR.Description,
		SourceBranch: newMR.SourceBranch,
		TargetBranch: newMR.TargetBranch,
		State:        newMR.State,
		Draft:        newMR.Draft,
		WebURL:       newMR.WebURL,
	}

	w.WriteHeader(http.StatusCreated)
	if err := json.NewEncoder(w).Encode(response); err != nil {
		s.t.Logf("Failed to encode created merge request: %v", err)
		w.WriteHeader(http.StatusInternalServerError)
	}
}

func (s *mockGitLabServer) updateMergeRequest(w http.ResponseWriter, r *http.Request, projectID, iid int64) {
	var req gitLabMRResponse
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	for i, mr := range s.mergeRequests {
		if mr.ProjectID == projectID && mr.IID == iid {
			// Update fields
			if req.Title != "" {
				s.mergeRequests[i].Title = req.Title
			}
			if req.Description != "" {
				s.mergeRequests[i].Description = req.Description
			}
			if req.State != "" {
				s.mergeRequests[i].State = req.State
			}
			s.mergeRequests[i].Draft = req.Draft

			response := gitLabMRResponse{
				ID:           s.mergeRequests[i].ID,
				IID:          s.mergeRequests[i].IID,
				ProjectID:    s.mergeRequests[i].ProjectID,
				Title:        s.mergeRequests[i].Title,
				Description:  s.mergeRequests[i].Description,
				SourceBranch: s.mergeRequests[i].SourceBranch,
				TargetBranch: s.mergeRequests[i].TargetBranch,
				State:        s.mergeRequests[i].State,
				Draft:        s.mergeRequests[i].Draft,
				WebURL:       s.mergeRequests[i].WebURL,
			}

			if err := json.NewEncoder(w).Encode(response); err != nil {
				s.t.Logf("Failed to encode updated merge request: %v", err)
				w.WriteHeader(http.StatusInternalServerError)
			}
			return
		}
	}

	w.WriteHeader(http.StatusNotFound)
	json.NewEncoder(w).Encode(map[string]string{"message": "404 Merge Request Not Found"})
}

func (s *mockGitLabServer) getProject(w http.ResponseWriter, projectID int64) {
	// Return a default project for testing
	project := gitLabProjectResponse{
		ID:                projectID,
		Name:              "test-repo",
		Path:              "test-repo",
		PathWithNamespace: "test-namespace/test-repo",
		WebURL:            fmt.Sprintf("%s/test-namespace/test-repo", strings.TrimSuffix(s.URL, gitLabAPIPrefix)),
		HTTPURLToRepo:     fmt.Sprintf("%s/test-namespace/test-repo.git", strings.TrimSuffix(s.URL, gitLabAPIPrefix)),
		SSHURLToRepo:      fmt.Sprintf("git@%s:test-namespace/test-repo.git", strings.TrimPrefix(s.URL, "http://")),
		DefaultBranch:     "main",
		Namespace: mockNamespace{
			ID:       200,
			Name:     "Test Namespace",
			Path:     "test-namespace",
			Kind:     "group",
			FullPath: "test-namespace",
		},
	}

	if err := json.NewEncoder(w).Encode(project); err != nil {
		s.t.Logf("Failed to encode project response: %v", err)
		w.WriteHeader(http.StatusInternalServerError)
	}
}

// AddMergeRequest adds a merge request to the mock server for testing
func (s *mockGitLabServer) AddMergeRequest(mr mockMR) {
	s.mergeRequests = append(s.mergeRequests, mr)
}