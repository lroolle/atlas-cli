package cmd

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/lroolle/atlas-cli/api"
)

func TestCommentSpecValidate(t *testing.T) {
	tests := []struct {
		name    string
		spec    commentSpec
		wantErr string
	}{
		{
			name: "line comment",
			spec: commentSpec{Body: "x", File: "a.js", Line: 10},
		},
		{
			name: "file comment",
			spec: commentSpec{Body: "x", File: "a.js"},
		},
		{
			name: "general comment",
			spec: commentSpec{Body: "x"},
		},
		{
			name:    "empty body",
			spec:    commentSpec{Body: "  "},
			wantErr: "body required",
		},
		{
			name:    "anchored reply",
			spec:    commentSpec{Body: "x", ReplyTo: 5, File: "a.js", Line: 10},
			wantErr: "inherits its parent's anchor",
		},
		{
			name:    "line without file",
			spec:    commentSpec{Body: "x", Line: 10},
			wantErr: "--line requires --file",
		},
		{
			name:    "unknown side",
			spec:    commentSpec{Body: "x", File: "a.js", Line: 10, Side: "middle"},
			wantErr: "invalid side",
		},
		{
			name:    "unknown line type",
			spec:    commentSpec{Body: "x", File: "a.js", Line: 10, LineType: "CHANGED"},
			wantErr: "invalid --line-type",
		},
		{
			name:    "unknown file type",
			spec:    commentSpec{Body: "x", File: "a.js", Line: 10, FileType: "BOTH"},
			wantErr: "invalid --file-type",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.spec.validate()
			if tt.wantErr == "" {
				if err != nil {
					t.Fatalf("validate() returned error: %v", err)
				}
				return
			}
			if err == nil {
				t.Fatalf("validate() = nil, want error containing %q", tt.wantErr)
			}
			if !strings.Contains(err.Error(), tt.wantErr) {
				t.Errorf("error = %q, want it to contain %q", err.Error(), tt.wantErr)
			}
		})
	}
}

func TestReadBatch(t *testing.T) {
	path := filepath.Join(t.TempDir(), "findings.json")
	content := `[
	  {"file": "src/app.js", "line": 543, "body": "this allocation leaks", "blocker": true},
	  {"file": "src/app.js", "line": 120, "side": "old", "body": "why drop this?"},
	  {"reply_to": 331, "body": "fixed"},
	  {"body": "overall NAK"}
	]`
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatalf("writing batch: %v", err)
	}

	specs, err := readBatch(path)
	if err != nil {
		t.Fatalf("readBatch returned error: %v", err)
	}
	if len(specs) != 4 {
		t.Fatalf("parsed %d specs, want 4", len(specs))
	}
	if !specs[0].Blocker || specs[0].Line != 543 {
		t.Errorf("first spec = %+v, want blocker on line 543", specs[0])
	}
	if specs[1].Side != "old" {
		t.Errorf("second spec side = %q, want old", specs[1].Side)
	}
	if specs[2].ReplyTo != 331 {
		t.Errorf("third spec reply_to = %d, want 331", specs[2].ReplyTo)
	}
}

func TestReadBatchRejectsInvalidEntry(t *testing.T) {
	path := filepath.Join(t.TempDir(), "findings.json")
	if err := os.WriteFile(path, []byte(`[{"body":"ok"},{"file":"a.js","line":3}]`), 0o644); err != nil {
		t.Fatalf("writing batch: %v", err)
	}

	_, err := readBatch(path)
	if err == nil || !strings.Contains(err.Error(), "batch entry 2") {
		t.Fatalf("error = %v, want it to name the bad entry", err)
	}
}

func TestReadBatchRejectsEmptyList(t *testing.T) {
	path := filepath.Join(t.TempDir(), "findings.json")
	if err := os.WriteFile(path, []byte(`[]`), 0o644); err != nil {
		t.Fatalf("writing batch: %v", err)
	}

	if _, err := readBatch(path); err == nil {
		t.Fatal("expected an error for an empty batch")
	}
}

func TestCollectSpecsBatch(t *testing.T) {
	path := filepath.Join(t.TempDir(), "findings.json")
	if err := os.WriteFile(path, []byte(`[{"body":"one"},{"body":"two","pending":false}]`), 0o644); err != nil {
		t.Fatalf("writing batch: %v", err)
	}

	cmd := prCommentCmd
	t.Cleanup(func() {
		_ = cmd.Flags().Set("pending", "false")
		cmd.Flags().Lookup("pending").Changed = false
		_ = cmd.Flags().Set("file", "")
		cmd.Flags().Lookup("file").Changed = false
	})

	// --pending is a property of the whole review, so it overrides the file.
	if err := cmd.Flags().Set("pending", "true"); err != nil {
		t.Fatalf("setting flag: %v", err)
	}

	specs, err := collectSpecs(cmd, "", path)
	if err != nil {
		t.Fatalf("collectSpecs returned error: %v", err)
	}
	if len(specs) != 2 || !specs[0].Pending || !specs[1].Pending {
		t.Fatalf("specs = %+v, want both pending", specs)
	}

	// Per-comment flags belong in the batch entries, not on the command line.
	if err := cmd.Flags().Set("file", "src/app.py"); err != nil {
		t.Fatalf("setting flag: %v", err)
	}
	if _, err := collectSpecs(cmd, "", path); err == nil {
		t.Error("expected an error for --batch combined with --file")
	}
}

func TestCommentSpecRequest(t *testing.T) {
	spec := commentSpec{Body: "x", Blocker: true, Pending: true}
	anchor := &api.CommentAnchor{Path: "a.js", Line: 3}

	req := spec.request(anchor)

	if req.Severity != api.SeverityBlocker {
		t.Errorf("severity = %q, want %q", req.Severity, api.SeverityBlocker)
	}
	if req.State != api.CommentStatePending {
		t.Errorf("state = %q, want %q", req.State, api.CommentStatePending)
	}
	if req.Anchor != anchor {
		t.Error("anchor not carried into the request")
	}

	replySpec := commentSpec{Body: "x", ReplyTo: 7}
	reply := replySpec.request(nil)
	if reply.Parent == nil || reply.Parent.ID != 7 {
		t.Errorf("parent = %+v, want id 7", reply.Parent)
	}
	if reply.Severity != "" || reply.State != "" {
		t.Errorf("plain reply carries severity %q / state %q", reply.Severity, reply.State)
	}
}

func TestParsePRArgs(t *testing.T) {
	project, repo, prID, rest, err := parsePRArgs([]string{"MYPROJ/myrepo", "140", "LGTM"})
	if err != nil {
		t.Fatalf("parsePRArgs returned error: %v", err)
	}
	if project != "MYPROJ" || repo != "myrepo" || prID != 140 || rest != "LGTM" {
		t.Errorf("got %s/%s #%d %q", project, repo, prID, rest)
	}

	if _, _, _, _, err := parsePRArgs([]string{"MYPROJ/myrepo", "nine"}); err == nil {
		t.Error("expected an error for a non-numeric PR ID")
	}
}

func TestCommentsFromActivities(t *testing.T) {
	reply := api.Comment{ID: 3, Text: "fixed", CreatedDate: 300}
	anchored := api.Comment{ID: 2, Text: "peak leak", CreatedDate: 200, Comments: []api.Comment{reply}}
	general := api.Comment{ID: 1, Text: "NAK", CreatedDate: 100}

	activities := []api.Activity{
		{Action: "COMMENTED", Comment: &general},
		{Action: "COMMENTED", Comment: &anchored, CommentAnchor: &api.CommentAnchor{Path: "src/app.js", Line: 543}},
		// The feed repeats replies as activities of their own.
		{Action: "COMMENTED", Comment: &reply},
		{Action: "RESCOPED"},
	}

	entries := commentsFromActivities(activities)

	if len(entries) != 3 {
		t.Fatalf("got %d entries, want 3 (reply must not be duplicated)", len(entries))
	}
	if entries[0].Comment.ID != 1 || entries[0].Anchor != nil {
		t.Errorf("first entry = %+v, want the general comment", entries[0].Comment)
	}
	if entries[1].Comment.ID != 2 || entries[1].Depth != 0 {
		t.Errorf("second entry = %+v, want the anchored root comment", entries[1].Comment)
	}
	if entries[2].Comment.ID != 3 || entries[2].Depth != 1 {
		t.Errorf("third entry = %+v (depth %d), want the reply at depth 1", entries[2].Comment, entries[2].Depth)
	}
	if entries[2].Anchor == nil || entries[2].Anchor.Line != 543 {
		t.Error("reply did not inherit its parent's anchor")
	}
}

func TestCommentsFromActivitiesOrdersByLocation(t *testing.T) {
	activities := []api.Activity{
		{Action: "COMMENTED", Comment: &api.Comment{ID: 1, CreatedDate: 100}, CommentAnchor: &api.CommentAnchor{Path: "b.js", Line: 10}},
		{Action: "COMMENTED", Comment: &api.Comment{ID: 2, CreatedDate: 200}, CommentAnchor: &api.CommentAnchor{Path: "a.js", Line: 20}},
		{Action: "COMMENTED", Comment: &api.Comment{ID: 3, CreatedDate: 300}, CommentAnchor: &api.CommentAnchor{Path: "a.js", Line: 5}},
		{Action: "COMMENTED", Comment: &api.Comment{ID: 4, CreatedDate: 400}},
	}

	entries := commentsFromActivities(activities)

	var order []int
	for _, e := range entries {
		order = append(order, e.Comment.ID)
	}
	want := []int{4, 3, 2, 1}
	for i := range want {
		if order[i] != want[i] {
			t.Fatalf("order = %v, want %v (general first, then path and line)", order, want)
		}
	}
}

func TestFilterEntriesByFileKeepsReplies(t *testing.T) {
	entries := []commentEntry{
		{Comment: api.Comment{ID: 1}},
		{Comment: api.Comment{ID: 2}, Anchor: &api.CommentAnchor{Path: "src/app.js", Line: 5}},
		{Comment: api.Comment{ID: 3}, Anchor: &api.CommentAnchor{Path: "src/app.js", Line: 5}, Depth: 1},
		{Comment: api.Comment{ID: 4}, Anchor: &api.CommentAnchor{Path: "src/toolbar.vue", Line: 9}},
	}

	filtered := filterEntriesByFile(entries, "app.js")

	if len(filtered) != 2 || filtered[0].Comment.ID != 2 || filtered[1].Comment.ID != 3 {
		t.Fatalf("filtered = %+v, want the app.js thread with its reply", filtered)
	}
}

func TestDescribeTarget(t *testing.T) {
	tests := []struct {
		name   string
		spec   commentSpec
		anchor *api.CommentAnchor
		want   string
	}{
		{
			name: "general",
			spec: commentSpec{Body: "x"},
			want: "pull request comment",
		},
		{
			name:   "file",
			spec:   commentSpec{Body: "x", File: "a.js"},
			anchor: &api.CommentAnchor{Path: "a.js"},
			want:   "file comment on a.js",
		},
		{
			name:   "line task",
			spec:   commentSpec{Body: "x", File: "a.js", Line: 5, Blocker: true},
			anchor: &api.CommentAnchor{Path: "a.js", Line: 5, LineType: api.SegmentAdded, FileType: api.FileTypeTo},
			want:   "a.js:5 [ADDED/TO] [task]",
		},
		{
			name: "reply",
			spec: commentSpec{Body: "x", ReplyTo: 42},
			want: "reply to comment #42",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := describeTarget(tt.spec, tt.anchor); got != tt.want {
				t.Errorf("describeTarget() = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestLifecycleFlagConflict(t *testing.T) {
	cmd := prCommentCmd
	t.Cleanup(func() {
		_ = cmd.Flags().Set("file", "")
		cmd.Flags().Lookup("file").Changed = false
	})

	if err := lifecycleFlagConflict(cmd, "edit", "delete"); err != nil {
		t.Fatalf("no flags set, got error: %v", err)
	}

	if err := cmd.Flags().Set("file", "src/app.py"); err != nil {
		t.Fatalf("setting flag: %v", err)
	}
	err := lifecycleFlagConflict(cmd, "edit", "delete")
	if err == nil || !strings.Contains(err.Error(), "--file cannot be combined with --edit") {
		t.Fatalf("expected --file/--edit conflict, got: %v", err)
	}
}
