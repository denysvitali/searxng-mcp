package server

import (
	"context"
	"net/url"
	"testing"

	"github.com/h2non/gock"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestFetchGitHubThread_IssueStructuredOutput(t *testing.T) {
	defer gock.OffAll()

	gock.New("https://api.github.com").
		Get("/repos/kubernetes/kubernetes/issues/22368").
		Reply(200).
		JSON(loadJSONFixture(t, "github_issue_22368_issue.json"))

	gock.New("https://api.github.com").
		Get("/repos/kubernetes/kubernetes/issues/22368/comments").
		Reply(200).
		JSON(loadJSONFixture(t, "github_issue_22368_comments.json"))

	parsedURL, err := url.Parse("https://github.com/kubernetes/kubernetes/issues/22368")
	require.NoError(t, err)

	thread, err := fetchGitHubThread(context.Background(), newHTTPClient(), parsedURL)
	require.NoError(t, err)

	assert.Equal(t, "kubernetes", thread.Owner)
	assert.Equal(t, "kubernetes", thread.Repo)
	assert.Equal(t, 22368, thread.Number)
	assert.Equal(t, GitHubThreadIssue, thread.Kind)
	assert.Equal(t, "Feature request: example issue", thread.Title)
	assert.Equal(t, "open", thread.State)
	assert.Equal(t, "issue-author", thread.Author)
	assert.Equal(t, "Issue body from Kubernetes.", thread.Body)
	assert.Equal(t, []string{"kind/feature", "sig/api-machinery"}, thread.Labels)
	assert.Len(t, thread.Comments, 2)
	assert.Equal(t, "commenter-one", thread.Comments[0].Author)
	assert.Equal(t, "First comment body.", thread.Comments[0].Body)
	assert.Equal(t, "commenter-two", thread.Comments[1].Author)
	assert.Equal(t, "Second comment body.", thread.Comments[1].Body)
	assert.True(t, gock.IsDone(), "expected all mocked GitHub endpoints to be called")
}

func TestFetchURLContent_GitHubIssueMarkdown(t *testing.T) {
	defer gock.OffAll()

	gock.New("https://api.github.com").
		Get("/repos/kubernetes/kubernetes/issues/22368").
		Reply(200).
		JSON(loadJSONFixture(t, "github_issue_22368_issue.json"))

	gock.New("https://api.github.com").
		Get("/repos/kubernetes/kubernetes/issues/22368/comments").
		Reply(200).
		JSON(loadJSONFixture(t, "github_issue_22368_comments.json"))

	markdown, err := fetchURLContent(context.Background(), "https://github.com/kubernetes/kubernetes/issues/22368")
	require.NoError(t, err)
	assert.Contains(t, markdown, "# kubernetes/kubernetes #22368: Feature request: example issue")
	assert.Contains(t, markdown, "## Comments (2)")
	assert.Contains(t, markdown, "First comment body.")
	assert.True(t, gock.IsDone(), "expected all mocked GitHub endpoints to be called")
}

func TestFetchURLContent_GitHubRepoMarkdown(t *testing.T) {
	defer gock.OffAll()

	gock.New("https://api.github.com").
		Get("/repos/denysvitali/searxng-mcp").
		Reply(200).
		JSON(map[string]interface{}{
			"full_name":         "denysvitali/searxng-mcp",
			"description":       "MCP server for Searxng",
			"html_url":          "https://github.com/denysvitali/searxng-mcp",
			"homepage":          "",
			"stargazers_count":  2,
			"forks_count":       1,
			"open_issues_count": 0,
			"language":          "Go",
			"topics":            []string{"mcp", "searxng"},
			"archived":          false,
			"default_branch":    "master",
			"license":           map[string]interface{}{"spdx_id": "MIT"},
		})

	gock.New("https://api.github.com").
		Get("/repos/denysvitali/searxng-mcp/readme").
		Reply(200).
		BodyString("# searxng-mcp\n\nA test README.")

	markdown, err := fetchURLContent(context.Background(), "https://github.com/denysvitali/searxng-mcp")
	require.NoError(t, err)
	assert.Contains(t, markdown, "# denysvitali/searxng-mcp")
	assert.Contains(t, markdown, "MCP server for Searxng")
	assert.Contains(t, markdown, "- Primary language: Go")
	assert.Contains(t, markdown, "- Stars: 2")
	assert.Contains(t, markdown, "- License: MIT")
	assert.Contains(t, markdown, "- Topics: mcp, searxng")
	assert.Contains(t, markdown, "## README")
	assert.Contains(t, markdown, "A test README.")
	assert.True(t, gock.IsDone(), "expected all mocked GitHub endpoints to be called")
}

func TestFetchGitHubThread_PullRequestIncludesReviewComments(t *testing.T) {
	defer gock.OffAll()

	gock.New("https://api.github.com").
		Get("/repos/example/repo/issues/10").
		Reply(200).
		JSON(loadJSONFixture(t, "github_pr_10_issue.json"))

	gock.New("https://api.github.com").
		Get("/repos/example/repo/issues/10/comments").
		Reply(200).
		JSON(loadJSONFixture(t, "github_pr_10_issue_comments.json"))

	gock.New("https://api.github.com").
		Get("/repos/example/repo/pulls/10").
		Reply(200).
		JSON(loadJSONFixture(t, "github_pr_10_pull.json"))

	gock.New("https://api.github.com").
		Get("/repos/example/repo/pulls/10/comments").
		Reply(200).
		JSON(loadJSONFixture(t, "github_pr_10_pull_comments.json"))

	parsedURL, err := url.Parse("https://github.com/example/repo/pull/10")
	require.NoError(t, err)

	thread, err := fetchGitHubThread(context.Background(), newHTTPClient(), parsedURL)
	require.NoError(t, err)

	assert.Equal(t, GitHubThreadPullRequest, thread.Kind)
	require.NotNil(t, thread.PullRequest)
	assert.Equal(t, "main", thread.PullRequest.BaseRef)
	assert.Equal(t, "feature-branch", thread.PullRequest.HeadRef)
	assert.Equal(t, 2, thread.PullRequest.Commits)
	assert.Len(t, thread.ReviewComments, 1)
	assert.Equal(t, "reviewer", thread.ReviewComments[0].Author)
	assert.Equal(t, "pkg/server/server.go", thread.ReviewComments[0].Path)
	assert.Contains(t, renderGitHubThreadMarkdown(thread), "## Review Comments (1)")
	assert.True(t, gock.IsDone(), "expected all mocked GitHub endpoints to be called")
}
