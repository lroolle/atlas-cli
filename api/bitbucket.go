package api

import (
	"fmt"
	"io"
	"net/url"
	"strconv"
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
		User User `json:"user"`
		Role string `json:"role"`
		Approved bool `json:"approved"`
		Status string `json:"status"`
	} `json:"author"`
	Reviewers []struct {
		User User `json:"user"`
		Role string `json:"role"`
		Approved bool `json:"approved"`
		Status string `json:"status"`
	} `json:"reviewers"`
	Links struct {
		Self []Link `json:"self"`
	} `json:"links"`
}

type Ref struct {
	ID           string `json:"id"`
	DisplayID    string `json:"displayId"`
	LatestCommit string `json:"latestCommit"`
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
	ID               string `json:"id"`
	DisplayID        string `json:"displayId"`
	Author           User   `json:"author"`
	AuthorTimestamp  int64  `json:"authorTimestamp"`
	Committer        User   `json:"committer"`
	CommitterTimestamp int64 `json:"committerTimestamp"`
	Message          string `json:"message"`
	Parents          []struct {
		ID        string `json:"id"`
		DisplayID string `json:"displayId"`
	} `json:"parents"`
}

func (c *BitbucketClient) ListPullRequests(project, repo string, state string, limit int) ([]PullRequest, error) {
	params := url.Values{}
	if state != "" {
		params.Set("state", state)
	}
	params.Set("limit", strconv.Itoa(limit))

	path := fmt.Sprintf("/rest/api/1.0/projects/%s/repos/%s/pull-requests", project, repo)
	
	var response struct {
		Values []PullRequest `json:"values"`
	}
	
	err := c.Get(path, params, &response)
	if err != nil {
		return nil, err
	}
	
	return response.Values, nil
}

func (c *BitbucketClient) GetPullRequest(project, repo string, prID int) (*PullRequest, error) {
	path := fmt.Sprintf("/rest/api/1.0/projects/%s/repos/%s/pull-requests/%d", project, repo, prID)
	
	var pr PullRequest
	err := c.Get(path, nil, &pr)
	if err != nil {
		return nil, err
	}
	
	return &pr, nil
}

func (c *BitbucketClient) GetPullRequestDiff(project, repo string, prID int) (string, error) {
	path := fmt.Sprintf("/rest/api/1.0/projects/%s/repos/%s/pull-requests/%d/diff", project, repo, prID)
	
	resp, err := c.doRequest("GET", path, nil)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()
	
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", err
	}
	
	return string(body), nil
}

func (c *BitbucketClient) AddPullRequestComment(project, repo string, prID int, text string) (*Comment, error) {
	path := fmt.Sprintf("/rest/api/1.0/projects/%s/repos/%s/pull-requests/%d/comments", project, repo, prID)
	
	body := map[string]string{
		"text": text,
	}
	
	var comment Comment
	err := c.Post(path, body, &comment)
	if err != nil {
		return nil, err
	}
	
	return &comment, nil
}

func (c *BitbucketClient) ListCommits(project, repo string, limit int) ([]Commit, error) {
	params := url.Values{}
	params.Set("limit", strconv.Itoa(limit))
	
	path := fmt.Sprintf("/rest/api/1.0/projects/%s/repos/%s/commits", project, repo)
	
	var response struct {
		Values []Commit `json:"values"`
	}
	
	err := c.Get(path, params, &response)
	if err != nil {
		return nil, err
	}
	
	return response.Values, nil
}

func (c *BitbucketClient) MergePullRequest(project, repo string, prID int, version int) error {
	path := fmt.Sprintf("/rest/api/1.0/projects/%s/repos/%s/pull-requests/%d/merge", project, repo, prID)
	
	body := map[string]interface{}{
		"version": version,
	}
	
	return c.Post(path, body, nil)
}

func (c *BitbucketClient) CreatePullRequest(project, repo string, title, description, fromBranch, toBranch string) (*PullRequest, error) {
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
	
	var pr PullRequest
	err := c.Post(path, body, &pr)
	if err != nil {
		return nil, err
	}
	
	return &pr, nil
}