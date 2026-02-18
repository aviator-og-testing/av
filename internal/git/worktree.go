package git

import (
	"context"
	"strings"

	"emperror.dev/errors"
)

// WorktreeInfo contains information about a Git worktree.
type WorktreeInfo struct {
	// Path is the absolute path to the worktree directory.
	Path string
	// Head is the commit hash that the worktree currently points to.
	Head string
	// Branch is the name of the checked out branch (short form, without refs/heads/).
	// Empty if the worktree is in detached HEAD state.
	Branch string
	// IsBare indicates if this is a bare repository.
	IsBare bool
	// IsDetached indicates if the worktree is in detached HEAD state.
	IsDetached bool
	// Locked indicates if the worktree is locked.
	Locked bool
	// LockReason is the reason the worktree is locked (if Locked is true).
	LockReason string
	// Prunable indicates if the worktree can be pruned.
	Prunable bool
	// PrunableReason is the reason the worktree is prunable (if Prunable is true).
	PrunableReason string
}

// ListWorktrees returns a list of all worktrees in the repository.
// It parses the output of `git worktree list --porcelain`.
func (r *Repo) ListWorktrees(ctx context.Context) ([]WorktreeInfo, error) {
	output, err := r.Run(ctx, &RunOpts{
		Args:      []string{"worktree", "list", "--porcelain"},
		ExitError: true,
	})
	if err != nil {
		return nil, errors.Wrap(err, "failed to list worktrees")
	}

	return ParseWorktreeListPorcelain(string(output.Stdout))
}

// ParseWorktreeListPorcelain parses the porcelain output of `git worktree list`.
// The porcelain format is:
//
//	worktree <path>
//	HEAD <commit-hash>
//	branch refs/heads/<branch-name>  (or detached if detached HEAD)
//	bare                              (optional, if bare)
//	detached                          (optional, if detached HEAD)
//	locked <reason>                   (optional, if locked)
//	prunable <reason>                 (optional, if prunable)
//
// Worktrees are separated by blank lines.
func ParseWorktreeListPorcelain(output string) ([]WorktreeInfo, error) {
	var worktrees []WorktreeInfo
	lines := strings.Split(strings.TrimSpace(output), "\n")

	var current *WorktreeInfo
	for _, line := range lines {
		line = strings.TrimSpace(line)

		// Blank line indicates end of a worktree entry
		if line == "" {
			if current != nil {
				worktrees = append(worktrees, *current)
				current = nil
			}
			continue
		}

		// Start a new worktree entry
		if strings.HasPrefix(line, "worktree ") {
			if current != nil {
				worktrees = append(worktrees, *current)
			}
			current = &WorktreeInfo{
				Path: strings.TrimPrefix(line, "worktree "),
			}
			continue
		}

		if current == nil {
			return nil, errors.Errorf("unexpected line before worktree declaration: %q", line)
		}

		// Parse worktree properties
		switch {
		case strings.HasPrefix(line, "HEAD "):
			current.Head = strings.TrimPrefix(line, "HEAD ")
		case strings.HasPrefix(line, "branch "):
			branchRef := strings.TrimPrefix(line, "branch ")
			// Convert refs/heads/branch-name to branch-name
			current.Branch = strings.TrimPrefix(branchRef, "refs/heads/")
		case line == "bare":
			current.IsBare = true
		case line == "detached":
			current.IsDetached = true
		case strings.HasPrefix(line, "locked"):
			current.Locked = true
			// locked can be "locked" or "locked <reason>"
			if len(line) > 7 {
				current.LockReason = strings.TrimSpace(line[7:])
			}
		case strings.HasPrefix(line, "prunable"):
			current.Prunable = true
			// prunable can be "prunable" or "prunable <reason>"
			if len(line) > 9 {
				current.PrunableReason = strings.TrimSpace(line[9:])
			}
		default:
			// Ignore unknown lines for forward compatibility
			// (don't error to maintain forward compatibility with future Git versions)
		}
	}

	// Don't forget the last worktree if there's no trailing blank line
	if current != nil {
		worktrees = append(worktrees, *current)
	}

	return worktrees, nil
}

// GetWorktreeForBranch returns the worktree information for the given branch,
// or nil if the branch is not checked out in any worktree.
// The branchName can be in short form (e.g., "feature") or full form (e.g., "refs/heads/feature").
func (r *Repo) GetWorktreeForBranch(ctx context.Context, branchName string) (*WorktreeInfo, error) {
	worktrees, err := r.ListWorktrees(ctx)
	if err != nil {
		return nil, err
	}

	// Normalize branch name to short form
	normalizedBranch := strings.TrimPrefix(branchName, "refs/heads/")

	for i := range worktrees {
		if worktrees[i].Branch == normalizedBranch {
			return &worktrees[i], nil
		}
	}

	return nil, nil
}

// IsWorktreeLockError checks if the error is a Git error indicating that
// a branch is already checked out in another worktree.
// Git returns errors like: "fatal: 'branch-name' is already checked out at '/path/to/worktree'"
func IsWorktreeLockError(err error) bool {
	if err == nil {
		return false
	}
	errStr := err.Error()
	return strings.Contains(errStr, "is already checked out at") ||
		strings.Contains(errStr, "used by worktree")
}

// ParseWorktreeLockError extracts the branch name and worktree path from a
// worktree lock error message.
// Returns empty strings if the error is not a worktree lock error or cannot be parsed.
func ParseWorktreeLockError(err error) (branch, path string) {
	if err == nil || !IsWorktreeLockError(err) {
		return "", ""
	}

	errStr := err.Error()

	// Pattern 1: "fatal: 'branch-name' is already checked out at '/path/to/worktree'"
	if strings.Contains(errStr, "is already checked out at") {
		// Find the branch name (between quotes or apostrophes)
		if idx := strings.Index(errStr, "'"); idx != -1 {
			rest := errStr[idx+1:]
			if endIdx := strings.Index(rest, "'"); endIdx != -1 {
				branch = rest[:endIdx]
			}
		}

		// Find the path (after "at ")
		if idx := strings.Index(errStr, "at '"); idx != -1 {
			rest := errStr[idx+4:]
			if endIdx := strings.Index(rest, "'"); endIdx != -1 {
				path = rest[:endIdx]
			}
		}
	}

	// Pattern 2: "error: Cannot delete branch 'branch-name' checked out at '/path/to/worktree'"
	// or similar messages with "used by worktree"
	// This is a fallback for other error formats
	if branch == "" && strings.Contains(errStr, "used by worktree") {
		// Try to extract from generic patterns
		if idx := strings.Index(errStr, "'"); idx != -1 {
			rest := errStr[idx+1:]
			if endIdx := strings.Index(rest, "'"); endIdx != -1 {
				branch = rest[:endIdx]
			}
		}
	}

	return branch, path
}

// IsBranchCheckedOut checks if a branch is currently checked out in any worktree.
// Returns (true, worktreePath, nil) if the branch is checked out,
// (false, "", nil) if it's not checked out, or (false, "", error) on error.
// The branchName can be in short form (e.g., "feature") or full form (e.g., "refs/heads/feature").
func (r *Repo) IsBranchCheckedOut(ctx context.Context, branchName string) (bool, string, error) {
	worktree, err := r.GetWorktreeForBranch(ctx, branchName)
	if err != nil {
		return false, "", err
	}
	if worktree == nil {
		return false, "", nil
	}
	return true, worktree.Path, nil
}
