package api

import (
	"context"
	"fmt"
	"net/url"
	"strconv"
)

type JiraClient struct {
	*Client
	InstallationType InstallationType
}

func NewJiraClient(baseURL, username, token string) *JiraClient {
	client := NewClient(baseURL, username, token)
	client.AuthType = "bearer"
	return &JiraClient{
		Client:           client,
		InstallationType: InstallationTypeCloud,
	}
}

type Issue struct {
	ID     string `json:"id"`
	Key    string `json:"key"`
	Self   string `json:"self"`
	Fields IssueFields `json:"fields"`
}

type IssueFields struct {
	Summary     string       `json:"summary"`
	Description string       `json:"description"`
	Status      Status       `json:"status"`
	Priority    Priority     `json:"priority"`
	IssueType   IssueType    `json:"issuetype"`
	Assignee    *JiraUser    `json:"assignee"`
	Reporter    JiraUser     `json:"reporter"`
	Created     string       `json:"created"`
	Updated     string       `json:"updated"`
	Resolution  *Resolution  `json:"resolution"`
	Project     JiraProject  `json:"project"`
	IssueLinks  []IssueLink  `json:"issuelinks,omitempty"`
	Parent      *Issue       `json:"parent,omitempty"`
}

type JiraProject struct {
	Key  string `json:"key"`
	ID   string `json:"id"`
	Name string `json:"name"`
}

type JiraUser struct {
	Name         string `json:"name"`
	EmailAddress string `json:"emailAddress"`
	DisplayName  string `json:"displayName"`
	Active       bool   `json:"active"`
}

type Status struct {
	ID          string `json:"id"`
	Name        string `json:"name"`
	Description string `json:"description"`
	IconURL     string `json:"iconUrl"`
}

type Priority struct {
	ID      string `json:"id"`
	Name    string `json:"name"`
	IconURL string `json:"iconUrl"`
}

type IssueType struct {
	ID          string `json:"id"`
	Name        string `json:"name"`
	Description string `json:"description"`
	IconURL     string `json:"iconUrl"`
}

type Resolution struct {
	ID          string `json:"id"`
	Name        string `json:"name"`
	Description string `json:"description"`
}

type IssueLink struct {
	ID           string    `json:"id"`
	Type         LinkType  `json:"type"`
	OutwardIssue *Issue    `json:"outwardIssue,omitempty"`
	InwardIssue  *Issue    `json:"inwardIssue,omitempty"`
}

type LinkType struct {
	ID      string `json:"id"`
	Name    string `json:"name"`
	Inward  string `json:"inward"`
	Outward string `json:"outward"`
}

type Transition struct {
	ID   string `json:"id"`
	Name string `json:"name"`
	To   Status `json:"to"`
}

type SearchResult struct {
	StartAt    int     `json:"startAt"`
	MaxResults int     `json:"maxResults"`
	Total      int     `json:"total"`
	Issues     []Issue `json:"issues"`
}

func (c *JiraClient) SearchIssues(ctx context.Context, jql string, maxResults int) ([]Issue, error) {
	params := url.Values{}
	params.Set("jql", jql)
	params.Set("maxResults", strconv.Itoa(maxResults))

	path := "/rest/api/2/search"

	var result SearchResult
	err := c.Get(ctx, path, params, &result)
	if err != nil {
		return nil, err
	}

	return result.Issues, nil
}

func (c *JiraClient) GetIssue(ctx context.Context, issueKey string) (*Issue, error) {
	path := fmt.Sprintf("/rest/api/2/issue/%s", issueKey)

	var issue Issue
	err := c.Get(ctx, path, nil, &issue)
	if err != nil {
		return nil, err
	}

	return &issue, nil
}

func (c *JiraClient) GetTransitions(ctx context.Context, issueKey string) ([]Transition, error) {
	path := fmt.Sprintf("/rest/api/2/issue/%s/transitions", issueKey)

	var response struct {
		Transitions []Transition `json:"transitions"`
	}

	err := c.Get(ctx, path, nil, &response)
	if err != nil {
		return nil, err
	}

	return response.Transitions, nil
}

func (c *JiraClient) TransitionIssue(ctx context.Context, issueKey string, transitionID string) error {
	path := fmt.Sprintf("/rest/api/2/issue/%s/transitions", issueKey)

	body := map[string]interface{}{
		"transition": map[string]string{
			"id": transitionID,
		},
	}

	return c.Post(ctx, path, body, nil)
}

func (c *JiraClient) AddComment(ctx context.Context, issueKey string, comment string) error {
	path := fmt.Sprintf("/rest/api/2/issue/%s/comment", issueKey)

	body := map[string]string{
		"body": comment,
	}

	return c.Post(ctx, path, body, nil)
}

type JiraComment struct {
	ID      string   `json:"id"`
	Body    string   `json:"body"`
	Author  JiraUser `json:"author"`
	Created string   `json:"created"`
	Updated string   `json:"updated"`
}

type CommentsResponse struct {
	StartAt    int           `json:"startAt"`
	MaxResults int           `json:"maxResults"`
	Total      int           `json:"total"`
	Comments   []JiraComment `json:"comments"`
}

func (c *JiraClient) GetComments(ctx context.Context, issueKey string) ([]JiraComment, error) {
	path := fmt.Sprintf("/rest/api/2/issue/%s/comment", issueKey)

	var response CommentsResponse
	err := c.Get(ctx, path, nil, &response)
	if err != nil {
		return nil, err
	}

	return response.Comments, nil
}

type DevelopmentInfo struct {
	Detail []struct {
		PullRequests []struct {
			ID           string `json:"id"`
			Name         string `json:"name"`
			URL          string `json:"url"`
			Status       string `json:"status"`
			LastUpdate   string `json:"lastUpdate"`
			CommentCount int    `json:"commentCount"`
			Author       struct {
				Name   string `json:"name"`
				Avatar string `json:"avatar"`
			} `json:"author"`
			Source struct {
				Branch     string `json:"branch"`
				Repository struct {
					Name string `json:"name"`
					URL  string `json:"url"`
				} `json:"repository"`
			} `json:"source"`
			Destination struct {
				Branch     string `json:"branch"`
				Repository struct {
					Name string `json:"name"`
					URL  string `json:"url"`
				} `json:"repository"`
			} `json:"destination"`
			Reviewers []struct {
				Name     string `json:"name"`
				Avatar   string `json:"avatar"`
				Approved bool   `json:"approved"`
			} `json:"reviewers"`
		} `json:"pullRequests"`
		Repositories []interface{} `json:"repositories"`
	} `json:"detail"`
	Errors []interface{} `json:"errors"`
}

func (c *JiraClient) GetDevelopmentInfo(ctx context.Context, issueKey string) (*DevelopmentInfo, error) {
	path := fmt.Sprintf("/rest/dev-status/1.0/issue/detail?issueId=%s&applicationType=stash&dataType=pullrequest", issueKey)

	var devInfo DevelopmentInfo
	err := c.Get(ctx, path, nil, &devInfo)
	if err != nil {
		return nil, err
	}

	return &devInfo, nil
}

func (c *JiraClient) GetRepositoryInfo(ctx context.Context, issueKey string) (*DevelopmentInfo, error) {
	path := fmt.Sprintf("/rest/dev-status/1.0/issue/detail?issueId=%s&applicationType=stash&dataType=repository", issueKey)

	var devInfo DevelopmentInfo
	err := c.Get(ctx, path, nil, &devInfo)
	if err != nil {
		return nil, err
	}

	return &devInfo, nil
}

func (c *JiraClient) UpdateIssue(ctx context.Context, issueKey string, fields map[string]interface{}) error {
	path := fmt.Sprintf("/rest/api/2/issue/%s", issueKey)

	body := map[string]interface{}{
		"fields": fields,
	}

	return c.Put(ctx, path, body, nil)
}