package api

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"
)

func newTestJiraClient(server *httptest.Server) *JiraClient {
	client := NewJiraClient(server.URL, "tester", "token")
	client.HTTPClient = server.Client()
	return client
}

func TestCreateIssue(t *testing.T) {
	var gotPath, gotMethod, gotAuth string
	var gotBody map[string]interface{}

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotPath = r.URL.Path
		gotMethod = r.Method
		gotAuth = r.Header.Get("Authorization")
		if err := json.NewDecoder(r.Body).Decode(&gotBody); err != nil {
			t.Fatalf("decoding request body: %v", err)
		}
		w.WriteHeader(http.StatusCreated)
		_, _ = w.Write([]byte(`{"id":"100001","key":"MYPROJ-1234","self":"https://jira.example.com/rest/api/2/issue/100001"}`))
	}))
	defer server.Close()

	client := newTestJiraClient(server)

	fields := map[string]interface{}{
		"project":   map[string]string{"key": "MYPROJ"},
		"issuetype": map[string]string{"name": "Story"},
		"summary":   "test story",
	}

	created, err := client.CreateIssue(context.Background(), fields)
	if err != nil {
		t.Fatalf("CreateIssue returned error: %v", err)
	}

	if gotMethod != "POST" {
		t.Errorf("method = %q, want POST", gotMethod)
	}
	if gotPath != "/rest/api/2/issue" {
		t.Errorf("path = %q, want /rest/api/2/issue", gotPath)
	}
	if gotAuth != "Bearer token" {
		t.Errorf("auth = %q, want Bearer token", gotAuth)
	}
	if created.Key != "MYPROJ-1234" {
		t.Errorf("key = %q, want MYPROJ-1234", created.Key)
	}
	if created.ID != "100001" {
		t.Errorf("id = %q, want 100001", created.ID)
	}

	sentFields, ok := gotBody["fields"].(map[string]interface{})
	if !ok {
		t.Fatalf("request body missing fields object: %v", gotBody)
	}
	if sentFields["summary"] != "test story" {
		t.Errorf("sent summary = %v, want test story", sentFields["summary"])
	}
}

func TestCreateIssueServerError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusBadRequest)
		_, _ = w.Write([]byte(`{"errorMessages":["Issue type is required"],"errors":{}}`))
	}))
	defer server.Close()

	client := newTestJiraClient(server)

	_, err := client.CreateIssue(context.Background(), map[string]interface{}{})
	if err == nil {
		t.Fatal("expected error, got nil")
	}

	var unexpected *ErrUnexpectedResponse
	if !errors.As(err, &unexpected) {
		t.Fatalf("error type = %T, want *ErrUnexpectedResponse", err)
	}
	if unexpected.StatusCode != http.StatusBadRequest {
		t.Errorf("status = %d, want 400", unexpected.StatusCode)
	}
}

func TestGetFields(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/rest/api/2/field" {
			t.Errorf("path = %q, want /rest/api/2/field", r.URL.Path)
		}
		_, _ = w.Write([]byte(`[
			{"id":"summary","name":"Summary","custom":false,"schema":{"type":"string"}},
			{"id":"customfield_10501","name":"Epic Link","custom":true,"schema":{"type":"any","custom":"com.pyxis.greenhopper.jira:gh-epic-link"}},
			{"id":"customfield_10401","name":"Sprint","custom":true,"schema":{"type":"array","custom":"com.pyxis.greenhopper.jira:gh-sprint"}}
		]`))
	}))
	defer server.Close()

	client := newTestJiraClient(server)

	fields, err := client.GetFields(context.Background())
	if err != nil {
		t.Fatalf("GetFields returned error: %v", err)
	}
	if len(fields) != 3 {
		t.Fatalf("len(fields) = %d, want 3", len(fields))
	}
	if fields[1].ID != "customfield_10501" || fields[1].Schema.Custom != "com.pyxis.greenhopper.jira:gh-epic-link" {
		t.Errorf("epic field parsed wrong: %+v", fields[1])
	}
}
