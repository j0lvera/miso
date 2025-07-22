package github

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"strings"

	"github.com/google/go-github/v57/github"
	"golang.org/x/oauth2"
)

type Client struct {
	client *github.Client
	owner  string
	repo   string
	ctx    context.Context
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
		ctx:    ctx,
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

func (c *Client) FindBotComment(prNumber int, identifier string) (*github.IssueComment, error) {
	opts := &github.IssueListCommentsOptions{
		ListOptions: github.ListOptions{PerPage: 100},
	}

	comments, _, err := c.client.Issues.ListComments(
		c.ctx, c.owner, c.repo, prNumber, opts,
	)
	if err != nil {
		if _, ok := err.(*github.RateLimitError); ok {
			return nil, fmt.Errorf("GitHub API rate limit exceeded, please try again later")
		}
		return nil, err
	}

	for _, comment := range comments {
		if comment.User.GetType() == "Bot" &&
			strings.Contains(comment.GetBody(), identifier) {
			return comment, nil
		}
	}

	return nil, nil
}

func (c *Client) PostOrUpdateComment(prNumber int, content string) error {
	identifier := "🍲 miso Code review"

	// Find existing comment
	existing, err := c.FindBotComment(prNumber, identifier)
	if err != nil {
		return fmt.Errorf("failed to find existing comment: %w", err)
	}

	if existing != nil {
		// Update existing comment
		_, _, err = c.client.Issues.EditComment(
			c.ctx, c.owner, c.repo, existing.GetID(),
			&github.IssueComment{Body: &content},
		)
		if _, ok := err.(*github.RateLimitError); ok {
			return fmt.Errorf("GitHub API rate limit exceeded, please try again later")
		}
		return err
	}

	// Create new comment
	_, _, err = c.client.Issues.CreateComment(
		c.ctx, c.owner, c.repo, prNumber,
		&github.IssueComment{Body: &content},
	)
	if _, ok := err.(*github.RateLimitError); ok {
		return fmt.Errorf("GitHub API rate limit exceeded, please try again later")
	}
	return err
}
