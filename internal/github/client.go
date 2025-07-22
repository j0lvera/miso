package github

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"os"
	"sort"
	"strings"

	"github.com/google/go-github/v57/github"
	"golang.org/x/oauth2"
)

const BotCommentIdentifier = "🍲 miso Code review"

type Client struct {
	client *github.Client
	owner  string
	repo   string
}

type PREvent struct {
	PullRequest struct {
		Number int `json:"number"`
		Base   struct {
			SHA string `json:"sha"`
		} `json:"base"`
		Head struct {
			SHA string `json:"sha"`
		} `json:"head"`
	} `json:"pull_request"`
}

func NewClient(token string) (*Client, error) {
	if token == "" {
		token = os.Getenv("GITHUB_TOKEN")
		if token == "" {
			return nil, fmt.Errorf("GitHub token not provided and GITHUB_TOKEN not set")
		}
	}

	// Parse GITHUB_REPOSITORY env var (format: owner/repo)
	repoEnv := os.Getenv("GITHUB_REPOSITORY")
	if repoEnv == "" {
		return nil, fmt.Errorf("GITHUB_REPOSITORY environment variable not set")
	}

	parts := strings.Split(repoEnv, "/")
	if len(parts) != 2 {
		return nil, fmt.Errorf("invalid GITHUB_REPOSITORY format: %s", repoEnv)
	}

	ctx := context.Background()
	ts := oauth2.StaticTokenSource(
		&oauth2.Token{AccessToken: token},
	)
	tc := oauth2.NewClient(ctx, ts)

	return &Client{
		client: github.NewClient(tc),
		owner:  parts[0],
		repo:   parts[1],
	}, nil
}

func (c *Client) GetPRInfo() (*PREvent, error) {
	eventPath := os.Getenv("GITHUB_EVENT_PATH")
	if eventPath == "" {
		return nil, fmt.Errorf("GITHUB_EVENT_PATH not set")
	}

	data, err := os.ReadFile(eventPath)
	if err != nil {
		return nil, fmt.Errorf("failed to read event file: %w", err)
	}

	var event PREvent
	if err := json.Unmarshal(data, &event); err != nil {
		return nil, fmt.Errorf("failed to parse event JSON: %w", err)
	}

	return &event, nil
}

// findAllBotComments is a helper to find all comments by the bot matching an identifier.
func (c *Client) findAllBotComments(ctx context.Context, prNumber int, identifier string) ([]*github.IssueComment, error) {
	var botComments []*github.IssueComment
	opts := &github.IssueListCommentsOptions{
		ListOptions: github.ListOptions{PerPage: 100},
	}

	for {
		comments, resp, err := c.client.Issues.ListComments(
			ctx, c.owner, c.repo, prNumber, opts,
		)
		if err != nil {
			if _, ok := err.(*github.RateLimitError); ok {
				return nil, fmt.Errorf("GitHub API rate limit exceeded, please try again later")
			}
			return nil, err
		}

		for _, comment := range comments {
			if comment.User != nil && comment.User.GetType() == "Bot" &&
				strings.Contains(comment.GetBody(), identifier) {
				botComments = append(botComments, comment)
			}
		}

		if resp.NextPage == 0 {
			break
		}
		opts.Page = resp.NextPage
	}

	return botComments, nil
}

// sortCommentsByCreatedDesc sorts comments by creation time, newest first.
func sortCommentsByCreatedDesc(comments []*github.IssueComment) {
	sort.Slice(comments, func(i, j int) bool {
		return comments[i].GetCreatedAt().After(comments[j].GetCreatedAt().Time)
	})
}

func (c *Client) FindBotComment(ctx context.Context, prNumber int, identifier string) (*github.IssueComment, error) {
	allComments, err := c.findAllBotComments(ctx, prNumber, identifier)
	if err != nil {
		return nil, err
	}

	if len(allComments) == 0 {
		return nil, nil
	}

	// Sort by creation time to find the most recent one
	sortCommentsByCreatedDesc(allComments)

	return allComments[0], nil
}

func (c *Client) PostOrUpdateComment(ctx context.Context, prNumber int, content string) error {
	identifier := BotCommentIdentifier

	// Find existing comment
	existing, err := c.FindBotComment(ctx, prNumber, identifier)
	if err != nil {
		return fmt.Errorf("failed to find existing comment: %w", err)
	}

	if existing != nil {
		// Update existing comment
		_, _, err = c.client.Issues.EditComment(
			ctx, c.owner, c.repo, existing.GetID(),
			&github.IssueComment{Body: &content},
		)
		if _, ok := err.(*github.RateLimitError); ok {
			return fmt.Errorf("GitHub API rate limit exceeded, please try again later")
		}
		return err
	}

	// Create new comment
	_, _, err = c.client.Issues.CreateComment(
		ctx, c.owner, c.repo, prNumber,
		&github.IssueComment{Body: &content},
	)
	if _, ok := err.(*github.RateLimitError); ok {
		return fmt.Errorf("GitHub API rate limit exceeded, please try again later")
	}
	return err
}

// CleanupOldComments finds all comments posted by the bot with the specific identifier
// and deletes all but the most recent one.
func (c *Client) CleanupOldComments(ctx context.Context, prNumber int) error {
	identifier := BotCommentIdentifier
	allBotComments, err := c.findAllBotComments(ctx, prNumber, identifier)
	if err != nil {
		return fmt.Errorf("failed to find bot comments for cleanup: %w", err)
	}

	if len(allBotComments) <= 1 {
		return nil // Nothing to clean up
	}

	// Sort comments by creation time, newest first
	sortCommentsByCreatedDesc(allBotComments)

	// Keep the first one (newest), delete the rest
	commentsToDelete := allBotComments[1:]
	if len(commentsToDelete) > 0 {
		log.Printf("Cleaning up %d old bot comments for PR #%d", len(commentsToDelete), prNumber)
	}
	for _, comment := range commentsToDelete {
		_, err := c.client.Issues.DeleteComment(ctx, c.owner, c.repo, comment.GetID())
		if err != nil {
			if _, ok := err.(*github.RateLimitError); ok {
				// Rate limit hit, stop trying to delete more comments
				log.Printf("GitHub API rate limit exceeded while cleaning up comments, stopping cleanup")
				return fmt.Errorf("GitHub API rate limit exceeded during cleanup")
			}
			// Log error but continue trying to delete others
			log.Printf("failed to delete old bot comment #%d: %v", comment.GetID(), err)
		}
	}

	return nil
}
