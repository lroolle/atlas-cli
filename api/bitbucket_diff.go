package api

import (
	"context"
	"fmt"
	"net/url"
	"strconv"
	"strings"
)

// Segment types returned by the structured diff endpoint.
const (
	SegmentContext = "CONTEXT"
	SegmentAdded   = "ADDED"
	SegmentRemoved = "REMOVED"
)

// Anchor file sides. FROM is the source (pre-change) file, TO the destination.
const (
	FileTypeFrom = "FROM"
	FileTypeTo   = "TO"
)

const (
	DiffTypeEffective = "EFFECTIVE"
	DiffTypeCommit    = "COMMIT"
	DiffTypeRange     = "RANGE"
)

// DiffSide selects which side of the diff a line number refers to.
type DiffSide string

const (
	SideNew DiffSide = "new"
	SideOld DiffSide = "old"
)

func ParseDiffSide(s string) (DiffSide, error) {
	switch strings.ToLower(strings.TrimSpace(s)) {
	case "", "new", "to", "destination":
		return SideNew, nil
	case "old", "from", "source":
		return SideOld, nil
	default:
		return "", fmt.Errorf("invalid side %q: use 'new' or 'old'", s)
	}
}

// Diff is the structured (JSON) form of a pull request diff.
type Diff struct {
	FromHash  string     `json:"fromHash"`
	ToHash    string     `json:"toHash"`
	Diffs     []FileDiff `json:"diffs"`
	Truncated bool       `json:"truncated"`
}

type FileDiff struct {
	Source      *Path  `json:"source"`
	Destination *Path  `json:"destination"`
	Hunks       []Hunk `json:"hunks"`
	Truncated   bool   `json:"truncated"`
}

type Hunk struct {
	Context         string    `json:"context"`
	SourceLine      int       `json:"sourceLine"`
	SourceSpan      int       `json:"sourceSpan"`
	DestinationLine int       `json:"destinationLine"`
	DestinationSpan int       `json:"destinationSpan"`
	Segments        []Segment `json:"segments"`
	Truncated       bool      `json:"truncated"`
}

type Segment struct {
	Type      string     `json:"type"`
	Lines     []DiffLine `json:"lines"`
	Truncated bool       `json:"truncated"`
}

type DiffLine struct {
	Source      int    `json:"source"`
	Destination int    `json:"destination"`
	Line        string `json:"line"`
	Truncated   bool   `json:"truncated"`
}

// GetPullRequestDiffJSON fetches the effective diff as structured JSON.
// contextLines controls how many unchanged lines surround each hunk, which
// determines how far from a change a comment can still be anchored.
func (c *BitbucketClient) GetPullRequestDiffJSON(ctx context.Context, project, repo string, prID, contextLines int) (*Diff, error) {
	params := url.Values{}
	if contextLines > 0 {
		params.Set("contextLines", strconv.Itoa(contextLines))
	}

	path := fmt.Sprintf("/rest/api/1.0/projects/%s/repos/%s/pull-requests/%d/diff", project, repo, prID)

	var diff Diff
	if err := c.Get(ctx, path, params, &diff); err != nil {
		return nil, err
	}

	return &diff, nil
}

// Paths lists every file path present in the diff, destination side first.
func (d *Diff) Paths() []string {
	paths := make([]string, 0, len(d.Diffs))
	for i := range d.Diffs {
		paths = append(paths, d.Diffs[i].Path())
	}
	return paths
}

// FindFile locates a file in the diff by destination or source path.
func (d *Diff) FindFile(path string) *FileDiff {
	path = strings.TrimPrefix(strings.TrimSpace(path), "./")

	for i := range d.Diffs {
		fd := &d.Diffs[i]
		if fd.Destination != nil && fd.Destination.ToString == path {
			return fd
		}
		if fd.Source != nil && fd.Source.ToString == path {
			return fd
		}
	}

	return nil
}

// Path is the path a comment anchor should use: the destination path, falling
// back to the source path for deleted files.
func (f *FileDiff) Path() string {
	if f.Destination != nil {
		return f.Destination.ToString
	}
	if f.Source != nil {
		return f.Source.ToString
	}
	return ""
}

// SrcPath is the pre-change path, set only for copies and moves.
func (f *FileDiff) SrcPath() string {
	if f.Source == nil || f.Destination == nil {
		return ""
	}
	if f.Source.ToString == f.Destination.ToString {
		return ""
	}
	return f.Source.ToString
}

// LineRanges reports the line ranges of the given side that the diff covers,
// i.e. the lines a comment can be anchored to.
func (f *FileDiff) LineRanges(side DiffSide) []string {
	ranges := make([]string, 0, len(f.Hunks))
	for _, h := range f.Hunks {
		start, span := h.DestinationLine, h.DestinationSpan
		if side == SideOld {
			start, span = h.SourceLine, h.SourceSpan
		}
		if span <= 0 {
			continue
		}
		ranges = append(ranges, fmt.Sprintf("%d-%d", start, start+span-1))
	}
	return ranges
}

// ErrPathNotInDiff reports a file that the pull request does not touch.
type ErrPathNotInDiff struct {
	Path      string
	Available []string
}

func (e *ErrPathNotInDiff) Error() string {
	msg := fmt.Sprintf("%q is not part of this pull request's diff", e.Path)
	if len(e.Available) > 0 {
		msg += "\nchanged files:\n  " + strings.Join(e.Available, "\n  ")
	}
	return msg
}

// ErrLineNotInDiff reports a line that exists in the file but not in the diff
// context, so Bitbucket cannot anchor a comment to it.
type ErrLineNotInDiff struct {
	Path   string
	Line   int
	Side   DiffSide
	Ranges []string
}

func (e *ErrLineNotInDiff) Error() string {
	msg := fmt.Sprintf("line %d (%s side) is not in the diff for %s", e.Line, e.Side, e.Path)
	if len(e.Ranges) > 0 {
		msg += fmt.Sprintf("\ncommentable lines: %s", strings.Join(e.Ranges, ", "))
	}
	return msg + "\n(comments can only be anchored to changed lines or their surrounding context)"
}

// ResolveFileAnchor builds a file-level anchor: a comment on the file as a
// whole rather than on one of its lines.
func (d *Diff) ResolveFileAnchor(path string) (*CommentAnchor, error) {
	fd := d.FindFile(path)
	if fd == nil {
		return nil, &ErrPathNotInDiff{Path: path, Available: d.Paths()}
	}

	return d.baseAnchor(fd), nil
}

// ResolveLineAnchor finds line on the requested side of the diff and returns
// the anchor Bitbucket needs to pin a comment to it. The line type (ADDED,
// REMOVED, CONTEXT) is derived from the diff, so callers only need a path and
// a line number as they read it in the file.
func (d *Diff) ResolveLineAnchor(path string, line int, side DiffSide) (*CommentAnchor, error) {
	fd := d.FindFile(path)
	if fd == nil {
		return nil, &ErrPathNotInDiff{Path: path, Available: d.Paths()}
	}

	for _, h := range fd.Hunks {
		for _, s := range h.Segments {
			for _, l := range s.Lines {
				lineNo, fileType := l.Destination, FileTypeTo
				if side == SideOld {
					lineNo, fileType = l.Source, FileTypeFrom
				}
				if lineNo != line || !segmentHasSide(s.Type, side) {
					continue
				}

				anchor := d.baseAnchor(fd)
				anchor.Line = lineNo
				anchor.LineType = s.Type
				anchor.FileType = fileType
				return anchor, nil
			}
		}
	}

	return nil, &ErrLineNotInDiff{
		Path:   fd.Path(),
		Line:   line,
		Side:   side,
		Ranges: fd.LineRanges(side),
	}
}

// segmentHasSide reports whether a segment exists on the given side of the
// diff: added lines only exist in the new file, removed lines only in the old.
func segmentHasSide(segmentType string, side DiffSide) bool {
	switch segmentType {
	case SegmentAdded:
		return side == SideNew
	case SegmentRemoved:
		return side == SideOld
	default:
		return true
	}
}

func (d *Diff) baseAnchor(fd *FileDiff) *CommentAnchor {
	return &CommentAnchor{
		DiffType: DiffTypeEffective,
		Path:     fd.Path(),
		SrcPath:  fd.SrcPath(),
		FromHash: d.FromHash,
		ToHash:   d.ToHash,
	}
}
