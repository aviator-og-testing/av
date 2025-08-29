package actions

import (
	"context"
	"fmt"
	"os"
	"strings"

	"emperror.dev/errors"
	"github.com/aviator-co/av/internal/utils/colors"
	"github.com/aviator-co/av/internal/vcs"
)

// AddPullRequestReviewers adds the given reviewers to the given pull request.
// It accepts a list of reviewers, which can be either user logins or
// team names in the format `@organization/team`.
// Works with both GitHub and GitLab providers through the VCS abstraction.
func AddPullRequestReviewers(
	ctx context.Context,
	provider vcs.Provider,
	prID string,
	reviewers []string,
) error {
	_, _ = fmt.Fprint(os.Stderr,
		"  - adding ", colors.UserInput(len(reviewers)), " reviewer(s) to pull request\n",
	)

	// We need to map the given reviewers to provider-specific user and team IDs.
	var reviewerIDs []string
	var teamIDs []string
	for _, reviewer := range reviewers {
		if ok, org, team := isTeamName(reviewer); ok {
			// Handle team/group assignment
			providerTeam, err := provider.GetOrganizationTeam(ctx, org, team)
			if err != nil {
				return errors.WrapIff(err, "failed to get team %s/%s", org, team)
			}
			teamIDs = append(teamIDs, providerTeam.GetID())
		} else {
			// Handle individual user assignment
			user, err := provider.GetUser(ctx, reviewer)
			if err != nil {
				return errors.WrapIff(err, "failed to get user %s", reviewer)
			}
			reviewerIDs = append(reviewerIDs, user.GetID())
		}
	}

	// Request reviews using the provider abstraction
	if _, err := provider.RequestReviews(ctx, vcs.RequestReviewsInput{
		PullRequestID: prID,
		UserIDs:       reviewerIDs,
		TeamIDs:       teamIDs,
		Union:         true,
	}); err != nil {
		return errors.WrapIf(err, "requesting reviews")
	}

	return nil
}

func isTeamName(s string) (bool, string, string) {
	before, after, found := strings.Cut(s, "/")
	if !found || before == "" || after == "" {
		return false, "", ""
	}

	// It's common to specify team names as `@aviator-co/engineering`. We want
	// just the organization name (`aviator-co`) and team slug (`engineering`)
	// here, so strip the leading `@` if it exists.
	// This shouldn't cause any ambiguity since GitHub user login's can't
	// contain a slash character.
	before = strings.TrimPrefix(before, "@")
	return true, before, after
}
