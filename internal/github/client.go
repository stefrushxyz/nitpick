package github

import (
	"context"
	"fmt"
	"slices"
	"sort"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/google/go-github/v57/github"
	"golang.org/x/oauth2"
)

// Client wraps the GitHub API client
type Client struct {
	gh *github.Client
}

// Messages for async operations
type ReposMsg struct {
	Repos []*github.Repository
	Err   error
}

// PRsMsg is a message containing pull requests
type PRsMsg struct {
	PRs []*github.PullRequest
	Err error
}

// CommentsMsg is a message containing pull request comments
type CommentsMsg struct {
	Comments []*github.PullRequestComment
	Err      error
}

// New creates a new GitHub client
func New(token string) *Client {
	ctx := context.Background()
	ts := oauth2.StaticTokenSource(
		&oauth2.Token{AccessToken: token},
	)
	tc := oauth2.NewClient(ctx, ts)
	gh := github.NewClient(tc)

	return &Client{gh: gh}
}

// FetchRepos fetches all repositories (personal and organizational)
func (c *Client) FetchRepos() tea.Cmd {
	return func() tea.Msg {
		ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
		defer cancel()

		// Fetch user repositories
		opts := &github.RepositoryListOptions{
			ListOptions: github.ListOptions{PerPage: 100},
			Sort:        "updated",
			Direction:   "desc",
		}

		var allRepos []*github.Repository

		// Get user repos
		userRepos, _, err := c.gh.Repositories.List(ctx, "", opts)
		if err != nil {
			return ReposMsg{Err: err}
		}
		allRepos = append(allRepos, userRepos...)

		// Get organization repos
		orgs, _, err := c.gh.Organizations.List(ctx, "", nil)
		if err == nil {
			for _, org := range orgs {
				orgRepos, _, err := c.gh.Repositories.ListByOrg(ctx, org.GetLogin(), &github.RepositoryListByOrgOptions{
					ListOptions: github.ListOptions{PerPage: 100},
					Sort:        "updated",
					Direction:   "desc",
				})
				if err == nil {
					allRepos = append(allRepos, orgRepos...)
				}
			}
		}

		return ReposMsg{Repos: allRepos}
	}
}

// FetchPRs fetches pull requests for the given repository
func (c *Client) FetchPRs(repo *github.Repository) tea.Cmd {
	return func() tea.Msg {
		if repo == nil {
			return PRsMsg{Err: fmt.Errorf("no repository provided")}
		}

		ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
		defer cancel()

		opts := &github.PullRequestListOptions{
			State:       "open",
			ListOptions: github.ListOptions{PerPage: 100},
		}

		prs, _, err := c.gh.PullRequests.List(ctx, repo.GetOwner().GetLogin(), repo.GetName(), opts)
		if err != nil {
			return PRsMsg{Err: err}
		}

		// Sort PRs by number in descending order (highest PR number first)
		sort.Slice(prs, func(i, j int) bool {
			return prs[i].GetNumber() > prs[j].GetNumber()
		})

		return PRsMsg{PRs: prs}
	}
}

// FetchComments fetches comments for the given pull request
func (c *Client) FetchComments(repo *github.Repository, pr *github.PullRequest) tea.Cmd {
	return func() tea.Msg {
		if repo == nil || pr == nil {
			return CommentsMsg{Err: fmt.Errorf("no repository or PR provided")}
		}

		ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
		defer cancel()

		opts := &github.PullRequestListCommentsOptions{
			ListOptions: github.ListOptions{PerPage: 100},
		}

		comments, _, err := c.gh.PullRequests.ListComments(ctx,
			repo.GetOwner().GetLogin(),
			repo.GetName(),
			pr.GetNumber(),
			opts)
		if err != nil {
			return CommentsMsg{Err: err}
		}

		// Filter for unresolved comments
		unresolvedComments := slices.Clone(comments)

		// Sort comments by UpdatedAt timestamp in descending order (most recently updated first)
		sort.Slice(unresolvedComments, func(i, j int) bool {
			if unresolvedComments[i].UpdatedAt == nil && unresolvedComments[j].UpdatedAt == nil {
				return false
			}
			if unresolvedComments[i].UpdatedAt == nil {
				return false
			}
			if unresolvedComments[j].UpdatedAt == nil {
				return true
			}

			// Sort by most recent first (descending order)
			return unresolvedComments[i].UpdatedAt.Time.After(unresolvedComments[j].UpdatedAt.Time)
		})

		return CommentsMsg{Comments: unresolvedComments}
	}
}
