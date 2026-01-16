package api

import (
	"context"
	"fmt"
	"io"
	"net/url"
	"strconv"
	"strings"
)

type BitbucketClient struct {
	*Client
	Username string
}

func NewBitbucketClient(baseURL, username, token string) *BitbucketClient {
	return &BitbucketClient{
		Client:   NewClient(baseURL, username, token),
		Username: username,
	}
}

type PullRequest struct {
	ID          int    `json:"id"`
	Version     int    `json:"version"`
	Title       string `json:"title"`
	Description string `json:"description"`
	State       string `json:"state"`
	Open        bool   `json:"open"`
	Closed      bool   `json:"closed"`
	CreatedDate int64  `json:"createdDate"`
	UpdatedDate int64  `json:"updatedDate"`
	FromRef     Ref    `json:"fromRef"`
	ToRef       Ref    `json:"toRef"`
	Author      struct {
		User     User   `json:"user"`
		Role     string `json:"role"`
		Approved bool   `json:"approved"`
		Status   string `json:"status"`
	} `json:"author"`
	Reviewers []struct {
		User     User   `json:"user"`
		Role     string `json:"role"`
		Approved bool   `json:"approved"`
		Status   string `json:"status"`
	} `json:"reviewers"`
	Links struct {
		Self []Link `json:"self"`
	} `json:"links"`
}

type Ref struct {
	ID           string     `json:"id"`
	DisplayID    string     `json:"displayId"`
	LatestCommit string     `json:"latestCommit"`
	Repository   Repository `json:"repository"`
}

type Repository struct {
	Slug    string  `json:"slug"`
	ID      int     `json:"id"`
	Name    string  `json:"name"`
	Project Project `json:"project"`
}

type Project struct {
	Key  string `json:"key"`
	ID   int    `json:"id"`
	Name string `json:"name"`
}

type User struct {
	Name         string `json:"name"`
	EmailAddress string `json:"emailAddress"`
	ID           int    `json:"id"`
	DisplayName  string `json:"displayName"`
	Active       bool   `json:"active"`
	Slug         string `json:"slug"`
	Type         string `json:"type"`
}

type Link struct {
	Href string `json:"href"`
}

type PagedResponse struct {
	Size          int         `json:"size"`
	Limit         int         `json:"limit"`
	Start         int         `json:"start"`
	IsLastPage    bool        `json:"isLastPage"`
	NextPageStart int         `json:"nextPageStart"`
	Values        interface{} `json:"values"`
}

type Comment struct {
	ID          int    `json:"id"`
	Version     int    `json:"version"`
	Text        string `json:"text"`
	Author      User   `json:"author"`
	CreatedDate int64  `json:"createdDate"`
	UpdatedDate int64  `json:"updatedDate"`
}

type Commit struct {
	ID                 string `json:"id"`
	DisplayID          string `json:"displayId"`
	Author             User   `json:"author"`
	AuthorTimestamp    int64  `json:"authorTimestamp"`
	Committer          User   `json:"committer"`
	CommitterTimestamp int64  `json:"committerTimestamp"`
	Message            string `json:"message"`
	Parents            []struct {
		ID        string `json:"id"`
		DisplayID string `json:"displayId"`
	} `json:"parents"`
}

func (c *BitbucketClient) ListPullRequests(ctx context.Context, project, repo string, state string, limit int) ([]PullRequest, error) {
	params := url.Values{}
	if state != "" {
		params.Set("state", state)
	}
	params.Set("limit", strconv.Itoa(limit))

	path := fmt.Sprintf("/rest/api/1.0/projects/%s/repos/%s/pull-requests", project, repo)

	var response struct {
		Values []PullRequest `json:"values"`
	}

	err := c.Get(ctx, path, params, &response)
	if err != nil {
		return nil, err
	}

	return response.Values, nil
}

func (c *BitbucketClient) GetPullRequest(ctx context.Context, project, repo string, prID int) (*PullRequest, error) {
	path := fmt.Sprintf("/rest/api/1.0/projects/%s/repos/%s/pull-requests/%d", project, repo, prID)

	var pr PullRequest
	err := c.Get(ctx, path, nil, &pr)
	if err != nil {
		return nil, err
	}

	return &pr, nil
}

func (c *BitbucketClient) GetPullRequestDiff(ctx context.Context, project, repo string, prID int) (string, error) {
	path := fmt.Sprintf("/rest/api/1.0/projects/%s/repos/%s/pull-requests/%d/diff", project, repo, prID)

	resp, err := c.doRequestWithAccept(ctx, "GET", path, nil, "text/plain")
	if resp != nil {
		defer resp.Body.Close()
	}
	if err != nil {
		return "", err
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", err
	}

	return string(body), nil
}

func (c *BitbucketClient) AddPullRequestComment(ctx context.Context, project, repo string, prID int, text string) (*Comment, error) {
	path := fmt.Sprintf("/rest/api/1.0/projects/%s/repos/%s/pull-requests/%d/comments", project, repo, prID)

	body := map[string]string{
		"text": text,
	}

	var comment Comment
	err := c.Post(ctx, path, body, &comment)
	if err != nil {
		return nil, err
	}

	return &comment, nil
}

func (c *BitbucketClient) ListCommits(ctx context.Context, project, repo string, limit int) ([]Commit, error) {
	params := url.Values{}
	params.Set("limit", strconv.Itoa(limit))

	path := fmt.Sprintf("/rest/api/1.0/projects/%s/repos/%s/commits", project, repo)

	var response struct {
		Values []Commit `json:"values"`
	}

	err := c.Get(ctx, path, params, &response)
	if err != nil {
		return nil, err
	}

	return response.Values, nil
}

func (c *BitbucketClient) MergePullRequest(ctx context.Context, project, repo string, prID int, version int) error {
	path := fmt.Sprintf("/rest/api/1.0/projects/%s/repos/%s/pull-requests/%d/merge?version=%d", project, repo, prID, version)
	return c.Post(ctx, path, nil, nil)
}

func (c *BitbucketClient) CreatePullRequest(ctx context.Context, project, repo string, title, description, fromBranch, toBranch string, reviewers []string) (*PullRequest, error) {
	path := fmt.Sprintf("/rest/api/1.0/projects/%s/repos/%s/pull-requests", project, repo)

	body := map[string]interface{}{
		"title":       title,
		"description": description,
		"fromRef": map[string]interface{}{
			"id": fmt.Sprintf("refs/heads/%s", fromBranch),
			"repository": map[string]interface{}{
				"slug": repo,
				"project": map[string]string{
					"key": project,
				},
			},
		},
		"toRef": map[string]interface{}{
			"id": fmt.Sprintf("refs/heads/%s", toBranch),
			"repository": map[string]interface{}{
				"slug": repo,
				"project": map[string]string{
					"key": project,
				},
			},
		},
	}

	if len(reviewers) > 0 {
		var reviewerList []map[string]interface{}
		for _, r := range reviewers {
			reviewerList = append(reviewerList, map[string]interface{}{
				"user": map[string]string{"name": r},
			})
		}
		body["reviewers"] = reviewerList
	}

	var pr PullRequest
	err := c.Post(ctx, path, body, &pr)
	if err != nil {
		return nil, err
	}

	return &pr, nil
}

type Change struct {
	ContentID        string `json:"contentId"`
	FromHash         string `json:"fromHash"`
	ToHash           string `json:"toHash"`
	Path             Path   `json:"path"`
	Executable       bool   `json:"executable"`
	PercentUnchanged int    `json:"percentUnchanged"`
	Type             string `json:"type"`
	NodeType         string `json:"nodeType"`
	SrcPath          *Path  `json:"srcPath,omitempty"`
}

type Path struct {
	Components []string `json:"components"`
	Parent     string   `json:"parent"`
	Name       string   `json:"name"`
	ToString   string   `json:"toString"`
}

type MergeResult struct {
	CanMerge   bool   `json:"canMerge"`
	Conflicted bool   `json:"conflicted"`
	Outcome    string `json:"outcome"`
	Vetoes     []Veto `json:"vetoes"`
}

type Veto struct {
	SummaryMessage  string `json:"summaryMessage"`
	DetailedMessage string `json:"detailedMessage"`
}

type Activity struct {
	ID          int      `json:"id"`
	CreatedDate int64    `json:"createdDate"`
	User        User     `json:"user"`
	Action      string   `json:"action"`
	Comment     *Comment `json:"comment,omitempty"`
}

type UpdatePROptions struct {
	Title       string
	Description string
	ToRef       string
	Reviewers   []string
	Version     int
}

func (c *BitbucketClient) ApprovePullRequest(ctx context.Context, project, repo string, prID int) error {
	path := fmt.Sprintf("/rest/api/1.0/projects/%s/repos/%s/pull-requests/%d/approve", project, repo, prID)
	return c.Post(ctx, path, nil, nil)
}

func (c *BitbucketClient) UnapprovePullRequest(ctx context.Context, project, repo string, prID int) error {
	path := fmt.Sprintf("/rest/api/1.0/projects/%s/repos/%s/pull-requests/%d/approve", project, repo, prID)
	return c.Delete(ctx, path)
}

func (c *BitbucketClient) SetReviewerStatus(ctx context.Context, project, repo string, prID int, status string) error {
	path := fmt.Sprintf("/rest/api/1.0/projects/%s/repos/%s/pull-requests/%d/participants/%s", project, repo, prID, c.Username)
	body := map[string]interface{}{
		"status": status,
	}
	return c.Put(ctx, path, body, nil)
}

func (c *BitbucketClient) DeclinePullRequest(ctx context.Context, project, repo string, prID, version int) error {
	path := fmt.Sprintf("/rest/api/1.0/projects/%s/repos/%s/pull-requests/%d/decline?version=%d", project, repo, prID, version)
	return c.Post(ctx, path, nil, nil)
}

func (c *BitbucketClient) ReopenPullRequest(ctx context.Context, project, repo string, prID, version int) error {
	path := fmt.Sprintf("/rest/api/1.0/projects/%s/repos/%s/pull-requests/%d/reopen?version=%d", project, repo, prID, version)
	return c.Post(ctx, path, nil, nil)
}

func (c *BitbucketClient) UpdatePullRequest(ctx context.Context, project, repo string, prID int, opts UpdatePROptions) (*PullRequest, error) {
	path := fmt.Sprintf("/rest/api/1.0/projects/%s/repos/%s/pull-requests/%d", project, repo, prID)

	body := map[string]interface{}{
		"version": opts.Version,
	}

	if opts.Title != "" {
		body["title"] = opts.Title
	}
	if opts.Description != "" {
		body["description"] = opts.Description
	}
	if opts.ToRef != "" {
		body["toRef"] = map[string]interface{}{
			"id": fmt.Sprintf("refs/heads/%s", opts.ToRef),
		}
	}
	if len(opts.Reviewers) > 0 {
		var reviewerList []map[string]interface{}
		for _, r := range opts.Reviewers {
			reviewerList = append(reviewerList, map[string]interface{}{
				"user": map[string]string{"name": r},
			})
		}
		body["reviewers"] = reviewerList
	}

	var pr PullRequest
	err := c.Put(ctx, path, body, &pr)
	if err != nil {
		return nil, err
	}

	return &pr, nil
}

func (c *BitbucketClient) GetPullRequestCommits(ctx context.Context, project, repo string, prID int, limit int) ([]Commit, error) {
	params := url.Values{}
	params.Set("limit", strconv.Itoa(limit))

	path := fmt.Sprintf("/rest/api/1.0/projects/%s/repos/%s/pull-requests/%d/commits", project, repo, prID)

	var response struct {
		Values []Commit `json:"values"`
	}

	err := c.Get(ctx, path, params, &response)
	if err != nil {
		return nil, err
	}

	return response.Values, nil
}

func (c *BitbucketClient) GetPullRequestChanges(ctx context.Context, project, repo string, prID int, limit int) ([]Change, error) {
	params := url.Values{}
	params.Set("limit", strconv.Itoa(limit))

	path := fmt.Sprintf("/rest/api/1.0/projects/%s/repos/%s/pull-requests/%d/changes", project, repo, prID)

	var response struct {
		Values []Change `json:"values"`
	}

	err := c.Get(ctx, path, params, &response)
	if err != nil {
		return nil, err
	}

	return response.Values, nil
}

func (c *BitbucketClient) CanMerge(ctx context.Context, project, repo string, prID int) (*MergeResult, error) {
	path := fmt.Sprintf("/rest/api/1.0/projects/%s/repos/%s/pull-requests/%d/merge", project, repo, prID)

	var result MergeResult
	err := c.Get(ctx, path, nil, &result)
	if err != nil {
		return nil, err
	}

	return &result, nil
}

func (c *BitbucketClient) RebasePullRequest(ctx context.Context, project, repo string, prID, version int) error {
	path := fmt.Sprintf("/rest/api/1.0/projects/%s/repos/%s/pull-requests/%d/rebase?version=%d", project, repo, prID, version)
	return c.Post(ctx, path, nil, nil)
}

func (c *BitbucketClient) GetPullRequestActivity(ctx context.Context, project, repo string, prID int, limit int) ([]Activity, error) {
	params := url.Values{}
	params.Set("limit", strconv.Itoa(limit))

	path := fmt.Sprintf("/rest/api/1.0/projects/%s/repos/%s/pull-requests/%d/activities", project, repo, prID)

	var response struct {
		Values []Activity `json:"values"`
	}

	err := c.Get(ctx, path, params, &response)
	if err != nil {
		return nil, err
	}

	return response.Values, nil
}

func (c *BitbucketClient) AddReviewer(ctx context.Context, project, repo string, prID int, username string) error {
	path := fmt.Sprintf("/rest/api/1.0/projects/%s/repos/%s/pull-requests/%d/participants", project, repo, prID)
	body := map[string]interface{}{
		"user": map[string]string{"name": username},
		"role": "REVIEWER",
	}
	return c.Post(ctx, path, body, nil)
}

func (c *BitbucketClient) RemoveReviewer(ctx context.Context, project, repo string, prID int, username string) error {
	path := fmt.Sprintf("/rest/api/1.0/projects/%s/repos/%s/pull-requests/%d/participants/%s", project, repo, prID, username)
	return c.Delete(ctx, path)
}

func (c *BitbucketClient) GetCurrentBranch(ctx context.Context, project, repo string) (string, error) {
	path := fmt.Sprintf("/rest/api/1.0/projects/%s/repos/%s/default-branch", project, repo)

	var response struct {
		DisplayID string `json:"displayId"`
	}

	err := c.Get(ctx, path, nil, &response)
	if err != nil {
		return "", err
	}

	return response.DisplayID, nil
}

func (c *BitbucketClient) GetDefaultBranch(ctx context.Context, project, repo string) (string, error) {
	return c.GetCurrentBranch(ctx, project, repo)
}

type branchDeleteRequest struct {
	Name     string `json:"name"`
	EndPoint string `json:"endPoint,omitempty"`
}

func (c *BitbucketClient) DeleteBranch(ctx context.Context, project, repo, name string) error {
	if name == "" {
		return fmt.Errorf("branch name required")
	}
	if !strings.HasPrefix(name, "refs/") {
		name = fmt.Sprintf("refs/heads/%s", name)
	}

	path := fmt.Sprintf("/rest/branch-utils/latest/projects/%s/repos/%s/branches", project, repo)
	return c.DeleteWithBody(ctx, path, branchDeleteRequest{Name: name})
}
