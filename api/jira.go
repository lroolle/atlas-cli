package api

import (
	"fmt"
	"net/url"
	"strconv"
)

type JiraClient struct {
	*Client
}

func NewJiraClient(baseURL, username, token string) *JiraClient {
	client := NewClient(baseURL, username, token)
	client.AuthType = "bearer"
	return &JiraClient{
		Client: client,
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

func (c *JiraClient) SearchIssues(jql string, maxResults int) ([]Issue, error) {
	params := url.Values{}
	params.Set("jql", jql)
	params.Set("maxResults", strconv.Itoa(maxResults))
	
	path := "/rest/api/2/search"
	
	var result SearchResult
	err := c.Get(path, params, &result)
	if err != nil {
		return nil, err
	}
	
	return result.Issues, nil
}

func (c *JiraClient) GetIssue(issueKey string) (*Issue, error) {
	path := fmt.Sprintf("/rest/api/2/issue/%s", issueKey)
	
	var issue Issue
	err := c.Get(path, nil, &issue)
	if err != nil {
		return nil, err
	}
	
	return &issue, nil
}

func (c *JiraClient) GetTransitions(issueKey string) ([]Transition, error) {
	path := fmt.Sprintf("/rest/api/2/issue/%s/transitions", issueKey)
	
	var response struct {
		Transitions []Transition `json:"transitions"`
	}
	
	err := c.Get(path, nil, &response)
	if err != nil {
		return nil, err
	}
	
	return response.Transitions, nil
}

func (c *JiraClient) TransitionIssue(issueKey string, transitionID string) error {
	path := fmt.Sprintf("/rest/api/2/issue/%s/transitions", issueKey)
	
	body := map[string]interface{}{
		"transition": map[string]string{
			"id": transitionID,
		},
	}
	
	return c.Post(path, body, nil)
}

func (c *JiraClient) AddComment(issueKey string, comment string) error {
	path := fmt.Sprintf("/rest/api/2/issue/%s/comment", issueKey)
	
	body := map[string]string{
		"body": comment,
	}
	
	return c.Post(path, body, nil)
}

func (c *JiraClient) UpdateIssue(issueKey string, fields map[string]interface{}) error {
	path := fmt.Sprintf("/rest/api/2/issue/%s", issueKey)
	
	body := map[string]interface{}{
		"fields": fields,
	}
	
	return c.Put(path, body, nil)
}