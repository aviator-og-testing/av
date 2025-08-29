# GitLab Setup Guide

This guide walks through setting up `av` with GitLab repositories, including GitLab.com and self-hosted GitLab instances.

## Prerequisites

- GitLab account (GitLab.com or self-hosted instance)
- Git repository hosted on GitLab
- `av` CLI installed

## Authentication Setup

### 1. Create a Personal Access Token

1. Go to your GitLab instance (e.g., https://gitlab.com)
2. Navigate to **Settings → Access Tokens** (under your profile menu)
3. Create a new token with the following scopes:
   - `api` - Full access to the GitLab API
   - `write_repository` - Write access to repositories

### 2. Configure the Token

Set your GitLab token using environment variables:

```bash
# For GitLab.com
export AV_GITLAB_TOKEN="your-gitlab-token"

# For self-hosted GitLab instances
export AV_GITLAB_TOKEN="your-gitlab-token"
export AV_GITLAB_BASE_URL="https://gitlab.example.com"
```

You can also set these in your shell profile (`.bashrc`, `.zshrc`, etc.) to persist across sessions.

### 3. Verify Authentication

Check that your GitLab authentication is working:

```bash
av auth
```

You should see output confirming your GitLab authentication status.

## Repository Setup

### 1. Initialize Repository

In your GitLab repository, initialize `av`:

```bash
av init
```

### 2. Verify Configuration

Check that `av` can detect your GitLab repository:

```bash
av tree
```

## Usage Examples

### Creating Merge Requests

Create a merge request (GitLab's term for pull requests):

```bash
# Create a simple MR
av pr --title "My Feature"

# Create with reviewers
av pr --title "My Feature" --reviewers "username1,@group-name"

# Create as draft
av pr --title "My Feature" --draft
```

### Working with Stacks

All standard `av` commands work with GitLab repositories:

```bash
# Create stacked branches
av branch feature-1
# ... make changes ...
av commit -m "Implement feature 1"
av pr

av branch feature-2
# ... make changes ...
av commit -m "Implement feature 2"
av pr

# View the stack
av tree

# Sync with GitLab
av sync
```

## Troubleshooting

### Authentication Issues

**Error: "No GitLab token is set"**
- Ensure `AV_GITLAB_TOKEN` environment variable is set
- Verify the token has correct scopes (`api` and `write_repository`)

**Error: "You are not logged in to GitLab"**
- Check that your token is valid and hasn't expired
- For self-hosted instances, ensure `AV_GITLAB_BASE_URL` is set correctly

### Repository Detection Issues

**Error: Repository not detected as GitLab**
- Ensure your Git remote URL points to a GitLab instance
- Check with: `git remote -v`

### Self-Hosted GitLab Issues

**Error: Connection failures**
- Verify `AV_GITLAB_BASE_URL` is set to your GitLab instance URL
- Ensure the URL is accessible from your machine
- Check that the GitLab instance supports the API version used by `av`

## Feature Differences

When using GitLab repositories, note these differences from GitHub:

- **Merge Requests vs Pull Requests**: GitLab uses "merge requests" terminology, but `av` commands remain the same
- **Groups vs Teams**: Use GitLab group names with `@` prefix for reviewer assignment
- **Self-hosted Support**: GitLab supports self-hosted instances via `AV_GITLAB_BASE_URL`

## Configuration Reference

Environment variables for GitLab setup:

| Variable | Description | Example |
|----------|-------------|---------|
| `AV_GITLAB_TOKEN` | GitLab Personal Access Token | `glpat-xxxxxxxxxxxxxxxxxxxx` |
| `AV_GITLAB_BASE_URL` | GitLab instance URL (for self-hosted) | `https://gitlab.example.com` |
| `GITLAB_TOKEN` | Alternative token variable | `glpat-xxxxxxxxxxxxxxxxxxxx` |

The token discovery order is:
1. `AV_GITLAB_TOKEN`
2. `GITLAB_TOKEN`