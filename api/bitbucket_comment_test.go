package api

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestCreatePullRequestCommentSendsAnchor(t *testing.T) {
	var gotPath string
	var gotBody map[string]interface{}

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotPath = r.URL.Path
		if err := json.NewDecoder(r.Body).Decode(&gotBody); err != nil {
			t.Fatalf("decoding request body: %v", err)
		}
		w.WriteHeader(http.StatusCreated)
		_, _ = w.Write([]byte(`{"id":331,"version":0,"text":"this allocation leaks"}`))
	}))
	defer server.Close()

	client := NewBitbucketClient(server.URL, "tester", "token")
	client.HTTPClient = server.Client()

	comment, err := client.CreatePullRequestComment(context.Background(), "MYPROJ", "myrepo", 140, CommentRequest{
		Text:     "this allocation leaks",
		Severity: SeverityBlocker,
		Anchor: &CommentAnchor{
			DiffType: DiffTypeEffective,
			LineType: SegmentAdded,
			FileType: FileTypeTo,
			Line:     543,
			Path:     "src/app.js",
			FromHash: "aaa111base",
			ToHash:   "bbb222head",
		},
	})
	if err != nil {
		t.Fatalf("CreatePullRequestComment returned error: %v", err)
	}
	if comment.ID != 331 {
		t.Errorf("comment ID = %d, want 331", comment.ID)
	}

	wantPath := "/rest/api/1.0/projects/MYPROJ/repos/myrepo/pull-requests/140/comments"
	if gotPath != wantPath {
		t.Errorf("path = %q, want %q", gotPath, wantPath)
	}
	if gotBody["severity"] != SeverityBlocker {
		t.Errorf("severity = %v, want %s", gotBody["severity"], SeverityBlocker)
	}

	anchor, ok := gotBody["anchor"].(map[string]interface{})
	if !ok {
		t.Fatalf("anchor missing from request body: %v", gotBody)
	}
	if anchor["line"] != float64(543) || anchor["lineType"] != SegmentAdded || anchor["fileType"] != FileTypeTo {
		t.Errorf("anchor = %v, want line 543 ADDED/TO", anchor)
	}
	if _, ok := anchor["srcPath"]; ok {
		t.Error("srcPath sent for a file that was not renamed")
	}
	if _, ok := anchor["orphaned"]; ok {
		t.Error("orphaned is server state and must not be sent")
	}
}

func TestCreatePullRequestCommentReply(t *testing.T) {
	var gotBody map[string]interface{}

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_ = json.NewDecoder(r.Body).Decode(&gotBody)
		w.WriteHeader(http.StatusCreated)
		_, _ = w.Write([]byte(`{"id":53240}`))
	}))
	defer server.Close()

	client := NewBitbucketClient(server.URL, "tester", "token")
	client.HTTPClient = server.Client()

	_, err := client.CreatePullRequestComment(context.Background(), "MYPROJ", "myrepo", 140, CommentRequest{
		Text:   "fixed in a1b2c3d",
		Parent: &CommentParent{ID: 331},
	})
	if err != nil {
		t.Fatalf("CreatePullRequestComment returned error: %v", err)
	}

	parent, ok := gotBody["parent"].(map[string]interface{})
	if !ok || parent["id"] != float64(331) {
		t.Errorf("parent = %v, want id 331", gotBody["parent"])
	}
	if _, ok := gotBody["anchor"]; ok {
		t.Error("reply must not carry an anchor")
	}
}

func TestCreatePullRequestCommentRejectsAnchoredReply(t *testing.T) {
	client := NewBitbucketClient("https://git.example.com", "tester", "token")

	_, err := client.CreatePullRequestComment(context.Background(), "MYPROJ", "myrepo", 140, CommentRequest{
		Text:   "both at once",
		Parent: &CommentParent{ID: 1},
		Anchor: &CommentAnchor{Path: "src/app.js"},
	})
	if err == nil {
		t.Fatal("expected an error for a reply carrying its own anchor")
	}
}

func TestCommentAnchorLocation(t *testing.T) {
	tests := []struct {
		name   string
		anchor *CommentAnchor
		want   string
	}{
		{name: "nil anchor", anchor: nil, want: ""},
		{name: "general comment", anchor: &CommentAnchor{}, want: ""},
		{name: "file comment", anchor: &CommentAnchor{Path: "src/app.js"}, want: "src/app.js"},
		{name: "line comment", anchor: &CommentAnchor{Path: "src/app.js", Line: 543}, want: "src/app.js:543"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.anchor.Location(); got != tt.want {
				t.Errorf("Location() = %q, want %q", got, tt.want)
			}
		})
	}
}
