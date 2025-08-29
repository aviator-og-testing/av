package main

import (
	"fmt"
	"os"

	"emperror.dev/errors"
	"github.com/aviator-co/av/internal/gh"
	"github.com/aviator-co/av/internal/meta"
	"github.com/aviator-co/av/internal/utils/cleanup"
	"github.com/aviator-co/av/internal/utils/colors"
	"github.com/aviator-co/av/internal/vcs"
	"github.com/fatih/color"
	"github.com/shurcooL/githubv4"
	"github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
)

var fetchCmd = &cobra.Command{
	Use:   "fetch",
	Short: "Fetch latest repository state from remote provider",
	Args:  cobra.NoArgs,
	RunE: func(cmd *cobra.Command, _ []string) (reterr error) {
		ctx := cmd.Context()

		repo, err := getRepo(ctx)
		if err != nil {
			return err
		}
		db, err := getDB(ctx, repo)
		if err != nil {
			return err
		}

		tx := db.WriteTx()
		var cu cleanup.Cleanup
		defer cu.Cleanup()
		cu.Add(func() {
			logrus.WithError(reterr).Debug("aborting db transaction")
			tx.Abort()
		})

		info := tx.Repository()
		branches := tx.AllBranches()

		origin, err := repo.Origin(ctx)
		if err != nil {
			return err
		}
		provider, err := vcs.DetectAndCreateProvider(ctx, origin.URL.String())
		if err != nil {
			// Enhance error message with provider-specific guidance
			if providerErr, ok := err.(vcs.ProviderError); ok {
				return errors.New(providerErr.Error())
			}
			return errors.Wrapf(err, "failed to initialize provider for repository at %s", origin.URL.String())
		}

		// Check if this is a GitHub repository for backward compatibility
		providerType := vcs.DetectProviderFromURL(origin.URL.String())
		if providerType == vcs.ProviderTypeGitHub {
			return fetchFromGitHub(ctx, tx, info, branches, provider)
		}

		// For GitLab and other providers, use generic provider interface
		return fetchFromGenericProvider(ctx, tx, info, branches, provider)
	},
}

func fetchFromGitHub(ctx context.Context, tx meta.WriteTx, info meta.Repository, branches map[string]meta.Branch, provider vcs.Provider) error {
	// Use existing GitHub-specific logic for backward compatibility
	client, err := getGitHubClient(ctx)
	if err != nil {
		return err
	}

	var cursor string
	updatedCount := 0
	for {
			prsPage, err := client.RepoPullRequests(ctx, gh.RepoPullRequestOpts{
				Owner:  info.Owner,
				Repo:   info.Name,
				After:  cursor,
				States: []githubv4.PullRequestState{githubv4.PullRequestStateOpen},
			})
			if err != nil {
				return errors.Wrap(err, "failed to fetch pull requests from GitHub")
			}
			if cursor == "" {
				// only do this once at the start
				fmt.Fprint(
					os.Stderr,
					"Fetching ", colors.UserInput(prsPage.TotalCount),
					" open pull requests from GitHub...",
					"\n",
				)
			}

			for _, pr := range prsPage.PullRequests {
				// TODO: maybe warn if local branch is not up-to-date with remote?
				branchMeta, ok := branches[pr.HeadBranchName()]
				if !ok {
					logrus.WithField("branch", pr.HeadBranchName()).
						Debug("skipping PR for unknown local branch")
					continue
				}
				logrus.WithField("branch", pr.HeadBranchName()).
					Debug("found PR for known local branch")
				if branchMeta.PullRequest == nil {
					fmt.Fprint(
						os.Stderr,
						"  - Found pull request ", colors.UserInput(pr.Number),
						" for branch ", colors.UserInput(pr.HeadBranchName()),
						"\n",
					)
				} else if branchMeta.PullRequest.Number != pr.Number {
					// This shouldn't usually ever happen, not sure what the
					// best thing to do here, but this handling allows you to
					// close a PR then open a new one and then run `av fetch`
					fmt.Fprint(
						os.Stderr,
						"  - ", color.RedString("WARNING: "),
						"found new pull request ", colors.UserInput("#", pr.Number, " ", pr.Title),
						" for branch ", colors.UserInput(pr.HeadBranchName()),
						", overwriting... ",
						" (old pull request: ", colors.UserInput("#", branchMeta.PullRequest.Number), ")",
						"\n",
					)
				} else {
					// nothing to do, we already have the PR stored in metadata
					continue
				}
				updatedCount++
				branchMeta.PullRequest = &meta.PullRequest{
					ID:        pr.ID,
					Number:    pr.Number,
					Permalink: pr.Permalink,
				}
				tx.SetBranch(branchMeta)
			}

			if prsPage.HasNextPage {
				cursor = prsPage.EndCursor
			} else {
				break
			}
		}

		cu.Cancel()
		if err := tx.Commit(); err != nil {
			return err
		}
		fmt.Fprint(
			os.Stderr,
			"Updated ", color.GreenString("%d", updatedCount), " pull requests",
			"\n",
		)
		return nil
	},
}

func fetchFromGenericProvider(ctx context.Context, tx meta.WriteTx, info meta.Repository, branches map[string]meta.Branch, provider vcs.Provider) error {
	// Fetch merge requests/pull requests from GitLab or other providers
	fmt.Fprint(os.Stderr, "Fetching merge requests from provider...\n")
	
	updatedCount := 0
	cursor := ""
	
	for {
		// Get pull requests/merge requests from the provider
		prsPage, err := provider.GetPRs(ctx, vcs.GetPullRequestsInput{
			Owner: info.Owner,
			Repo:  info.Name,
			After: cursor,
			First: 100,
			States: []string{"open"},
		})
		if err != nil {
			// Provide GitLab-specific error guidance
			if errors.Is(err, context.DeadlineExceeded) {
				return errors.Wrap(err, "timeout while fetching merge requests from provider - the provider API may be slow or unavailable")
			}
			return errors.Wrap(err, "failed to fetch merge requests from provider. Please check your authentication token and network connectivity")
		}

		if cursor == "" && len(prsPage.PullRequests) > 0 {
			// only do this once at the start
			fmt.Fprintf(os.Stderr, "Fetching %d open merge requests from provider...\n", len(prsPage.PullRequests))
		}

		for _, pr := range prsPage.PullRequests {
			branchMeta, ok := branches[pr.HeadBranchName()]
			if !ok {
				logrus.WithField("branch", pr.HeadBranchName()).
					Debug("skipping MR for unknown local branch")
				continue
			}
			logrus.WithField("branch", pr.HeadBranchName()).
				Debug("found MR for known local branch")
			
			if branchMeta.PullRequest == nil {
				fmt.Fprintf(os.Stderr, "  - Found merge request %d for branch %s\n", 
					pr.GetNumber(), pr.HeadBranchName())
			} else if branchMeta.PullRequest.Number != pr.GetNumber() {
				fmt.Fprintf(os.Stderr, "  - WARNING: found new merge request #%d %s for branch %s, overwriting... (old merge request: #%d)\n",
					pr.GetNumber(), pr.GetTitle(), pr.HeadBranchName(), branchMeta.PullRequest.Number)
			} else {
				// nothing to do, we already have the MR stored in metadata
				continue
			}
			
			updatedCount++
			branchMeta.PullRequest = &meta.PullRequest{
				ID:        pr.GetID(),
				Number:    pr.GetNumber(),
				Permalink: pr.GetPermalink(),
			}
			tx.SetBranch(branchMeta)
		}

		if prsPage.HasNextPage {
			cursor = prsPage.EndCursor
		} else {
			break
		}
	}

	fmt.Fprintf(os.Stderr, "Updated %s merge requests\n", 
		color.GreenString("%d", updatedCount))
	return nil
}
