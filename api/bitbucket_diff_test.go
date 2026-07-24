package api

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

// diffFixture mirrors the shape Bitbucket Server returns for
// GET /pull-requests/{id}/diff: one modified file, one renamed file.
const diffFixture = `{
  "fromHash": "aaa111base",
  "toHash": "bbb222head",
  "contextLines": 10,
  "diffs": [
    {
      "source": {"toString": "src/app.js"},
      "destination": {"toString": "src/app.js"},
      "hunks": [
        {
          "sourceLine": 540,
          "sourceSpan": 4,
          "destinationLine": 540,
          "destinationSpan": 5,
          "segments": [
            {"type": "CONTEXT", "lines": [
              {"source": 540, "destination": 540, "line": "function peakUnit(u) {"},
              {"source": 541, "destination": 541, "line": "  const x = 1"}
            ]},
            {"type": "REMOVED", "lines": [
              {"source": 542, "destination": 541, "line": "  return u + '_peak'"}
            ]},
            {"type": "ADDED", "lines": [
              {"source": 542, "destination": 542, "line": "  return stripPeak(u)"},
              {"source": 542, "destination": 543, "line": "  // FIXME"}
            ]},
            {"type": "CONTEXT", "lines": [
              {"source": 543, "destination": 544, "line": "}"}
            ]}
          ]
        }
      ]
    },
    {
      "source": {"toString": "src/old/toolbar.vue"},
      "destination": {"toString": "src/new/toolbar.vue"},
      "hunks": [
        {
          "sourceLine": 10,
          "sourceSpan": 2,
          "destinationLine": 10,
          "destinationSpan": 2,
          "segments": [
            {"type": "ADDED", "lines": [
              {"source": 10, "destination": 10, "line": "const algo = ref('avg')"}
            ]},
            {"type": "CONTEXT", "lines": [
              {"source": 10, "destination": 11, "line": "</script>"}
            ]}
          ]
        }
      ]
    }
  ]
}`

func testDiff(t *testing.T) *Diff {
	t.Helper()

	var diff Diff
	if err := json.Unmarshal([]byte(diffFixture), &diff); err != nil {
		t.Fatalf("decoding fixture: %v", err)
	}
	return &diff
}

func TestResolveLineAnchor(t *testing.T) {
	diff := testDiff(t)

	tests := []struct {
		name     string
		path     string
		line     int
		side     DiffSide
		wantLine int
		wantType string
		wantFile string
		wantSrc  string
	}{
		{
			name: "added line on the new side",
			path: "src/app.js", line: 542, side: SideNew,
			wantLine: 542, wantType: SegmentAdded, wantFile: FileTypeTo,
		},
		{
			name: "context line on the new side",
			path: "src/app.js", line: 544, side: SideNew,
			wantLine: 544, wantType: SegmentContext, wantFile: FileTypeTo,
		},
		{
			name: "removed line on the old side",
			path: "src/app.js", line: 542, side: SideOld,
			wantLine: 542, wantType: SegmentRemoved, wantFile: FileTypeFrom,
		},
		{
			name: "context line on the old side",
			path: "src/app.js", line: 540, side: SideOld,
			wantLine: 540, wantType: SegmentContext, wantFile: FileTypeFrom,
		},
		{
			name: "renamed file keeps the source path",
			path: "src/new/toolbar.vue", line: 10, side: SideNew,
			wantLine: 10, wantType: SegmentAdded, wantFile: FileTypeTo,
			wantSrc: "src/old/toolbar.vue",
		},
		{
			name: "renamed file is also found by its old path",
			path: "src/old/toolbar.vue", line: 10, side: SideNew,
			wantLine: 10, wantType: SegmentAdded, wantFile: FileTypeTo,
			wantSrc: "src/old/toolbar.vue",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			anchor, err := diff.ResolveLineAnchor(tt.path, tt.line, tt.side)
			if err != nil {
				t.Fatalf("ResolveLineAnchor returned error: %v", err)
			}
			if anchor.Line != tt.wantLine {
				t.Errorf("line = %d, want %d", anchor.Line, tt.wantLine)
			}
			if anchor.LineType != tt.wantType {
				t.Errorf("lineType = %q, want %q", anchor.LineType, tt.wantType)
			}
			if anchor.FileType != tt.wantFile {
				t.Errorf("fileType = %q, want %q", anchor.FileType, tt.wantFile)
			}
			if anchor.SrcPath != tt.wantSrc {
				t.Errorf("srcPath = %q, want %q", anchor.SrcPath, tt.wantSrc)
			}
			if anchor.DiffType != DiffTypeEffective {
				t.Errorf("diffType = %q, want %q", anchor.DiffType, DiffTypeEffective)
			}
			if anchor.FromHash != "aaa111base" || anchor.ToHash != "bbb222head" {
				t.Errorf("hashes = %q/%q, want the diff's own hashes", anchor.FromHash, anchor.ToHash)
			}
		})
	}
}

func TestResolveLineAnchorRenamedFileUsesDestinationPath(t *testing.T) {
	anchor, err := testDiff(t).ResolveLineAnchor("src/old/toolbar.vue", 10, SideNew)
	if err != nil {
		t.Fatalf("ResolveLineAnchor returned error: %v", err)
	}

	if anchor.Path != "src/new/toolbar.vue" {
		t.Errorf("path = %q, want the destination path", anchor.Path)
	}
}

func TestResolveLineAnchorRejectsWrongSide(t *testing.T) {
	// Bitbucket fills in a source line number for added lines too, so an
	// old-side lookup must ignore ADDED segments rather than match them: the
	// old file ends at line 543.
	_, err := testDiff(t).ResolveLineAnchor("src/app.js", 544, SideOld)

	var lineErr *ErrLineNotInDiff
	if !errors.As(err, &lineErr) {
		t.Fatalf("error = %v, want ErrLineNotInDiff", err)
	}
	if !strings.Contains(lineErr.Error(), "540-543") {
		t.Errorf("error %q does not report the commentable source range", lineErr.Error())
	}
}

func TestResolveLineAnchorLineOutsideDiff(t *testing.T) {
	_, err := testDiff(t).ResolveLineAnchor("src/app.js", 9000, SideNew)

	var lineErr *ErrLineNotInDiff
	if !errors.As(err, &lineErr) {
		t.Fatalf("error = %v, want ErrLineNotInDiff", err)
	}
	if !strings.Contains(lineErr.Error(), "540-544") {
		t.Errorf("error %q does not report the commentable destination range", lineErr.Error())
	}
}

func TestResolveLineAnchorUnknownPath(t *testing.T) {
	_, err := testDiff(t).ResolveLineAnchor("src/untouched.js", 1, SideNew)

	var pathErr *ErrPathNotInDiff
	if !errors.As(err, &pathErr) {
		t.Fatalf("error = %v, want ErrPathNotInDiff", err)
	}
	if !strings.Contains(pathErr.Error(), "src/app.js") {
		t.Errorf("error %q does not list the changed files", pathErr.Error())
	}
}

func TestResolveFileAnchorHasNoLine(t *testing.T) {
	anchor, err := testDiff(t).ResolveFileAnchor("src/app.js")
	if err != nil {
		t.Fatalf("ResolveFileAnchor returned error: %v", err)
	}

	if anchor.Line != 0 || anchor.LineType != "" || anchor.FileType != "" {
		t.Errorf("file anchor carries line data: %+v", anchor)
	}
}

func TestParseDiffSide(t *testing.T) {
	tests := map[string]DiffSide{
		"":            SideNew,
		"new":         SideNew,
		"TO":          SideNew,
		"destination": SideNew,
		"old":         SideOld,
		"from":        SideOld,
		"source":      SideOld,
	}

	for in, want := range tests {
		got, err := ParseDiffSide(in)
		if err != nil {
			t.Fatalf("ParseDiffSide(%q) returned error: %v", in, err)
		}
		if got != want {
			t.Errorf("ParseDiffSide(%q) = %q, want %q", in, got, want)
		}
	}

	if _, err := ParseDiffSide("sideways"); err == nil {
		t.Error("ParseDiffSide accepted an invalid side")
	}
}

func TestGetPullRequestDiffJSONRequestsContextLines(t *testing.T) {
	var gotPath, gotContext, gotAccept string

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotPath = r.URL.Path
		gotContext = r.URL.Query().Get("contextLines")
		gotAccept = r.Header.Get("Accept")
		_, _ = w.Write([]byte(diffFixture))
	}))
	defer server.Close()

	client := NewBitbucketClient(server.URL, "tester", "token")
	client.HTTPClient = server.Client()

	diff, err := client.GetPullRequestDiffJSON(context.Background(), "MYPROJ", "myrepo", 140, 10)
	if err != nil {
		t.Fatalf("GetPullRequestDiffJSON returned error: %v", err)
	}

	wantPath := "/rest/api/1.0/projects/MYPROJ/repos/myrepo/pull-requests/140/diff"
	if gotPath != wantPath {
		t.Errorf("path = %q, want %q", gotPath, wantPath)
	}
	if gotContext != "10" {
		t.Errorf("contextLines = %q, want 10", gotContext)
	}
	if gotAccept != "application/json" {
		t.Errorf("Accept = %q, want application/json", gotAccept)
	}
	if len(diff.Diffs) != 2 {
		t.Errorf("decoded %d file diffs, want 2", len(diff.Diffs))
	}
}
