package cmd

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/lroolle/atlas-cli/api"
	"github.com/lroolle/atlas-cli/internal/cmdutil"
	"github.com/spf13/cobra"
)

// Context lines requested when resolving an anchor. The first value matches
// what the Bitbucket diff view shows; the second is a fallback for lines
// further away from a change.
const (
	anchorContextLines     = 10
	anchorWideContextLines = 500
)

// commentSpec describes one comment to post. It is both the internal
// representation of the flags and the schema of --batch entries.
type commentSpec struct {
	Body     string `json:"body"`
	File     string `json:"file,omitempty"`
	Line     int    `json:"line,omitempty"`
	Side     string `json:"side,omitempty"`
	ReplyTo  int    `json:"reply_to,omitempty"`
	Blocker  bool   `json:"blocker,omitempty"`
	Pending  bool   `json:"pending,omitempty"`
	LineType string `json:"line_type,omitempty"`
	FileType string `json:"file_type,omitempty"`
}

func (s *commentSpec) validate() error {
	if strings.TrimSpace(s.Body) == "" {
		return errors.New("body required")
	}
	if s.ReplyTo != 0 && (s.File != "" || s.Line != 0) {
		return errors.New("--reply cannot be combined with --file/--line: a reply inherits its parent's anchor")
	}
	if s.Line != 0 && s.File == "" {
		return errors.New("--line requires --file")
	}
	if s.Line < 0 {
		return fmt.Errorf("invalid line %d", s.Line)
	}
	if _, err := api.ParseDiffSide(s.Side); err != nil {
		return err
	}
	if err := validateEnum("line-type", s.LineType, api.SegmentAdded, api.SegmentRemoved, api.SegmentContext); err != nil {
		return err
	}
	return validateEnum("file-type", s.FileType, api.FileTypeTo, api.FileTypeFrom)
}

func validateEnum(name, value string, allowed ...string) error {
	if value == "" {
		return nil
	}
	for _, a := range allowed {
		if strings.EqualFold(value, a) {
			return nil
		}
	}
	return fmt.Errorf("invalid --%s %q: use one of %s", name, value, strings.Join(allowed, ", "))
}

// hasManualAnchor reports whether the caller pinned down the anchor themselves,
// in which case the diff does not have to resolve it.
func (s *commentSpec) hasManualAnchor() bool {
	return s.LineType != "" && s.FileType != ""
}

func (s *commentSpec) request(anchor *api.CommentAnchor) api.CommentRequest {
	req := api.CommentRequest{Text: s.Body, Anchor: anchor}
	if s.ReplyTo != 0 {
		req.Parent = &api.CommentParent{ID: s.ReplyTo}
	}
	if s.Blocker {
		req.Severity = api.SeverityBlocker
	}
	if s.Pending {
		req.State = api.CommentStatePending
	}
	return req
}

// anchorResolver turns file/line references into Bitbucket anchors, fetching
// the pull request diff at most twice regardless of how many comments it
// resolves.
type anchorResolver struct {
	client  *api.BitbucketClient
	project string
	repo    string
	prID    int

	diff       *api.Diff
	diffIsWide bool
}

func (r *anchorResolver) getDiff(ctx context.Context, wide bool) (*api.Diff, error) {
	if r.diff != nil && (r.diffIsWide || !wide) {
		return r.diff, nil
	}

	contextLines := anchorContextLines
	if wide {
		contextLines = anchorWideContextLines
	}

	diff, err := r.client.GetPullRequestDiffJSON(ctx, r.project, r.repo, r.prID, contextLines)
	if err != nil {
		return nil, fmt.Errorf("fetching diff: %w", err)
	}

	r.diff, r.diffIsWide = diff, wide
	return diff, nil
}

func (r *anchorResolver) resolve(ctx context.Context, spec commentSpec) (*api.CommentAnchor, error) {
	if spec.File == "" {
		return nil, nil
	}

	side, err := api.ParseDiffSide(spec.Side)
	if err != nil {
		return nil, err
	}

	diff, err := r.getDiff(ctx, false)
	if err != nil {
		return nil, err
	}

	anchor, err := resolveAnchor(diff, spec, side)

	// A line outside the default diff context may still be commentable; widen
	// the context once before giving up.
	var lineErr *api.ErrLineNotInDiff
	if errors.As(err, &lineErr) && !r.diffIsWide {
		wideDiff, wideErr := r.getDiff(ctx, true)
		if wideErr != nil {
			return nil, wideErr
		}
		anchor, err = resolveAnchor(wideDiff, spec, side)
	}

	if err != nil {
		if !spec.hasManualAnchor() {
			return nil, err
		}
		anchor = &api.CommentAnchor{
			DiffType: api.DiffTypeEffective,
			Path:     spec.File,
			Line:     spec.Line,
			FromHash: diff.FromHash,
			ToHash:   diff.ToHash,
		}
	}

	if spec.LineType != "" {
		anchor.LineType = strings.ToUpper(spec.LineType)
	}
	if spec.FileType != "" {
		anchor.FileType = strings.ToUpper(spec.FileType)
	}

	return anchor, nil
}

func resolveAnchor(diff *api.Diff, spec commentSpec, side api.DiffSide) (*api.CommentAnchor, error) {
	if spec.Line == 0 {
		return diff.ResolveFileAnchor(spec.File)
	}
	return diff.ResolveLineAnchor(spec.File, spec.Line, side)
}

var prCommentCmd = &cobra.Command{
	Use:   "comment [project/repo] <pr-id> [text]",
	Short: "Comment on a pull request, optionally on a specific line",
	Long: `Add a comment to a pull request.

Without --file the comment lands on the pull request itself. With --file and
--line it is anchored to that line of the diff, the way inline review comments
are in the web UI. The line type (ADDED/REMOVED/CONTEXT) is resolved from the
diff, so only a path and a line number are needed.

--line refers to the line in the new file; use --side old to comment on a line
that the pull request deletes. Only lines the diff touches (changed lines and
their surrounding context) can be anchored.`,
	Example: `  # General comment
  atl pr comment MYPROJ/myrepo 140 "LGTM"

  # Inline comment on a line of the new file
  atl pr comment MYPROJ/myrepo 140 -f src/app.js -L 543 -b "this allocation leaks"

  # Comment on a deleted line, as a blocker task
  atl pr comment MYPROJ/myrepo 140 -f src/app.js -L 120 --side old --blocker -b "why drop this?"

  # Reply to an existing comment
  atl pr comment MYPROJ/myrepo 140 --reply 331 -b "fixed in a1b2c3d"

  # Post a whole review from JSON, checking the anchors first
  atl pr comment MYPROJ/myrepo 140 --batch findings.json --dry-run
  atl pr comment MYPROJ/myrepo 140 --batch findings.json

  # Correct or remove an existing comment (yours; pending or published)
  atl pr comment MYPROJ/myrepo 140 --edit 331 -b "corrected text"
  atl pr comment MYPROJ/myrepo 140 --delete 331`,
	Args: cobra.RangeArgs(1, 3),
	RunE: runPRComment,
}

func runPRComment(cmd *cobra.Command, args []string) error {
	ctx := cmd.Context()

	project, repo, prID, rest, err := parsePRArgs(args)
	if err != nil {
		return err
	}

	if editID, _ := cmd.Flags().GetInt("edit"); editID != 0 {
		return runPRCommentEdit(cmd, project, repo, prID, editID, rest)
	}
	if deleteID, _ := cmd.Flags().GetInt("delete"); deleteID != 0 {
		return runPRCommentDelete(cmd, project, repo, prID, deleteID, rest)
	}

	batchFile, _ := cmd.Flags().GetString("batch")
	dryRun, _ := cmd.Flags().GetBool("dry-run")
	jsonOutput, _ := cmd.Flags().GetBool("json")

	specs, err := collectSpecs(cmd, rest, batchFile)
	if err != nil {
		return err
	}

	client, err := getClient()
	if err != nil {
		return err
	}

	resolver := &anchorResolver{client: client, project: project, repo: repo, prID: prID}

	// Resolve every anchor before posting anything, so a bad path or line in
	// one entry does not leave a batch half-posted.
	anchors := make([]*api.CommentAnchor, len(specs))
	for i, spec := range specs {
		anchor, err := resolver.resolve(ctx, spec)
		if err != nil {
			return fmt.Errorf("comment %d/%d: %w", i+1, len(specs), err)
		}
		anchors[i] = anchor
	}

	if dryRun {
		return printCommentDryRun(prID, specs, anchors)
	}

	posted := make([]*api.Comment, 0, len(specs))
	for i, spec := range specs {
		comment, err := client.CreatePullRequestComment(ctx, project, repo, prID, spec.request(anchors[i]))
		if err != nil {
			if len(posted) > 0 {
				fmt.Fprintf(os.Stderr, "posted %d of %d comments before failing\n", len(posted), len(specs))
			}
			return fmt.Errorf("comment %d/%d: %w", i+1, len(specs), err)
		}
		posted = append(posted, comment)

		if !jsonOutput {
			fmt.Printf("✓ %s (ID: %d)\n", describeTarget(spec, anchors[i]), comment.ID)
		}
	}

	if jsonOutput {
		return json.NewEncoder(os.Stdout).Encode(posted)
	}

	return nil
}

// lifecycleFlagConflict reports the first anchor/review flag that makes no
// sense when editing or deleting an existing comment.
func lifecycleFlagConflict(cmd *cobra.Command, action string, extra ...string) error {
	conflicting := append([]string{"file", "line", "side", "reply", "blocker", "pending", "batch", "dry-run", "line-type", "file-type"}, extra...)
	for _, flag := range conflicting {
		if cmd.Flags().Changed(flag) {
			return fmt.Errorf("--%s cannot be combined with --%s: %s targets an existing comment", flag, action, action)
		}
	}
	return nil
}

func runPRCommentEdit(cmd *cobra.Command, project, repo string, prID, commentID int, positionalText string) error {
	ctx := cmd.Context()

	if err := lifecycleFlagConflict(cmd, "edit", "delete"); err != nil {
		return err
	}
	body, err := resolveCommentBody(cmd, positionalText)
	if err != nil {
		return err
	}

	client, err := getClient()
	if err != nil {
		return err
	}

	current, err := client.GetPullRequestComment(ctx, project, repo, prID, commentID)
	if err != nil {
		return fmt.Errorf("fetching comment %d: %w", commentID, err)
	}
	updated, err := client.UpdatePullRequestComment(ctx, project, repo, prID, commentID, current.Version, body)
	if err != nil {
		return fmt.Errorf("updating comment %d: %w", commentID, err)
	}

	if jsonOutput, _ := cmd.Flags().GetBool("json"); jsonOutput {
		return json.NewEncoder(os.Stdout).Encode(updated)
	}
	fmt.Printf("✓ Updated comment %d (v%d -> v%d)\n", updated.ID, current.Version, updated.Version)
	return nil
}

func runPRCommentDelete(cmd *cobra.Command, project, repo string, prID, commentID int, positionalText string) error {
	ctx := cmd.Context()

	if err := lifecycleFlagConflict(cmd, "delete", "body", "body-file", "edit"); err != nil {
		return err
	}
	if positionalText != "" {
		return errors.New("--delete takes no comment text")
	}

	client, err := getClient()
	if err != nil {
		return err
	}

	current, err := client.GetPullRequestComment(ctx, project, repo, prID, commentID)
	if err != nil {
		return fmt.Errorf("fetching comment %d: %w", commentID, err)
	}
	if err := client.DeletePullRequestComment(ctx, project, repo, prID, commentID, current.Version); err != nil {
		return fmt.Errorf("deleting comment %d: %w", commentID, err)
	}

	fmt.Printf("✓ Deleted comment %d\n", commentID)
	return nil
}

// collectSpecs builds the comment list from either --batch or the flags and
// positional text.
func collectSpecs(cmd *cobra.Command, positionalText, batchFile string) ([]commentSpec, error) {
	if batchFile != "" {
		if positionalText != "" {
			return nil, errors.New("--batch carries its own comments: drop the comment text argument")
		}
		for _, flag := range []string{"body", "body-file", "file", "line", "side", "reply", "blocker", "line-type", "file-type"} {
			if cmd.Flags().Changed(flag) {
				return nil, fmt.Errorf("--batch carries its own comments: --%s belongs in the batch entries", flag)
			}
		}

		specs, err := readBatch(batchFile)
		if err != nil {
			return nil, err
		}

		// Drafting is a property of the whole review, so the flag overrides
		// what the batch file says.
		if pending, _ := cmd.Flags().GetBool("pending"); pending {
			for i := range specs {
				specs[i].Pending = true
			}
		}

		return specs, nil
	}

	body, err := resolveCommentBody(cmd, positionalText)
	if err != nil {
		return nil, err
	}

	spec := commentSpec{Body: body}
	spec.File, _ = cmd.Flags().GetString("file")
	spec.Line, _ = cmd.Flags().GetInt("line")
	spec.Side, _ = cmd.Flags().GetString("side")
	spec.ReplyTo, _ = cmd.Flags().GetInt("reply")
	spec.Blocker, _ = cmd.Flags().GetBool("blocker")
	spec.Pending, _ = cmd.Flags().GetBool("pending")
	spec.LineType, _ = cmd.Flags().GetString("line-type")
	spec.FileType, _ = cmd.Flags().GetString("file-type")

	if err := spec.validate(); err != nil {
		return nil, err
	}

	return []commentSpec{spec}, nil
}

func resolveCommentBody(cmd *cobra.Command, positionalText string) (string, error) {
	body, _ := cmd.Flags().GetString("body")
	bodyFile, _ := cmd.Flags().GetString("body-file")

	set := 0
	for _, v := range []string{positionalText, body, bodyFile} {
		if v != "" {
			set++
		}
	}
	if set > 1 {
		return "", errors.New("provide comment text once: as an argument, --body or --body-file")
	}

	if bodyFile != "" {
		data, err := readFileOrStdin(bodyFile)
		if err != nil {
			return "", err
		}
		return strings.TrimRight(string(data), "\n"), nil
	}

	if body != "" {
		return body, nil
	}
	if positionalText != "" {
		return positionalText, nil
	}

	return "", errors.New("comment text required: pass it as an argument, --body or --body-file")
}

func readBatch(path string) ([]commentSpec, error) {
	data, err := readFileOrStdin(path)
	if err != nil {
		return nil, err
	}

	var specs []commentSpec
	if err := json.Unmarshal(data, &specs); err != nil {
		return nil, fmt.Errorf("parsing batch %s: %w (expected a JSON array of {body, file, line, side, reply_to, blocker, pending})", path, err)
	}
	if len(specs) == 0 {
		return nil, fmt.Errorf("batch %s contains no comments", path)
	}

	for i := range specs {
		if err := specs[i].validate(); err != nil {
			return nil, fmt.Errorf("batch entry %d: %w", i+1, err)
		}
	}

	return specs, nil
}

func readFileOrStdin(path string) ([]byte, error) {
	if path == "-" {
		return io.ReadAll(os.Stdin)
	}
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("reading %s: %w", path, err)
	}
	return data, nil
}

func printCommentDryRun(prID int, specs []commentSpec, anchors []*api.CommentAnchor) error {
	fmt.Printf("Would post %d comment(s) on PR #%d:\n", len(specs), prID)
	for i, spec := range specs {
		fmt.Printf("\n%d. %s\n", i+1, describeTarget(spec, anchors[i]))
		for _, line := range strings.Split(spec.Body, "\n") {
			fmt.Printf("   | %s\n", line)
		}
	}
	return nil
}

func describeTarget(spec commentSpec, anchor *api.CommentAnchor) string {
	var target string
	switch {
	case spec.ReplyTo != 0:
		target = fmt.Sprintf("reply to comment #%d", spec.ReplyTo)
	case anchor == nil:
		target = "pull request comment"
	case anchor.Line == 0:
		target = fmt.Sprintf("file comment on %s", anchor.Path)
	default:
		target = fmt.Sprintf("%s:%d [%s/%s]", anchor.Path, anchor.Line, anchor.LineType, anchor.FileType)
	}

	if spec.Blocker {
		target += " [task]"
	}
	if spec.Pending {
		target += " [pending]"
	}
	return target
}

var prCommentsCmd = &cobra.Command{
	Use:     "comments [project/repo] <pr-id>",
	Short:   "List comments on a pull request, grouped by file and line",
	Aliases: []string{"reviews"},
	Long: `List the comments on a pull request.

Comments are grouped by what they are anchored to: general pull request
comments first, then each file and line. Comment IDs shown here are what
'atl pr comment --reply' takes.`,
	Args: cobra.RangeArgs(1, 2),
	RunE: runPRComments,
}

// commentEntry is a comment plus where it sits in the review.
type commentEntry struct {
	Comment api.Comment        `json:"comment"`
	Anchor  *api.CommentAnchor `json:"anchor,omitempty"`
	Depth   int                `json:"depth"`
}

func runPRComments(cmd *cobra.Command, args []string) error {
	ctx := cmd.Context()

	project, repo, prID, _, err := parsePRArgs(args)
	if err != nil {
		return err
	}

	client, err := getClient()
	if err != nil {
		return err
	}

	limit, _ := cmd.Flags().GetInt("limit")
	fileFilter, _ := cmd.Flags().GetString("file")
	jsonOutput, _ := cmd.Flags().GetBool("json")
	pending, _ := cmd.Flags().GetBool("pending")

	var entries []commentEntry
	if pending {
		comments, err := client.GetPendingReview(ctx, project, repo, prID, limit)
		if err != nil {
			return fmt.Errorf("fetching pending review: %w", err)
		}
		for _, c := range comments {
			entries = appendCommentTree(entries, c, nil, 0)
		}
		sortCommentEntries(entries)
	} else {
		activities, err := client.GetPullRequestActivity(ctx, project, repo, prID, limit)
		if err != nil {
			return fmt.Errorf("fetching comments: %w", err)
		}
		entries = commentsFromActivities(activities)
	}

	if fileFilter != "" {
		entries = filterEntriesByFile(entries, fileFilter)
	}

	if jsonOutput {
		return json.NewEncoder(os.Stdout).Encode(entries)
	}

	if len(entries) == 0 {
		fmt.Println("No comments found")
		return nil
	}

	printCommentEntries(entries)
	return nil
}

// commentsFromActivities flattens the activity feed into comment threads. The
// feed reports replies both nested under their parent and as activities of
// their own, so nested IDs win and the duplicates are dropped.
func commentsFromActivities(activities []api.Activity) []commentEntry {
	nested := map[int]bool{}
	for _, a := range activities {
		if a.Comment != nil {
			markNested(nested, *a.Comment)
		}
	}

	var entries []commentEntry
	seen := map[int]bool{}
	for _, a := range activities {
		if a.Comment == nil || nested[a.Comment.ID] || seen[a.Comment.ID] {
			continue
		}
		seen[a.Comment.ID] = true
		entries = appendCommentTree(entries, *a.Comment, a.CommentAnchor, 0)
	}

	sortCommentEntries(entries)
	return entries
}

func markNested(nested map[int]bool, c api.Comment) {
	for _, reply := range c.Comments {
		nested[reply.ID] = true
		markNested(nested, reply)
	}
}

func appendCommentTree(entries []commentEntry, c api.Comment, anchor *api.CommentAnchor, depth int) []commentEntry {
	if anchor == nil {
		anchor = c.Anchor
	}

	replies := c.Comments
	c.Comments = nil
	entries = append(entries, commentEntry{Comment: c, Anchor: anchor, Depth: depth})

	for _, reply := range replies {
		entries = appendCommentTree(entries, reply, anchor, depth+1)
	}
	return entries
}

// sortCommentEntries orders threads by location: general comments first, then
// by path and line. Replies keep their position under their root comment.
func sortCommentEntries(entries []commentEntry) {
	type thread struct {
		entries []commentEntry
		path    string
		line    int
		created int64
	}

	var threads []thread
	for _, e := range entries {
		if e.Depth == 0 {
			t := thread{created: e.Comment.CreatedDate}
			if e.Anchor != nil {
				t.path, t.line = e.Anchor.Path, e.Anchor.Line
			}
			threads = append(threads, t)
		}
		if len(threads) == 0 {
			continue
		}
		threads[len(threads)-1].entries = append(threads[len(threads)-1].entries, e)
	}

	sort.SliceStable(threads, func(i, j int) bool {
		a, b := threads[i], threads[j]
		if (a.path == "") != (b.path == "") {
			return a.path == ""
		}
		if a.path != b.path {
			return a.path < b.path
		}
		if a.line != b.line {
			return a.line < b.line
		}
		return a.created < b.created
	})

	i := 0
	for _, t := range threads {
		for _, e := range t.entries {
			entries[i] = e
			i++
		}
	}
}

func filterEntriesByFile(entries []commentEntry, file string) []commentEntry {
	var filtered []commentEntry
	keep := false
	for _, e := range entries {
		if e.Depth == 0 {
			keep = e.Anchor != nil && strings.Contains(e.Anchor.Path, file)
		}
		if keep {
			filtered = append(filtered, e)
		}
	}
	return filtered
}

func printCommentEntries(entries []commentEntry) {
	lastGroup := "\x00"

	for _, e := range entries {
		if e.Depth == 0 {
			if group := commentGroup(e.Anchor); group != lastGroup {
				if lastGroup != "\x00" {
					fmt.Println()
				}
				fmt.Println(group)
				lastGroup = group
			}
		}

		indent := strings.Repeat("  ", e.Depth+1)
		date := time.Unix(e.Comment.CreatedDate/1000, 0).Format("2006-01-02 15:04")
		fmt.Printf("%s#%d  %s  %s%s\n", indent, e.Comment.ID, e.Comment.Author.Name, date, commentMarkers(e))

		for _, line := range strings.Split(strings.TrimRight(e.Comment.Text, "\n"), "\n") {
			fmt.Printf("%s  %s\n", indent, line)
		}
	}
}

func commentGroup(anchor *api.CommentAnchor) string {
	if anchor == nil || anchor.Path == "" {
		return "General"
	}
	group := anchor.Location()
	if anchor.LineType != "" {
		group += "  [" + anchor.LineType + "]"
	}
	return group
}

func commentMarkers(e commentEntry) string {
	var markers []string
	if e.Comment.Severity == api.SeverityBlocker {
		markers = append(markers, "TASK")
	}
	if state := e.Comment.State; state != "" && state != api.CommentStateOpen {
		markers = append(markers, state)
	}
	if e.Anchor != nil && e.Anchor.Orphaned {
		markers = append(markers, "ORPHANED")
	}
	if len(markers) == 0 {
		return ""
	}
	return "  [" + strings.Join(markers, "] [") + "]"
}

// parsePRArgs handles the shared [project/repo] <pr-id> [text] argument shape.
func parsePRArgs(args []string) (project, repo string, prID int, rest string, err error) {
	var prIDStr string

	if len(args) > 1 && !isNumeric(args[0]) {
		project, repo, err = parseRepoArg(args[0])
		if err != nil {
			return "", "", 0, "", err
		}
		prIDStr = args[1]
		if len(args) > 2 {
			rest = args[2]
		}
	} else {
		project, repo, err = parseRepoArg("")
		if err != nil {
			return "", "", 0, "", fmt.Errorf("PR ID required: %w", err)
		}
		prIDStr = args[0]
		if len(args) > 1 {
			rest = args[1]
		}
	}

	prID, err = strconv.Atoi(prIDStr)
	if err != nil {
		return "", "", 0, "", fmt.Errorf("invalid PR ID: %q", prIDStr)
	}

	return project, repo, prID, rest, nil
}

func isNumeric(s string) bool {
	_, err := strconv.Atoi(s)
	return err == nil
}

func init() {
	prCmd.AddCommand(prCommentCmd)
	prCmd.AddCommand(prCommentsCmd)

	prCommentCmd.Flags().StringP("body", "b", "", "Comment text")
	prCommentCmd.Flags().StringP("body-file", "F", "", "Read comment text from file ('-' for stdin)")
	prCommentCmd.Flags().StringP("file", "f", "", "Anchor the comment to this file of the diff")
	prCommentCmd.Flags().IntP("line", "L", 0, "Anchor the comment to this line (requires --file)")
	prCommentCmd.Flags().String("side", "new", "Which side --line refers to: new or old")
	prCommentCmd.Flags().Int("reply", 0, "Reply to an existing comment ID")
	prCommentCmd.Flags().Bool("blocker", false, "Post as a task that blocks the merge")
	prCommentCmd.Flags().Bool("pending", false, "Keep as an unpublished review comment")
	prCommentCmd.Flags().String("batch", "", "Post multiple comments from a JSON file ('-' for stdin)")
	prCommentCmd.Flags().Bool("dry-run", false, "Resolve anchors and print what would be posted")
	prCommentCmd.Flags().Bool("json", false, "Output created comments as JSON")
	prCommentCmd.Flags().String("line-type", "", "Override the resolved line type (ADDED, REMOVED, CONTEXT)")
	prCommentCmd.Flags().String("file-type", "", "Override the resolved file side (TO, FROM)")
	prCommentCmd.Flags().Int("edit", 0, "Replace the text of an existing comment ID")
	prCommentCmd.Flags().Int("delete", 0, "Delete an existing comment ID")

	prCommentsCmd.Flags().StringP("file", "f", "", "Only show comments anchored to paths containing this string")
	prCommentsCmd.Flags().Int("limit", cmdutil.DefaultActivityLimit, "Maximum number of activities to scan")
	prCommentsCmd.Flags().Bool("pending", false, "Show your unpublished review comments instead")
	prCommentsCmd.Flags().Bool("json", false, "Output as JSON")
}
