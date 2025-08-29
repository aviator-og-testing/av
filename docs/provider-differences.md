# Provider Differences: GitHub vs GitLab

This document outlines the key differences when using `av` with GitHub versus GitLab repositories.

## Terminology Differences

| Feature | GitHub | GitLab |
|---------|--------|--------|
| Code review mechanism | Pull Request (PR) | Merge Request (MR) |
| User groups | Teams | Groups |
| Code hosting | Repositories | Projects |

Note: Despite these terminology differences, `av` commands remain consistent (e.g., `av pr` works for both GitHub PRs and GitLab MRs).

## Authentication

### GitHub
- Supports GitHub CLI token auto-discovery (`gh auth token`)
- Personal Access Token via environment variables
- Token scopes: `repo`, `workflow` (if using GitHub Actions)

### GitLab
- Personal Access Token only (no CLI auto-discovery)
- Self-hosted instance support via `AV_GITLAB_BASE_URL`
- Token scopes: `api`, `write_repository`

## Configuration

### GitHub
```bash
# Auto-discovered from GitHub CLI
gh auth login

# Or manual configuration
export AV_GITHUB_TOKEN="ghp_xxxxxxxxxxxxxxxxxxxx"
```

### GitLab
```bash
# GitLab.com
export AV_GITLAB_TOKEN="glpat-xxxxxxxxxxxxxxxxxxxx"

# Self-hosted GitLab
export AV_GITLAB_TOKEN="glpat-xxxxxxxxxxxxxxxxxxxx"
export AV_GITLAB_BASE_URL="https://gitlab.example.com"
```

## Feature Parity

### Supported Features (Both Providers)
- ✅ Create pull/merge requests
- ✅ Update pull/merge requests
- ✅ Assign reviewers
- ✅ Draft mode support
- ✅ Stack visualization
- ✅ Branch synchronization
- ✅ Repository detection
- ✅ User authentication verification

### GitHub-Specific Features
- 🔵 GitHub CLI integration for token management
- 🔵 GitHub Actions workflow triggers (if applicable)
- 🔵 GitHub Teams for reviewer assignment

### GitLab-Specific Features
- 🟠 Self-hosted GitLab instance support
- 🟠 GitLab Groups for reviewer assignment
- 🟠 GitLab-specific merge request settings

## Reviewer Assignment

### GitHub
```bash
# Individual users
av pr --reviewers "username1,username2"

# Teams (requires @ prefix)
av pr --reviewers "@org/team-name"

# Mixed
av pr --reviewers "username1,@org/team-name"
```

### GitLab
```bash
# Individual users
av pr --reviewers "username1,username2"

# Groups (requires @ prefix)
av pr --reviewers "@group-name"

# Mixed
av pr --reviewers "username1,@group-name"
```

## URL Patterns

### GitHub
- Repository: `https://github.com/owner/repo`
- Pull Request: `https://github.com/owner/repo/pull/123`
- SSH: `git@github.com:owner/repo.git`

### GitLab
- Repository: `https://gitlab.com/namespace/project`
- Merge Request: `https://gitlab.com/namespace/project/-/merge_requests/123`
- SSH: `git@gitlab.com:namespace/project.git`
- Self-hosted: `https://gitlab.example.com/namespace/project`

## API Differences

### GitHub
- Uses GitHub GraphQL API v4
- Rate limiting: 5000 requests/hour (authenticated)
- Supports fine-grained personal access tokens

### GitLab
- Uses GitLab REST API v4
- Rate limiting: 2000 requests/hour (GitLab.com)
- Self-hosted instances may have different rate limits

## Limitations

### Current Limitations (Both Providers)
- No direct merge/close operations from CLI (use web interface or provider CLI)
- Limited to basic reviewer assignment (no approval requirements)

### GitHub-Specific Limitations
- None currently identified

### GitLab-Specific Limitations
- Self-hosted instances may have API differences depending on version
- Some advanced GitLab features (like merge request dependencies) not yet supported

## Migration Between Providers

When moving a repository from GitHub to GitLab (or vice versa):

1. **Repository Migration**
   - Update Git remotes: `git remote set-url origin <new-url>`
   - Re-run `av init` to reinitialize metadata

2. **Authentication**
   - Set up new provider tokens
   - Remove old provider tokens if desired

3. **Existing Stacks**
   - Existing branch metadata is preserved
   - New PRs/MRs will be created on the new provider
   - Old PRs/MRs remain on the previous provider

## Best Practices

### For GitHub Users
- Use GitHub CLI for token management: `gh auth login`
- Leverage GitHub Teams for better reviewer management
- Consider using fine-grained tokens for enhanced security

### For GitLab Users
- Store tokens securely (consider using secret management tools)
- Use GitLab Groups for efficient reviewer assignment
- For self-hosted instances, ensure API compatibility (GitLab 13.0+)

### For Multi-Provider Users
- Set up both GitHub and GitLab authentication
- Repository detection is automatic based on remote URL
- Keep tokens in separate environment variables for clarity

## Troubleshooting

### Common Issues
- **Wrong provider detected**: Check your Git remote URL with `git remote -v`
- **Authentication failures**: Verify token scopes and expiration
- **API errors**: Check network connectivity and provider status pages

### Provider-Specific Issues
- **GitHub**: Rate limiting with large repositories
- **GitLab**: Self-hosted instance version compatibility