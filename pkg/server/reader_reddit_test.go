package server

import (
	"context"
	"fmt"
	"net/url"
	"testing"

	"github.com/h2non/gock"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestFetchRedditContentAsMarkdown_UsesJSONEndpoint(t *testing.T) {
	defer gock.OffAll()

	gock.New("https://www.reddit.com").
		Get("/r/ClaudeAI/comments/1r2zjgl/anyone_feel_everything_has_changed_over_the_last.json").
		Reply(200).
		JSON(loadJSONFixture(t, "reddit_thread_claudeai.json"))

	parsedURL, err := url.Parse("https://www.reddit.com/r/ClaudeAI/comments/1r2zjgl/anyone_feel_everything_has_changed_over_the_last/")
	require.NoError(t, err)

	markdown, err := fetchRedditContentAsMarkdown(context.Background(), newHTTPClient(), parsedURL)
	require.NoError(t, err)

	assert.Contains(t, markdown, "# Anyone feel everything has changed over the last year?")
	assert.Contains(t, markdown, "## Post")
	assert.Contains(t, markdown, "## Comments")
	assert.Contains(t, markdown, "Top level comment")
	assert.True(t, gock.IsDone(), "expected mocked Reddit JSON endpoint to be called")
}

func TestFetchURLContent_RedditThreadUsesJSONEndpoint(t *testing.T) {
	defer gock.OffAll()

	gock.New("https://www.reddit.com").
		Get("/r/ClaudeAI/comments/1r2zjgl/anyone_feel_everything_has_changed_over_the_last.json").
		Reply(200).
		JSON(loadJSONFixture(t, "reddit_thread_claudeai.json"))

	markdown, err := fetchURLContent(context.Background(), "https://www.reddit.com/r/ClaudeAI/comments/1r2zjgl/anyone_feel_everything_has_changed_over_the_last/")
	require.NoError(t, err)
	assert.Contains(t, markdown, "Anyone feel everything has changed over the last year?")
	assert.True(t, gock.IsDone(), "expected mocked Reddit JSON endpoint to be called")
}

func TestFetchRedditThread_DepthLimit(t *testing.T) {
	defer gock.OffAll()

	gock.New("https://www.reddit.com").
		Get("/r/ClaudeAI/comments/1r2zjgl/anyone_feel_everything_has_changed_over_the_last.json").
		Reply(200).
		JSON(loadJSONFixture(t, "reddit_thread_claudeai.json"))

	parsedURL, err := url.Parse("https://www.reddit.com/r/ClaudeAI/comments/1r2zjgl/anyone_feel_everything_has_changed_over_the_last/")
	require.NoError(t, err)

	thread, err := fetchRedditThread(context.Background(), newHTTPClient(), parsedURL)
	require.NoError(t, err)

	require.Len(t, thread.Comments, 1)
	require.Len(t, thread.Comments[0].Replies, 1)
	assert.Len(t, thread.Comments[0].Replies[0].Replies, 0, "depth should be capped at top-level + one reply level")

	markdown := renderRedditThreadMarkdown(thread)
	assert.Contains(t, markdown, "Reply level 1")
	assert.NotContains(t, markdown, "Reply level 2")
}

func TestRenderRedditThreadMarkdown_TruncatesCommentsAndReplies(t *testing.T) {
	thread := &RedditThread{
		Title:       "Thread title",
		Subreddit:   "ClaudeAI",
		Author:      "author",
		Score:       10,
		NumComments: redditTopCommentLimit + 1,
		Body:        "Body",
		Comments:    make([]RedditComment, 0, redditTopCommentLimit+1),
	}

	for idx := 0; idx < redditTopCommentLimit+1; idx++ {
		comment := RedditComment{
			Author: fmt.Sprintf("commenter-%d", idx),
			Body:   "Comment body",
			Score:  idx,
		}
		if idx == 0 {
			for replyIndex := 0; replyIndex < redditReplyPerCommentLimit+1; replyIndex++ {
				comment.Replies = append(comment.Replies, RedditComment{
					Author: fmt.Sprintf("replier-%d", replyIndex),
					Body:   "Reply body",
					Score:  replyIndex,
				})
			}
		}
		thread.Comments = append(thread.Comments, comment)
	}

	markdown := renderRedditThreadMarkdown(thread)
	assert.Contains(t, markdown, "_... 1 more top-level comments omitted._")
	assert.Contains(t, markdown, "_... 1 more replies omitted._")
}
