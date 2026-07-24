package api

import (
	"context"
	"fmt"
	"net/url"
	"strconv"
)

// Comment severity. BLOCKER comments are tasks: they block the merge until
// they are resolved.
const (
	SeverityNormal  = "NORMAL"
	SeverityBlocker = "BLOCKER"
)

// Comment state. PENDING comments belong to an unpublished review and are only
// visible to their author until the review is completed.
const (
	CommentStateOpen     = "OPEN"
	CommentStatePending  = "PENDING"
	CommentStateResolved = "RESOLVED"
)

// CommentAnchor pins a comment to a file, or to a single line within a file.
type CommentAnchor struct {
	DiffType string `json:"diffType,omitempty"`
	LineType string `json:"lineType,omitempty"`
	FileType string `json:"fileType,omitempty"`
	Line     int    `json:"line,omitempty"`
	Path     string `json:"path,omitempty"`
	SrcPath  string `json:"srcPath,omitempty"`
	FromHash string `json:"fromHash,omitempty"`
	ToHash   string `json:"toHash,omitempty"`
	// Orphaned is set by the server when the anchored line no longer exists
	// in the current diff, e.g. after a force push.
	Orphaned bool `json:"orphaned,omitempty"`
}

// Location renders the anchor as path:line for display.
func (a *CommentAnchor) Location() string {
	if a == nil || a.Path == "" {
		return ""
	}
	if a.Line == 0 {
		return a.Path
	}
	return fmt.Sprintf("%s:%d", a.Path, a.Line)
}

type CommentParent struct {
	ID int `json:"id"`
}

// CommentRequest is the payload for creating a pull request comment. Anchor
// and Parent are mutually exclusive: a reply inherits its parent's anchor.
type CommentRequest struct {
	Text     string         `json:"text"`
	Parent   *CommentParent `json:"parent,omitempty"`
	Anchor   *CommentAnchor `json:"anchor,omitempty"`
	Severity string         `json:"severity,omitempty"`
	State    string         `json:"state,omitempty"`
}

// CreatePullRequestComment posts a comment. Depending on the request it lands
// as a general comment, a file comment, a line comment or a reply.
func (c *BitbucketClient) CreatePullRequestComment(ctx context.Context, project, repo string, prID int, req CommentRequest) (*Comment, error) {
	if req.Text == "" {
		return nil, fmt.Errorf("comment text required")
	}
	if req.Parent != nil && req.Anchor != nil {
		return nil, fmt.Errorf("a reply cannot carry its own anchor")
	}

	path := fmt.Sprintf("/rest/api/1.0/projects/%s/repos/%s/pull-requests/%d/comments", project, repo, prID)

	var comment Comment
	if err := c.Post(ctx, path, req, &comment); err != nil {
		return nil, err
	}

	return &comment, nil
}

// GetPullRequestCommentsForPath returns the comments anchored to one file.
// anchorState is ACTIVE, ORPHANED or ALL.
func (c *BitbucketClient) GetPullRequestCommentsForPath(ctx context.Context, project, repo string, prID int, filePath, anchorState string, limit int) ([]Comment, error) {
	params := url.Values{}
	params.Set("path", filePath)
	if anchorState != "" {
		params.Set("anchorState", anchorState)
	}
	if limit > 0 {
		params.Set("limit", strconv.Itoa(limit))
	}

	path := fmt.Sprintf("/rest/api/1.0/projects/%s/repos/%s/pull-requests/%d/comments", project, repo, prID)

	var response struct {
		Values []Comment `json:"values"`
	}

	if err := c.Get(ctx, path, params, &response); err != nil {
		return nil, err
	}

	return response.Values, nil
}

// GetPendingReview returns the authenticated user's unpublished review
// comments on a pull request. Unlike the activity feed, this endpoint carries
// each comment's anchor on the comment itself.
func (c *BitbucketClient) GetPendingReview(ctx context.Context, project, repo string, prID, limit int) ([]Comment, error) {
	params := url.Values{}
	if limit > 0 {
		params.Set("limit", strconv.Itoa(limit))
	}

	path := fmt.Sprintf("/rest/api/1.0/projects/%s/repos/%s/pull-requests/%d/review", project, repo, prID)

	var response struct {
		Values []Comment `json:"values"`
	}

	if err := c.Get(ctx, path, params, &response); err != nil {
		return nil, err
	}

	return response.Values, nil
}

// DiscardPendingReview drops every pending comment the authenticated user has
// drafted on a pull request.
func (c *BitbucketClient) DiscardPendingReview(ctx context.Context, project, repo string, prID int) error {
	path := fmt.Sprintf("/rest/api/1.0/projects/%s/repos/%s/pull-requests/%d/review", project, repo, prID)
	return c.Delete(ctx, path)
}
