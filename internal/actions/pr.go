package actions

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"text/template"

	"github.com/aviator-co/av/internal/utils/errutils"

	"emperror.dev/errors"
	"github.com/aviator-co/av/internal/config"
	"github.com/aviator-co/av/internal/editor"
	"github.com/aviator-co/av/internal/gh"
	"github.com/aviator-co/av/internal/git"
	"github.com/aviator-co/av/internal/meta"
	"github.com/aviator-co/av/internal/utils/browser"
	"github.com/aviator-co/av/internal/utils/colors"
	"github.com/aviator-co/av/internal/utils/sanitize"
	"github.com/aviator-co/av/internal/utils/stackutils"
	"github.com/aviator-co/av/internal/utils/stringutils"
	"github.com/aviator-co/av/internal/utils/templateutils"
	"github.com/aviator-co/av/internal/vcs"
	"github.com/fatih/color"
	"github.com/shurcooL/githubv4"
	"github.com/sirupsen/logrus"
)

type CreatePullRequestOpts struct {
	// The HEAD branch to create a pull request for.
	BranchName string
	// The pull request title.
	Title string
	// The pull request body (description).
	Body string
	// If true, create the pull request as a draft PR.
	Draft bool
	// If true, do not push the branch to remote
	NoPush bool
	// If true, force push the branch to remote
	ForcePush bool
	// If true, create a PR even if we think one already exists
	Force bool
	// If true, open an editor for editing the title and body
	Edit bool
	// If true, do not open the browser after creating the PR
	NoOpenBrowser bool
}

type CreatePullRequestResult struct {
	// True if the pull request was created
	Created bool
	// The (updated) branch metadata.
	Branch meta.Branch
	// The pull request object that was returned from the provider
	Pull vcs.PullRequest
}

// getPRMetadata constructs the PRMetadata for the current state of the branch.
// TODO:
// The way we pass/load all the relevant data here is not great(tm). The
// `parent` argument is optional because sometimes it's loaded by the
// calling function and sometimes not. :shrug: It can also be nil if the
// branch doesn't have a parent (i.e., the branch is a stack root).
func getPRMetadata(
	tx meta.ReadTx,
	branch meta.Branch,
	parent *meta.Branch,
) (PRMetadata, error) {
	trunk, _ := meta.Trunk(tx, branch.Name)
	prMeta := PRMetadata{
		Parent:     branch.Parent.Name,
		ParentHead: branch.Parent.Head,
		Trunk:      trunk,
	}
	if parent == nil && branch.Parent.Name != "" {
		p, _ := tx.Branch(branch.Parent.Name)
		parent = &p
	}
	if parent != nil && parent.PullRequest != nil {
		prMeta.ParentPull = parent.PullRequest.Number
	}
	return prMeta, nil
}

type errPullRequestClosed struct {
	pr vcs.PullRequest
}

func (e errPullRequestClosed) Error() string {
	state := "closed"
	if e.pr.IsMerged() {
		state = "merged"
	}
	return fmt.Sprintf("pull request #%d is %s", e.pr.GetNumber(), state)
}

// getExistingOpenPR returns an existing pull request for the given branch if
// any exist and are open.
func getExistingOpenPR(
	ctx context.Context,
	provider vcs.Provider,
	repoMeta meta.Repository,
	branchMeta meta.Branch,
	baseRefName string,
) (vcs.PullRequest, error) {
	if branchMeta.PullRequest != nil {
		logrus.WithField("pr", branchMeta.PullRequest.Number).
			Debug("querying data for existing PR from provider")
		pr, err := provider.GetPR(ctx, branchMeta.PullRequest.ID)
		if err != nil {
			return nil, errors.WrapIf(err, "querying existing pull request")
		}
		if !pr.IsOpen() {
			return nil, errPullRequestClosed{pr: pr}
		}
		return pr, nil
	}
	logrus.WithField("branch", branchMeta.Name).Debug("querying existing open PRs from provider")
	existing, err := provider.GetPRs(ctx, vcs.GetPullRequestsInput{
		Owner:       repoMeta.Owner,
		Repo:        repoMeta.Name,
		HeadRefName: branchMeta.Name,
		BaseRefName: baseRefName,
		States:      []string{"open"},
		First:       100,
	})
	if err != nil {
		return nil, errors.WrapIf(err, "querying existing pull requests")
	}
	if len(existing.PullRequests) > 1 {
		return nil, errors.Errorf("multiple existing PRs found for %q", branchMeta.Name)
	} else if len(existing.PullRequests) == 1 {
		return existing.PullRequests[0], nil
	}
	return nil, nil
}

// CreatePullRequest creates a pull request for the current branch, if
// one doesn't already exist.
func CreatePullRequest(
	ctx context.Context,
	repo *git.Repo,
	tx meta.WriteTx,
	opts CreatePullRequestOpts,
) (_ *CreatePullRequestResult, reterr error) {
	if opts.BranchName == "" {
		logrus.Panicf("internal invariant error: CreatePullRequest called with empty branch name")
	}

	repoMeta := tx.Repository()
	branchMeta, _ := tx.Branch(opts.BranchName)

	origin, err := repo.Origin(ctx)
	if err != nil {
		return nil, errors.WrapIf(err, "failed to determine repository origin")
	}
	
	provider, err := vcs.DetectAndCreateProvider(ctx, origin.URL.String())
	if err != nil {
		return nil, errors.WrapIf(err, "failed to create provider")
	}

	var existingPR vcs.PullRequest
	if !opts.Force {
		var err error
		existingPR, err = getExistingOpenPR(ctx, provider, repoMeta, branchMeta, opts.BranchName)
		if closed, ok := errutils.As[errPullRequestClosed](err); ok {
			state := "closed"
			if closed.pr.IsMerged() {
				state = "merged"
			}
			_, _ = fmt.Fprint(os.Stderr,
				colors.Failure("Existing pull request for branch "),
				colors.UserInput(opts.BranchName),
				colors.Failure(" is "), colors.UserInput(state),
				colors.Failure(": "), colors.UserInput(closed.pr.GetPermalink()),
				"\n",
			)
			_, _ = fmt.Fprint(os.Stderr,
				colors.Faint("  - use "), colors.CliCmd("av pr --force"),
				colors.Faint(" to create a new pull request for this branch\n"),
			)
			return nil, err
		} else if err != nil {
			return nil, errors.WrapIf(err, "failed to get existing pull request")
		}
	} else {
		// If we're forcing creation of a new PR, we always want to force push.
		// This prevents errors where we try to use `--force-with-lease` but the
		// remote branch has been deleted.
		opts.ForcePush = true
	}

	verb := "Creating"
	if existingPR != nil {
		verb = "Updating"
	}
	_, _ = fmt.Fprint(os.Stderr,
		verb, " pull request for branch ", colors.UserInput(opts.BranchName), ":",
		"\n",
	)
	if !opts.NoPush || opts.ForcePush {
		pushFlags := []string{"push"}

		if opts.ForcePush {
			pushFlags = append(pushFlags, "--force")
		} else {
			// Use --force-with-lease to allow pushing branches that have been
			// rebased but don't overwrite changes if we don't expect them to
			// be there.
			pushFlags = append(pushFlags, "--force-with-lease")
		}

		remote := repo.GetRemoteName()
		pushFlags = append(pushFlags, remote, opts.BranchName)
		logrus.Debug("pushing latest changes")

		_, _ = fmt.Fprint(os.Stderr,
			"  - pushing to ", color.CyanString("%s/%s", remote, opts.BranchName),
			"\n",
		)
		if _, err := repo.Git(ctx, pushFlags...); err != nil {
			return nil, errors.WrapIf(err, "failed to push")
		}
		if err := repo.BranchSetConfig(ctx, opts.BranchName, "av-pushed-remote", remote); err != nil {
			return nil, err
		}
		if err := repo.BranchSetConfig(ctx, opts.BranchName, "av-pushed-ref", fmt.Sprintf("refs/heads/%s", opts.BranchName)); err != nil {
			return nil, err
		}
	} else {
		_, _ = fmt.Fprint(os.Stderr,
			"  - skipping push to GitHub",
			"\n",
		)
	}

	// figure this out based on whether or not we're on a stacked branch
	parentState := branchMeta.Parent
	if parentState.Name == "" {
		defaultBranch, err := repo.DefaultBranch(ctx)
		if err != nil {
			return nil, errors.WrapIf(err, "failed to determine default branch")
		}
		parentState = meta.BranchState{
			Name:  defaultBranch,
			Trunk: true,
		}
		branchMeta.Parent = parentState
	}
	prCompareRef := parentState.Name
	var parentMeta meta.Branch
	if !parentState.Trunk {
		// check if the base branch also has an associated PR
		var ok bool
		parentMeta, ok = tx.Branch(parentState.Name)
		if !ok {
			return nil, errors.Errorf("failed to read branch metadata for %q", parentState.Name)
		}
		if parentMeta.PullRequest == nil {
			// TODO:
			//     We should automagically create PRs for every branch in the stack
			return nil, errors.Errorf(
				"base branch %q does not have an associated pull request "+
					"(create one by checking out the branch and running 'av pr')",
				parentState.Name,
			)
		}
	} else {
		logrus.WithField("base", parentState.Name).Debug("base branch is a trunk branch")
		prCompareRef = fmt.Sprintf("%s/%s", repo.GetRemoteName(), parentState.Name)
	}

	commitsList, err := repo.Git(
		ctx,
		"rev-list",
		"--reverse",
		fmt.Sprintf("%s..%s", prCompareRef, opts.BranchName),
		"--",
	)
	if err != nil {
		return nil, errors.WrapIf(err, "failed to determine commits to include in PR")
	}
	if commitsList == "" {
		return nil, errors.Errorf("no commits between %q and %q", prCompareRef, opts.BranchName)
	}

	// Check if a parent branch has already been merged or not
	if parentMeta.MergeCommit != "" {
		return nil, errors.Errorf(
			"failed to create a pull request. The parent branch %q has already been merged\nPlease run 'av sync' to rebase the branch first.",
			parentMeta.Name,
		)
	}

	if existingPR != nil {
		// If there's an existing PR, use that as the new PR title and body. If --edit is
		// used, an editor is opened later.
		if opts.Title == "" {
			opts.Title = existingPR.GetTitle()
		}
		if opts.Body == "" {
			opts.Body = existingPR.GetBody()
		}
	}

	if opts.Edit || opts.Body == "" || opts.Title == "" {
		var commits []git.CommitInfo
		for _, commitHash := range strings.Split(commitsList, "\n") {
			commit, err := repo.CommitInfo(ctx, git.CommitInfoOpts{Rev: commitHash})
			if err != nil {
				return nil, errors.WrapIff(err, "failed to get commit info for %q", commitHash)
			}
			commits = append(commits, *commit)
		}

		// If a saved pull request description exists, use that.
		saveFile := filepath.Join(
			repo.AvTmpDir(),
			fmt.Sprintf("av-pr-%s.md", sanitize.FileName(opts.BranchName)),
		)
		if _, err := os.Stat(saveFile); err == nil {
			contents, err := os.ReadFile(saveFile)
			if err != nil {
				logrus.WithError(err).Warn("failed to read saved pull request description")
			} else {
				title, body := stringutils.ParseSubjectBody(string(contents))
				if opts.Title == "" {
					opts.Title = title
				}
				if opts.Body == "" {
					opts.Body = body
				}
			}
		}

		// Try to populate the editor text using contextual information from the
		// repository and commits included in this pull request.
		if opts.Title == "" {
			opts.Title = commits[0].Subject
		}
		// Reasonable defaults for body:
		// 1. Try and find a pull request template
		if opts.Body == "" {
			opts.Body = readDefaultPullRequestTemplate(repo)
		}
		// 2. Use the commit message from the first PR
		if opts.Body == "" {
			opts.Body = commits[0].Body
		}

		editorText := templateutils.MustString(prBodyTemplate, prBodyTemplateData{
			Branch:  opts.BranchName,
			Title:   opts.Title,
			Body:    opts.Body,
			Commits: commits,
		})

		res, err := editor.Launch(ctx, repo, editor.Config{
			Text:           editorText,
			TmpFilePattern: "pr-*.av.md",
			CommentPrefix:  "%%",
		})
		if err != nil {
			if res != "" {
				savePRDescriptionToTemporaryFile(saveFile, res)
			}
			return nil, errors.WrapIf(err, "text editor failed")
		}
		opts.Title, opts.Body = stringutils.ParseSubjectBody(res)
		if opts.Title == "" {
			return nil, errors.New("aborting pull request due to empty message")
		}
		// The tailing new line is needed for compare with `PRMetadataCommentEnd` during the metadata parsing.
		opts.Body += "\n"

		defer func() {
			// If we created the PR successfully, just make sure to clean up any
			// lingering files.
			if reterr == nil {
				_ = os.Remove(saveFile)
				return
			}

			// Otherwise, save what the user entered to a file so that it's not
			// lost forever (and we can reuse it if they try again).
			savePRDescriptionToTemporaryFile(saveFile, res)
		}()
	}

	prMeta, err := getPRMetadata(tx, branchMeta, &parentMeta)
	if err != nil {
		return nil, err
	}

	draft := opts.Draft
	if !config.Av.PullRequest.NoWIPDetection && strings.Contains(opts.Title, "WIP") {
		draft = true
	}

	pull, didCreatePR, err := ensurePR(ctx, provider, repoMeta, tx, ensurePROpts{
		baseRefName: parentState.Name,
		headRefName: opts.BranchName,
		title:       opts.Title,
		body:        opts.Body,
		meta:        prMeta,
		draft:       draft,
		existingPR:  existingPR,
	})
	if err != nil {
		_, _ = fmt.Fprint(os.Stderr,
			colors.Failure("  - failed to create pull request: "), err, "\n",
		)
		return nil, errors.WrapIf(err, "failed to create PR")
	}

	branchMeta.PullRequest = &meta.PullRequest{
		Number:    pull.GetNumber(),
		ID:        pull.GetID(),
		Permalink: pull.GetPermalink(),
	}
	// It's possible that a new PR is created with the same branch. Reset the MergeCommit.
	branchMeta.MergeCommit = ""
	var action string
	if didCreatePR {
		action = "created"
	} else {
		action = "synchronized"
	}
	_, _ = fmt.Fprint(os.Stderr,
		"  - ", action, " pull request ",
		colors.UserInput(pull.GetPermalink()), "\n",
	)

	if didCreatePR && !opts.NoOpenBrowser && config.Av.PullRequest.OpenBrowser {
		OpenPullRequestInBrowser(ctx, pull.GetPermalink())
	}

	tx.SetBranch(branchMeta)
	return &CreatePullRequestResult{didCreatePR, branchMeta, pull}, nil
}

func OpenPullRequestInBrowser(ctx context.Context, pullRequestLink string) {
	if err := browser.Open(ctx, pullRequestLink); err != nil {
		_, _ = fmt.Fprint(os.Stderr,
			"  - couldn't open browser ",
			colors.UserInput(err),
			" for pull request link ",
			colors.UserInput(pullRequestLink),
		)
	}
}

func savePRDescriptionToTemporaryFile(saveFile string, contents string) {
	if err := os.WriteFile(saveFile, []byte(contents), 0o644); err != nil {
		logrus.WithError(err).
			Error("failed to write pull request description to temporary file")
		return
	}
	_, _ = fmt.Fprint(os.Stderr,
		"  - saved pull request description to ", colors.UserInput(saveFile),
		" (it will be automatically re-used if you try again)\n",
	)
}

type prBodyTemplateData struct {
	Branch  string
	Title   string
	Body    string
	Commits []git.CommitInfo
}

var templateFuncs = template.FuncMap{"trimSpace": strings.TrimSpace}

var prBodyTemplate = template.Must(
	template.New("prBody").
		Funcs(templateFuncs).
		Parse(`%% Creating pull request for branch '{{ .Branch }}'
%% Lines starting with '%%' will be ignored and an empty message aborts the
%% creation of the pull request.

%% Pull request title (single line)
{{ .Title }}

%% Pull request body (multiple lines)
{{ .Body }}

%% This branch includes the following commits:
{{- range $c := .Commits }}
%%     {{ $c.ShortHash }}    {{ $c.Subject }}
{{- if trimSpace $c.Body }}
{{- range $line := $c.BodyWithPrefix "%%         " }}
{{ trimSpace $line }}
{{- end }}
{{- end }}
{{- end }}
`),
)

func readDefaultPullRequestTemplate(repo *git.Repo) string {
	for _, dir := range []string{"", ".github", "data"} {
		for _, f := range []string{
			"PULL_REQUEST_TEMPLATE.md",
			"pull_request_template.md",
		} {
			tpl := filepath.Join(repo.Dir(), dir, f)
			data, err := os.ReadFile(tpl)
			if err != nil {
				continue
			}
			return string(data)
		}
	}
	return ""
}

type ensurePROpts struct {
	baseRefName string
	headRefName string
	title       string
	body        string
	meta        PRMetadata
	draft       bool
	existingPR  vcs.PullRequest
}

// ensurePR returns the pull request for the given input, creating a new
// pull request if one doesn't exist. It returns the pull request, a boolean
// indicating whether or not the pull request was created, and an error if one
// occurred.
func ensurePR(
	ctx context.Context,
	provider vcs.Provider,
	repoMeta meta.Repository,
	tx meta.ReadTx,
	opts ensurePROpts,
) (vcs.PullRequest, bool, error) {
	// Don't pass in a stack to start; we'll do a pass over all open PRs in the stack later.
	var initialStack *stackutils.StackTreeNode = nil

	if opts.existingPR != nil {
		newBody := AddPRMetadataAndStack(opts.body, opts.meta, opts.headRefName, initialStack, tx)
		updatedPR, err := provider.UpdatePR(ctx, vcs.UpdatePullRequestInput{
			ID:    opts.existingPR.GetID(),
			Title: &opts.title,
			Body:  &newBody,
		})
		if err != nil {
			return nil, false, errors.WithStack(err)
		}
		return updatedPR, false, nil
	}
	pull, err := provider.CreatePR(ctx, vcs.CreatePullRequestInput{
		RepositoryID: repoMeta.ID,
		BaseRefName:  opts.baseRefName,
		HeadRefName:  opts.headRefName,
		Title:        opts.title,
		Body:         AddPRMetadataAndStack(opts.body, opts.meta, opts.headRefName, initialStack, tx),
		Draft:        opts.draft,
	})
	if err != nil {
		return nil, false, errors.WithStack(err)
	}
	return pull, true, nil
}

type UpdatePullRequestResult struct {
	// True if the pull request information changed (e.g., a new pull request
	// was found or if the pull request changed state)
	Changed bool
	// The pull request object that was returned from the provider
	Pull vcs.PullRequest
}

// UpdatePullRequestState fetches the latest pull request information from the provider
// and writes the relevant branch metadata.
func UpdatePullRequestState(
	ctx context.Context,
	repo *git.Repo,
	tx meta.WriteTx,
	branchName string,
) (*UpdatePullRequestResult, error) {
	repoMeta := tx.Repository()
	branch, _ := tx.Branch(branchName)

	origin, err := repo.Origin(ctx)
	if err != nil {
		return nil, errors.WrapIf(err, "failed to determine repository origin")
	}
	
	provider, err := vcs.DetectAndCreateProvider(ctx, origin.URL.String())
	if err != nil {
		return nil, errors.WrapIf(err, "failed to create provider")
	}

	page, err := provider.GetPRs(ctx, vcs.GetPullRequestsInput{
		Owner:       repoMeta.Owner,
		Repo:        repoMeta.Name,
		HeadRefName: branchName,
		First:       100,
	})
	if err != nil {
		return nil, errors.WrapIf(
			err,
			"querying pull requests. Make sure provider token is set or refresh.\nSee: https://docs.aviator.co/aviator-cli#getting-started",
		)
	}

	if len(page.PullRequests) == 0 {
		// branch has no pull request
		if branch.PullRequest != nil {
			// This should never happen?
			logrus.WithFields(logrus.Fields{
				"branch": branch.Name,
				"pull":   branch.PullRequest.Permalink,
			}).Error("Provider reported no pull requests for branch but local metadata has pull request")
			return nil, errors.New(
				"Provider reported no pull requests for branch but local metadata has pull request",
			)
		}

		return &UpdatePullRequestResult{false, nil}, nil
	}

	// The latest info for the pull request that we have stored in local metadata
	// (we can use this to check if the pull was closed/merged)
	var currentPull vcs.PullRequest
	// The current open pull request (if any)
	var openPull vcs.PullRequest
	for _, pull := range page.PullRequests {
		if branch.PullRequest != nil && pull.GetID() == branch.PullRequest.ID {
			currentPull = pull
		}
		if !pull.IsOpen() {
			continue
		}
		// Providers only allow one open pull for a given (head, base) pair, but
		// we only support one open pull per head branch (the workflow of
		// opening a pull from a head branch into multiple base branches is
		// rare). This probably isn't necessary but better to be defensive
		// here.
		if openPull != nil {
			return nil, errors.Errorf(
				"multiple open pull requests for branch %q (#%d into %q and #%d into %q)",
				branchName,
				openPull.GetNumber(), openPull.BaseBranchName(),
				pull.GetNumber(), pull.BaseBranchName(),
			)
		}
		openPull = pull
	}

	changed := false
	var oldId string
	if branch.PullRequest != nil {
		oldId = branch.PullRequest.ID
	}

	var newPull vcs.PullRequest
	if openPull != nil {
		if oldId != openPull.GetID() {
			changed = true
		}
		state := "open"
		if openPull.IsMerged() {
			state = "merged"
		} else if !openPull.IsOpen() {
			state = "closed"
		}
		branch.PullRequest = &meta.PullRequest{
			ID:        openPull.GetID(),
			Number:    openPull.GetNumber(),
			Permalink: openPull.GetPermalink(),
			State:     githubv4.PullRequestState(state),
		}
		newPull = openPull
	} else {
		// openPull is nil so the PR should be merged or closed
		if currentPull != nil {
			branch.MergeCommit = currentPull.GetMergeCommit()
			state := "closed"
			if currentPull.IsMerged() {
				state = "merged"
			}
			branch.PullRequest = &meta.PullRequest{
				ID:        currentPull.GetID(),
				Number:    currentPull.GetNumber(),
				Permalink: currentPull.GetPermalink(),
				State:     githubv4.PullRequestState(state),
			}
		} else {
			// openPull and currentPull is nil
			if branch.PullRequest != nil {
				changed = true
			}
			branch.PullRequest = nil
		}
		newPull = currentPull
	}

	tx.SetBranch(branch)
	return &UpdatePullRequestResult{changed, newPull}, nil
}

type PRMetadata struct {
	Parent     string `json:"parent"`
	ParentHead string `json:"parentHead"`
	ParentPull int64  `json:"parentPull,omitempty"`
	Trunk      string `json:"trunk"`
}

const PRMetadataCommentStart = "<!-- av pr metadata"

const (
	PRMetadataCommentHelpText = "This information is embedded by the av CLI when creating PRs to track the status of stacks when using Aviator. Please do not delete or edit this section of the PR.\n"
	PRMetadataCommentEnd      = "-->"
)

const (
	PRStackCommentStart = "<!-- av pr stack begin -->"
	PRStackCommentEnd   = "<!-- av pr stack end -->"
)

// extractContent parses the given input and looks for the start and end
// strings. It returns the content between the start and end strings and the
// remaining input. If the start or end strings are not found, the content is
// empty and the input is returned as-is.
func extractContent(input string, start string, end string) (content string, output string) {
	startIndex := strings.Index(input, start)
	if startIndex == -1 {
		return "", input
	}
	contentIndex := startIndex + len(start)
	endIndex := strings.Index(input[contentIndex:], end)
	if endIndex == -1 {
		return "", input
	}

	content = strings.TrimSpace(input[contentIndex : contentIndex+endIndex])
	preContent := strings.TrimSpace(input[:startIndex])
	postContent := strings.TrimSpace(input[contentIndex+endIndex+len(end):])
	output = preContent
	if postContent != "" {
		output += "\n" + postContent
	}
	return
}

func ParsePRBody(input string) (body string, prMeta PRMetadata, retErr error) {
	metadata, body := extractContent(input, PRMetadataCommentStart, PRMetadataCommentEnd)
	metadataContent, _ := extractContent(metadata, "```", "```")
	if err := json.Unmarshal([]byte(metadataContent), &prMeta); err != nil {
		retErr = errors.WrapIff(err, "decoding PR metadata")
		return
	}

	_, body = extractContent(body, PRStackCommentStart, PRStackCommentEnd)

	return
}

func ReadPRMetadata(body string) (PRMetadata, error) {
	_, prMeta, err := ParsePRBody(body)
	return prMeta, err
}

func walkStack(tx meta.ReadTx, stack *stackutils.StackTreeNode, branchName string) string {
	ssb := strings.Builder{}

	// For simple stacks (i.e., degenerate trees) print them top-down. For example:
	// - #1
	// - #2
	// - main
	var visitSimple func(node *stackutils.StackTreeNode, depth int)
	visitSimple = func(node *stackutils.StackTreeNode, depth int) {
		bi, _ := tx.Branch(node.Branch.BranchName)
		if len(node.Children) > 1 {
			panic("stack tree has more than one child")
		} else if len(node.Children) == 1 {
			visitSimple(node.Children[0], depth+1)
		}

		ssb.WriteString("* ")

		if depth == 0 || bi.PullRequest == nil {
			ssb.WriteString("`")
			ssb.WriteString(node.Branch.BranchName)
			ssb.WriteString("`")
		} else {
			if node.Branch.BranchName == branchName {
				ssb.WriteString("➡️ ")
			}
			ssb.WriteString("**#")
			ssb.WriteString(strconv.FormatInt(bi.PullRequest.Number, 10))
			ssb.WriteString("**")
		}
		ssb.WriteString("\n")
	}

	// For more complex stacks, print them sideways using a bulleted list. For example:
	// - main
	//   - #1
	//     - #2
	//   - #3
	var visitComplex func(node *stackutils.StackTreeNode, depth int)
	visitComplex = func(node *stackutils.StackTreeNode, depth int) {
		if depth == 0 {
			ssb.WriteString("* ")
			ssb.WriteString("`")
			ssb.WriteString(node.Branch.BranchName)
			ssb.WriteString("`")
		} else {
			bi, _ := tx.Branch(node.Branch.BranchName)
			ssb.WriteString(strings.Repeat("  ", depth))
			ssb.WriteString("* ")
			if node.Branch.BranchName == branchName {
				ssb.WriteString("➡️ ")
			}
			if bi.PullRequest != nil {
				ssb.WriteString("**#")
				ssb.WriteString(strconv.FormatInt(bi.PullRequest.Number, 10))
				ssb.WriteString("**")
			} else {
				ssb.WriteString("`")
				ssb.WriteString(node.Branch.BranchName)
				ssb.WriteString("`")
			}
		}
		ssb.WriteString("\n")

		for _, child := range node.Children {
			visitComplex(child, depth+1)
		}
	}

	var hasMultipleChildren func(node *stackutils.StackTreeNode) bool
	hasMultipleChildren = func(node *stackutils.StackTreeNode) bool {
		if len(node.Children) > 1 {
			return true
		} else if len(node.Children) == 1 {
			return hasMultipleChildren(node.Children[0])
		}
		return false
	}

	// Optimize navigation within a stack by making sure the output has the same shape everywhere.
	if hasMultipleChildren(stack) {
		visitComplex(stack, 0)
	} else {
		visitSimple(stack, 0)
	}

	return ssb.String()
}

func AddPRMetadataAndStack(
	body string,
	prMeta PRMetadata,
	branchName string,
	stack *stackutils.StackTreeNode,
	tx meta.ReadTx,
) string {
	body, _, err := ParsePRBody(body)
	if err != nil {
		// No existing metadata comment, so add one.
		logrus.WithError(err).Debug("could not parse PR metadata (assuming it doesn't exist)")
	}

	sb := strings.Builder{}

	// Don't write out a stack unless there is more than one PR in it.
	hasMultilevelStack := stack != nil && len(stack.Children) > 0 &&
		len(stack.Children[0].Children) > 0
	if hasMultilevelStack {
		bi, _ := tx.Branch(branchName)
		stackString := walkStack(tx, stack, branchName)
		sb.WriteString(PRStackCommentStart)

		// Enclose this stack summary in a table for two reasons:
		// 1. It actually looks nicer on GitHub
		// 2. For the Slack GitHub integration, Slack doesn't support and strips out <table> elements in unfurls - we can avoid showing the stack in the unfurl.
		sb.WriteString("\n<table><tr><td>")
		sb.WriteString("<details><summary>")
		if !bi.Parent.Trunk {
			parentBi, _ := tx.Branch(bi.Parent.Name)
			sb.WriteString("<b>Depends on #")
			sb.WriteString(strconv.FormatInt(parentBi.PullRequest.Number, 10))
			sb.WriteString(".</b> ")
		}
		sb.WriteString(
			"This PR is part of a stack created with <a href=\"https://github.com/aviator-co/av\">Aviator</a>.",
		)
		sb.WriteString("</summary>")
		sb.WriteString("\n\n")
		sb.WriteString(stackString)
		sb.WriteString("</details>")
		sb.WriteString("</td></tr></table>\n")
		sb.WriteString(PRStackCommentEnd)
		sb.WriteString("\n\n")
	}

	sb.WriteString(body)

	sb.WriteString("\n\n")
	sb.WriteString(PRMetadataCommentStart)
	sb.WriteString("\n")
	sb.WriteString(PRMetadataCommentHelpText)
	sb.WriteString("```\n")

	// Note: Encoder.Encode implicitly adds a newline at the end of the JSON
	// which is important here so that the ``` below appears on its own line.
	if err := json.NewEncoder(&sb).Encode(prMeta); err != nil {
		// shouldn't ever happen since we're encoding a simple struct to a buffer
		panic(errors.WrapIff(err, "encoding PR metadata"))
	}
	sb.WriteString("```\n")
	sb.WriteString(PRMetadataCommentEnd)
	sb.WriteString("\n")

	return sb.String()
}

// UpdatePullRequestWithStack updates the pull request associated with the given branch to include
// the stack of branches that the branch is a part of.
// This should be called after all applicable PRs have been created to ensure we can properly link them.
func UpdatePullRequestWithStack(
	ctx context.Context,
	repo *git.Repo,
	tx meta.WriteTx,
	branchName string,
) error {
	branchMeta, exists := tx.Branch(branchName)
	// it's possible that this branch is not part of the primary stack, ex: they have main->branchA->branchB,
	// but forkA is coming from branchA in which case forkA may not have a pull request associated with it.
	// In this case, we should not try to update forkA's pull request with the stack.
	if !exists || branchMeta.PullRequest == nil {
		return nil
	}
	logrus.WithField("branch", branchName).
		WithField("pr", branchMeta.PullRequest.ID).
		Debug("Updating pull requests with stack")

	repoMeta := tx.Repository()
	
	origin, err := repo.Origin(ctx)
	if err != nil {
		return errors.WrapIf(err, "failed to determine repository origin")
	}
	
	provider, err := vcs.DetectAndCreateProvider(ctx, origin.URL.String())
	if err != nil {
		return errors.WrapIf(err, "failed to create provider")
	}

	// Don't sort based on the current branch so that the output is consistent between branches.
	stackToWrite, err := stackutils.BuildStackTreeCurrentStack(tx, branchName, false)
	if err != nil {
		return err
	}

	existingPR, err := getExistingOpenPR(ctx, provider, repoMeta, branchMeta, branchName)
	if err != nil {
		return errors.WithStack(err)
	}

	body, prMeta, err := ParsePRBody(existingPR.GetBody())
	if err != nil {
		return err
	}

	newBody := AddPRMetadataAndStack(body, prMeta, branchName, stackToWrite, tx)
	_, err = provider.UpdatePR(ctx, vcs.UpdatePullRequestInput{
		ID:   existingPR.GetID(),
		Body: &newBody,
	})
	if err != nil {
		return errors.WithStack(err)
	}

	return nil
}

// UpdatePullRequestsWithStack updates the pull requests associated with the given branches to include
// the stack of branches that each branch is a part of.
func UpdatePullRequestsWithStack(
	ctx context.Context,
	repo *git.Repo,
	tx meta.WriteTx,
	branchNames []string,
) error {
	for _, branchName := range branchNames {
		if err := UpdatePullRequestWithStack(ctx, repo, tx, branchName); err != nil {
			return err
		}
	}

	return nil
}
