package api

import (
	"fmt"
	"net/url"
)

type ConfluenceClient struct {
	*Client
}

func NewConfluenceClient(baseURL, username, token string) *ConfluenceClient {
	client := NewClient(baseURL, username, token)
	// Confluence uses Bearer auth
	client.AuthType = "bearer"
	return &ConfluenceClient{
		Client: client,
	}
}

type Space struct {
	ID     int         `json:"id"`
	Key    string      `json:"key"`
	Name   string      `json:"name"`
	Type   string      `json:"type"`
	Status string      `json:"status"`
	Links  interface{} `json:"_links"`
}

type Content struct {
	ID       string              `json:"id"`
	Type     string              `json:"type"`
	Status   string              `json:"status"`
	Title    string              `json:"title"`
	Space    Space               `json:"space,omitempty"`
	Body     ContentBody         `json:"body,omitempty"`
	Version  ContentVersion      `json:"version,omitempty"`
	Links    map[string]string   `json:"_links,omitempty"`
	Metadata ContentMetadata     `json:"metadata,omitempty"`
}

type ContentBody struct {
	Storage ContentBodyStorage `json:"storage,omitempty"`
	View    ContentBodyView    `json:"view,omitempty"`
}

type ContentBodyStorage struct {
	Value          string `json:"value"`
	Representation string `json:"representation"`
}

type ContentBodyView struct {
	Value          string `json:"value"`
	Representation string `json:"representation"`
}

type ContentVersion struct {
	Number    int    `json:"number"`
	Message   string `json:"message,omitempty"`
	MinorEdit bool   `json:"minorEdit,omitempty"`
}

type ContentMetadata struct {
	Labels []Label `json:"labels,omitempty"`
}

type Label struct {
	ID     string `json:"id,omitempty"`
	Name   string `json:"name"`
	Prefix string `json:"prefix,omitempty"`
}

type ConfluencePagedResponse struct {
	Results []interface{} `json:"results"`
	Start   int           `json:"start"`
	Limit   int           `json:"limit"`
	Size    int           `json:"size"`
	Links   interface{}   `json:"_links"`
}

// GetSpaces returns a list of spaces
func (c *ConfluenceClient) GetSpaces(limit int) ([]Space, error) {
	params := url.Values{}
	params.Set("limit", fmt.Sprintf("%d", limit))
	
	path := "/rest/api/space"
	
	var response struct {
		Results []Space `json:"results"`
	}
	
	err := c.Get(path, params, &response)
	if err != nil {
		return nil, err
	}
	
	return response.Results, nil
}

// GetContent returns content in a space
func (c *ConfluenceClient) GetContent(spaceKey string, contentType string, limit int) ([]Content, error) {
	params := url.Values{}
	params.Set("spaceKey", spaceKey)
	params.Set("type", contentType)
	params.Set("limit", fmt.Sprintf("%d", limit))
	params.Set("expand", "body.view,version")
	
	path := "/rest/api/content"
	
	var response struct {
		Results []Content `json:"results"`
	}
	
	err := c.Get(path, params, &response)
	if err != nil {
		return nil, err
	}
	
	return response.Results, nil
}

// GetPage returns a specific page by ID
func (c *ConfluenceClient) GetPage(pageID string) (*Content, error) {
	params := url.Values{}
	params.Set("expand", "body.storage,body.view,version,space")
	
	path := fmt.Sprintf("/rest/api/content/%s", pageID)
	
	var content Content
	err := c.Get(path, params, &content)
	if err != nil {
		return nil, err
	}
	
	return &content, nil
}

// SearchContent searches for content
func (c *ConfluenceClient) SearchContent(query string, limit int) ([]Content, error) {
	params := url.Values{}
	params.Set("cql", query)
	params.Set("limit", fmt.Sprintf("%d", limit))
	params.Set("expand", "content.version,content.space")
	
	path := "/rest/api/content/search"
	
	var response struct {
		Results []struct {
			Content Content `json:"content"`
		} `json:"results"`
	}
	
	err := c.Get(path, params, &response)
	if err != nil {
		return nil, err
	}
	
	contents := make([]Content, len(response.Results))
	for i, r := range response.Results {
		contents[i] = r.Content
	}
	
	return contents, nil
}

// CreatePage creates a new page
func (c *ConfluenceClient) CreatePage(spaceKey, title, content string, parentID string) (*Content, error) {
	path := "/rest/api/content"
	
	body := map[string]interface{}{
		"type":  "page",
		"title": title,
		"space": map[string]string{
			"key": spaceKey,
		},
		"body": map[string]interface{}{
			"storage": map[string]string{
				"value":          content,
				"representation": "storage",
			},
		},
	}
	
	if parentID != "" {
		body["ancestors"] = []map[string]string{
			{"id": parentID},
		}
	}
	
	var page Content
	err := c.Post(path, body, &page)
	if err != nil {
		return nil, err
	}
	
	return &page, nil
}

// UpdatePage updates an existing page
func (c *ConfluenceClient) UpdatePage(pageID string, title, content string, version int) (*Content, error) {
	path := fmt.Sprintf("/rest/api/content/%s", pageID)
	
	body := map[string]interface{}{
		"type":  "page",
		"title": title,
		"body": map[string]interface{}{
			"storage": map[string]string{
				"value":          content,
				"representation": "storage",
			},
		},
		"version": map[string]interface{}{
			"number": version + 1,
		},
	}
	
	var page Content
	err := c.Put(path, body, &page)
	if err != nil {
		return nil, err
	}
	
	return &page, nil
}