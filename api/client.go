package api

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"
)

type Client struct {
	BaseURL    string
	Username   string
	Token      string
	AuthType   string // "basic" or "bearer"
	HTTPClient *http.Client
}

func NewClient(baseURL, username, token string) *Client {
	return &Client{
		BaseURL:  strings.TrimRight(baseURL, "/"),
		Username: username,
		Token:    token,
		AuthType: "basic",
		HTTPClient: &http.Client{
			Timeout: 30 * time.Second,
		},
	}
}

func (c *Client) doRequest(ctx context.Context, method, path string, body io.Reader) (*http.Response, error) {
	url := c.BaseURL + path

	req, err := http.NewRequestWithContext(ctx, method, url, body)
	if err != nil {
		return nil, fmt.Errorf("creating request: %w", err)
	}

	if c.AuthType == "bearer" {
		req.Header.Set("Authorization", "Bearer "+c.Token)
	} else {
		auth := base64.StdEncoding.EncodeToString([]byte(c.Username + ":" + c.Token))
		req.Header.Set("Authorization", "Basic "+auth)
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "application/json")

	resp, err := c.HTTPClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("executing request: %w", err)
	}

	if resp.StatusCode >= 400 {
		return resp, formatUnexpectedResponse(resp)
	}

	return resp, nil
}

func formatUnexpectedResponse(resp *http.Response) *ErrUnexpectedResponse {
	var errResp ErrorResponse

	if resp.Body != nil {
		_ = json.NewDecoder(resp.Body).Decode(&errResp)
	}

	return &ErrUnexpectedResponse{
		Body:       errResp,
		Status:     resp.Status,
		StatusCode: resp.StatusCode,
	}
}

func (c *Client) Get(ctx context.Context, path string, params url.Values, result interface{}) error {
	if params != nil {
		path = path + "?" + params.Encode()
	}

	resp, err := c.doRequest(ctx, "GET", path, nil)
	if resp != nil {
		defer resp.Body.Close()
	}
	if err != nil {
		return err
	}

	if result != nil {
		return json.NewDecoder(resp.Body).Decode(result)
	}

	return nil
}

func (c *Client) Post(ctx context.Context, path string, body interface{}, result interface{}) error {
	var reader io.Reader
	if body != nil {
		jsonData, err := json.Marshal(body)
		if err != nil {
			return fmt.Errorf("marshaling body: %w", err)
		}
		reader = strings.NewReader(string(jsonData))
	}

	resp, err := c.doRequest(ctx, "POST", path, reader)
	if resp != nil {
		defer resp.Body.Close()
	}
	if err != nil {
		return err
	}

	if result != nil {
		return json.NewDecoder(resp.Body).Decode(result)
	}

	return nil
}

func (c *Client) Put(ctx context.Context, path string, body interface{}, result interface{}) error {
	var reader io.Reader
	if body != nil {
		jsonData, err := json.Marshal(body)
		if err != nil {
			return fmt.Errorf("marshaling body: %w", err)
		}
		reader = strings.NewReader(string(jsonData))
	}

	resp, err := c.doRequest(ctx, "PUT", path, reader)
	if resp != nil {
		defer resp.Body.Close()
	}
	if err != nil {
		return err
	}

	if result != nil {
		return json.NewDecoder(resp.Body).Decode(result)
	}

	return nil
}

func (c *Client) GetRaw(ctx context.Context, path string) ([]byte, error) {
	resp, err := c.doRequest(ctx, "GET", path, nil)
	if resp != nil {
		defer resp.Body.Close()
	}
	if err != nil {
		return nil, err
	}

	return io.ReadAll(resp.Body)
}