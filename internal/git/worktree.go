package git

import (
	"context"
	"fmt"
	"regexp"
	"strings"

	"emperror.dev/errors"
)

// WorktreeInfo contains information about a Git worktree.
type WorktreeInfo struct {
	// Path is the absolute path to the worktree directory
	Path string
	// Head is the commit hash that HEAD points to
	Head string
	// Branch is the checked out branch (empty if detached)
	Branch string
	// IsBare indicates if this is a bare repository
	IsBare bool
	// IsDetached indicates if HEAD is detached
	IsDetached bool
	// Locked indicates if the worktree is locked
	Locked bool
	// LockReason is the reason the worktree is locked (if Locked is true)
	LockReason string
	// Prunable indicates if the worktree is prunable
	Prunable bool
	// PrunableReason is the reason the worktree is prunable (if Prunable is true)
	PrunableReason string
}

// ListWorktrees returns a list of all worktrees in the repository.
// It parses the output of `git worktree list --porcelain`.
func (r *Repo) ListWorktrees(ctx context.Context) ([]WorktreeInfo, error) {
	out, err := r.Run(ctx, &RunOpts{
		Args:      []string{"worktree", "list", "--porcelain"},
		ExitError: true,
	})
	if err != nil {
		return nil, errors.Wrap(err, "failed to list worktrees")
	}

	var worktrees []WorktreeInfo
	var current *WorktreeInfo

	for _, line := range out.Lines() {
		// Empty line separates worktree entries
		if line == "" {
			if current != nil {
				worktrees = append(worktrees, *current)
				current = nil
			}
			continue
		}

		if current == nil {
			current = &WorktreeInfo{}
		}

		// Parse porcelain format
		// Format: <key> <value>
		// Keys: worktree, HEAD, branch, bare, detached, locked, prunable
		parts := strings.SplitN(line, " ", 2)
		if len(parts) < 1 {
			continue
		}

		key := parts[0]
		value := ""
		if len(parts) == 2 {
			value = parts[1]
		}

		switch key {
		case "worktree":
			current.Path = value
		case "HEAD":
			current.Head = value
		case "branch":
			// Branch is in the format "refs/heads/branch-name"
			current.Branch = strings.TrimPrefix(value, "refs/heads/")
		case "bare":
			current.IsBare = true
		case "detached":
			current.IsDetached = true
		case "locked":
			current.Locked = true
			current.LockReason = value
		case "prunable":
			current.Prunable = true
			current.PrunableReason = value
		}
	}

	// Don't forget the last entry if there's no trailing newline
	if current != nil {
		worktrees = append(worktrees, *current)
	}

	return worktrees, nil
}

// GetWorktreeForBranch finds the worktree that has the given branch checked out.
// Returns nil if the branch is not checked out in any worktree.
func (r *Repo) GetWorktreeForBranch(ctx context.Context, branchName string) (*WorktreeInfo, error) {
	worktrees, err := r.ListWorktrees(ctx)
	if err != nil {
		return nil, err
	}

	for _, wt := range worktrees {
		if wt.Branch == branchName {
			return &wt, nil
		}
	}

	return nil, nil
}

// IsWorktreeLockError checks if an error is a worktree lock error.
// Git returns an error like "fatal: 'branch-name' is already checked out at '/path/to/worktree'"
// when trying to check out a branch that is already checked out in another worktree.
func IsWorktreeLockError(err error) bool {
	if err == nil {
		return false
	}

	// Pattern: fatal: 'branch' is already checked out at 'path'
	// Also match: error: cannot lock ref 'refs/heads/branch': is at <hash> but expected <hash>
	pattern := `(?i)(fatal|error):.*(already checked out|cannot lock ref)`
	matched, _ := regexp.MatchString(pattern, err.Error())
	return matched
}

// ParseWorktreeLockError attempts to extract the branch name and worktree path
// from a worktree lock error message.
// Returns empty strings if the error is not a worktree lock error or cannot be parsed.
func ParseWorktreeLockError(err error) (branch string, worktreePath string) {
	if err == nil {
		return "", ""
	}

	errMsg := err.Error()

	// Pattern: fatal: 'branch-name' is already checked out at '/path/to/worktree'
	re := regexp.MustCompile(`'([^']+)'\s+is already checked out at\s+'([^']+)'`)
	matches := re.FindStringSubmatch(errMsg)
	if len(matches) >= 3 {
		return matches[1], matches[2]
	}

	// Alternative pattern without quotes
	re = regexp.MustCompile(`(\S+)\s+is already checked out at\s+(\S+)`)
	matches = re.FindStringSubmatch(errMsg)
	if len(matches) >= 3 {
		return matches[1], matches[2]
	}

	return "", ""
}

// ErrWorktreeLock represents an error when a branch is locked by another worktree.
type ErrWorktreeLock struct {
	Branch       string
	WorktreePath string
	OriginalErr  error
}

func (e ErrWorktreeLock) Error() string {
	return fmt.Sprintf(
		"cannot checkout '%s': already checked out at '%s'",
		e.Branch,
		e.WorktreePath,
	)
}

func (e ErrWorktreeLock) Unwrap() error {
	return e.OriginalErr
}

// WrapWorktreeLockError wraps an error with worktree lock information if it's a worktree lock error.
// Returns the original error if it's not a worktree lock error.
func WrapWorktreeLockError(err error) error {
	if err == nil || !IsWorktreeLockError(err) {
		return err
	}

	branch, worktreePath := ParseWorktreeLockError(err)
	if branch == "" && worktreePath == "" {
		return err
	}

	return &ErrWorktreeLock{
		Branch:       branch,
		WorktreePath: worktreePath,
		OriginalErr:  err,
	}
}
