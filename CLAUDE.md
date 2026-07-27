# Atlas CLI Architecture Notes

## Reference Analysis

### gh-cli (GitHub CLI) - Gold Standard

**Structure:**
```
pkg/cmd/{resource}/{action}/
├── {action}.go      # Command implementation
└── {action}_test.go # Tests

pkg/cmd/issue/
├── issue.go         # Root command, aggregates subcommands
├── delete/delete.go # Each action in own package
├── create/create.go
├── view/view.go
└── shared/lookup.go # Common utilities (ParseIssueFromArg)
```

**Key Patterns:**

1. **Factory Pattern** - DI container for common dependencies
   ```go
   type Factory struct {
       IOStreams  *iostreams.IOStreams
       HttpClient func() (*http.Client, error)
       Config     func() (gh.Config, error)
       Prompter   iprompter
       BaseRepo   func() (ghrepo.Interface, error)
   }
   ```

2. **Options Struct** - Each command encapsulates its state
   ```go
   type DeleteOptions struct {
       HttpClient  func() (*http.Client, error)
       IO          *iostreams.IOStreams
       IssueNumber int
       Confirmed   bool
   }
   ```

3. **Test Injection** - Commands accept optional test runner
   ```go
   func NewCmdDelete(f *Factory, runF func(*Options) error) *cobra.Command
   ```

4. **Shared Lookup** - Unified argument parsing (ID or URL)
   ```go
   // Handles: 123, #123, https://github.com/owner/repo/issues/123
   issueNumber, repo, err := shared.ParseIssueFromArg(arg)
   ```

5. **Confirmation Pattern** - Interactive prompt for destructive ops
   ```go
   if opts.IO.CanPrompt() && !opts.Confirmed {
       fmt.Printf("%s Deleted issues cannot be recovered.\n", cs.WarningIcon())
       err := opts.Prompter.ConfirmDeletion(fmt.Sprintf("%d", issue.Number))
   }
   ```

### jira-cli - Atlassian Reference

**Structure:**
```
internal/cmd/{resource}/{action}/
├── {action}.go

internal/cmdutil/
├── utils.go     # ExitIfError, Info (spinner), Success, Fail
```

**Key Patterns:**

1. **Simpler DI** - Uses viper globals, less abstraction
2. **Spinner for long ops** - `cmdutil.Info("Removing issue...")` returns stoppable spinner
3. **Color output** - Success (green ✓), Fail (red ✗), Warn (yellow)
4. **Key normalization** - `GetJiraIssueKey(project, "123")` → `"PROJECT-123"`

**Delete Command:**
```go
cmd.Flags().Bool("cascade", false, "Delete along with subtasks")
Aliases: []string{"remove", "rm", "del"}
```

---

## Atlas CLI Current State

**Problems:**
- Flat `cmd/` structure - all commands in single package
- `page.go` is 17KB monolith (540+ lines)
- No confirmation for destructive operations
- No spinner feedback for slow operations
- No color output

**What Works:**
- `internal/cmdutil/` exists with basic utilities
- API layer is reasonably structured
- Parent resolution (ID/title/URL) implemented

---

## Recommended Architecture

### Phase 1: Command Restructure (Current Priority)

```
pkg/cmd/
├── factory/factory.go      # Optional: DI container
├── page/
│   ├── page.go             # Root command
│   ├── list/list.go
│   ├── view/view.go
│   ├── create/create.go
│   ├── edit/edit.go
│   ├── delete/delete.go
│   ├── children/children.go
│   └── shared/
│       ├── lookup.go       # resolveParentPage, parseConfluenceURL
│       └── display.go      # formatPage, tabwriter helpers
├── pr/
│   ├── pr.go
│   ├── list/
│   ├── view/
│   ├── merge/
│   └── shared/
├── issue/
│   ├── issue.go
│   ├── list/
│   ├── view/
│   └── shared/
└── root/root.go

internal/cmdutil/
├── errors.go       # ExitIfError (existing)
├── constants.go    # (existing)
├── output.go       # Success, Fail, Warn, Info (spinner) [NEW]
├── prompt.go       # ConfirmDeletion, AskInput [NEW]
└── truncate.go     # (existing)
```

### Phase 2: Enhanced cmdutil

```go
// internal/cmdutil/output.go
func Success(msg string, args ...interface{})  // Green ✓
func Fail(msg string, args ...interface{})     // Red ✗
func Warn(msg string, args ...interface{})     // Yellow
func Info(msg string) *spinner.Spinner         // Spinner for long ops

// internal/cmdutil/prompt.go
func ConfirmDeletion(itemDesc string) error    // "Type 'pageID' to confirm"
func Confirm(msg string) (bool, error)         // y/n prompt
```

### Phase 3: Factory Pattern (Optional)

Only if we need proper test coverage. Current scale doesn't require it.

---

## Delete Command Design

```go
// pkg/cmd/page/delete/delete.go

var deleteCmd = &cobra.Command{
    Use:     "delete {<id> | <title> | <url>}",
    Short:   "Delete a Confluence page",
    Aliases: []string{"rm", "del", "remove"},
    Args:    cobra.ExactArgs(1),
    RunE:    runDelete,
}

// Flags:
// --yes, -y     Skip confirmation prompt
// --cascade     Delete page and all descendants (future)

// Behavior:
// 1. Resolve page (ID/title/URL)
// 2. Fetch page info (title, children count)
// 3. If has children, warn and require --cascade or abort
// 4. If interactive && !yes: prompt "Type 'PageTitle' to confirm deletion"
// 5. Delete via API
// 6. Print success with URL that was deleted
```

---

## API Additions Needed

```go
// api/confluence.go

// DeletePage deletes a page by ID
func (c *ConfluenceClient) DeletePage(ctx context.Context, pageID string) error

// GetPageInfo returns page metadata without body (faster)
func (c *ConfluenceClient) GetPageInfo(ctx context.Context, pageID string) (*Content, error)
```

---

## Confluence Delete API

```
DELETE /rest/api/content/{id}
DELETE /rest/api/content/{id}?status=trashed  # Move to trash first
```

Note: Confluence Server vs Cloud may differ. Server typically allows direct delete,
Cloud may require trash then purge.

---

## Decision Log

| Date | Decision | Rationale |
|------|----------|-----------|
| 2024-12-03 | Keep flat cmd/ for now, refactor incrementally | Avoid big-bang refactor, ship delete first |
| 2024-12-03 | Add --yes flag, require confirmation by default | gh-cli pattern, prevents accidents |
| 2024-12-03 | Support ID/title/URL for delete target | Consistent with create --parent |
| 2024-12-03 | Skip Factory pattern for now | Overkill at current scale |

---

## Bitbucket PR Commands Analysis (2024-12-18)

### Current atl pr Commands (IMPLEMENTED)

```
atl pr
├── list        # List PRs (--state, --author, --limit, --json, --web)
├── view        # View PR details (--json, --web)
├── diff        # View raw diff
├── comment     # Add comment
├── status      # PRs relevant to you (created/reviewing)
├── merge       # Merge PR (--force, --delete-branch)
├── create      # Create PR (--title, --body, --base, --head, --reviewer, --fill, --web)
├── checkout    # Checkout PR locally (-b, --detach, -f) [alias: co]
├── review      # Review PR (--approve, --request-changes, --comment, --body)
├── approve     # Approve PR
├── unapprove   # Remove approval
├── decline     # Decline/close PR (--comment) [alias: close]
├── reopen      # Reopen declined PR
├── edit        # Edit PR (--title, --body, --base, --add-reviewer, --remove-reviewer)
├── rebase      # Server-side rebase
├── commits     # List PR commits (--limit, --json)
├── files       # List changed files (--limit, --json) [alias: changes]
├── activity    # Show PR activity (--limit, --json)
└── can-merge   # Check merge status (--json)
```

### gh pr Commands (Reference)

```
gh pr
├── create      # ← atl: ✓
├── list        # ← atl: ✓
├── view        # ← atl: ✓
├── diff        # ← atl: ✓
├── status      # ← atl: ✓
├── checkout    # ← atl: ✓
├── checks      # GitHub Actions specific, skip
├── close       # ← atl: "decline"
├── comment     # ← atl: ✓
├── edit        # ← atl: "update metadata"
├── merge       # ← atl: ✓
├── ready       # draft PR concept
├── reopen      # ← atl: ✓
├── review      # ← atl: approve/unapprove/needs-work
├── lock/unlock # not in BB API
└── update-branch # ← atl: `rebase`
```

### Bitbucket Server API Capabilities

From `docs/api-specs/bitbucketserver.906.postman.json`:

| Endpoint | Purpose | atl Status |
|----------|---------|------------|
| Create pull request | POST .../pull-requests | ✓ |
| Get pull request | GET .../pull-requests/{id} | ✓ |
| Update pull request metadata | PUT .../pull-requests/{id} | ✓ |
| Delete pull request | DELETE .../pull-requests/{id} | missing |
| Approve pull request | POST .../participants | ✓ |
| Unapprove pull request | DELETE .../participants | ✓ |
| Decline pull request | POST .../decline | ✓ |
| Re-open pull request | POST .../reopen | ✓ |
| Merge pull request | POST .../merge | ✓ |
| Rebase pull request | POST .../rebase | ✓ |
| Test if can merge | GET .../merge | ✓ |
| Get pull request commits | GET .../commits | ✓ |
| Gets pull request changes | GET .../changes | ✓ |
| Get pull request activity | GET .../activity | ✓ |
| Stream raw diff | GET .../diff | ✓ |
| Change participant status | PUT .../participants/{user} | ✓ |
| Add comment (general/file/line/reply/task) | POST .../comments | ✓ |
| Get comments for a path | GET .../comments?path= | ✓ |
| Structured diff (JSON) | GET .../diff (Accept: json) | ✓ |
| Get pending review | GET .../review | ✓ |
| Discard pending review | DELETE .../review | ✓ |
| Complete (publish) review | PUT .../review | missing (UI only) |
| Edit/delete a comment | PUT/DELETE .../comments/{id} | missing |
| Watch/Unwatch | POST/DELETE .../watch | missing |
| Auto-merge settings | GET/POST/DELETE .../auto-merge | missing |

### Feature Matrix (Updated)

```
Command          gh    atl   BB API   Status
─────────────────────────────────────────────────
create           ✓     ✓     ✓        DONE
checkout         ✓     ✓     -        DONE
approve/review   ✓     ✓     ✓        DONE
decline          ✓     ✓     ✓        DONE
reopen           ✓     ✓     ✓        DONE
edit             ✓     ✓     ✓        DONE
commits          ✓     ✓     ✓        DONE
files/changes    ✓     ✓     ✓        DONE
rebase           -     ✓     ✓        DONE
can-merge        -     ✓     ✓        DONE
activity         -     ✓     ✓        DONE
watch            -     ✗     ✓        TODO (low priority)
auto-merge       ✓     ✗     ✓        TODO (low priority)
```

### gh Patterns Worth Adopting

1. **`--json <fields>`** - Scriptable output
   ```bash
   atl pr list --json id,title,author | jq '.[] | select(.author=="me")'
   ```

2. **`--web` / `-w`** - Open in browser
   ```bash
   atl pr view 123 --web  # opens Bitbucket URL
   ```

3. **Review workflow flags** - Mutually exclusive
   ```bash
   atl pr review 123 --approve
   atl pr review 123 --request-changes -b "fix the tests"
   atl pr review 123 --comment -b "LGTM"
   ```

4. **Smart arg parsing** - Accept ID, URL, or branch
   ```bash
   atl pr view 123
   atl pr view https://bitbucket.example.com/.../pull-requests/123
   atl pr view feature/foo  # find PR by source branch
   ```

5. **Content input patterns**
   - `-t/--title`, `-b/--body` direct flags
   - `-F/--body-file` for file input (stdin with `-`)
   - `--fill` auto-fill from commits
   - `-e/--editor` interactive editor

### API Methods to Add

```go
// api/bitbucket.go

func (c *BitbucketClient) ApprovePullRequest(ctx, project, repo string, prID, version int) error
func (c *BitbucketClient) UnapprovePullRequest(ctx, project, repo string, prID, version int) error
func (c *BitbucketClient) SetReviewerStatus(ctx, project, repo string, prID int, user, status string) error
func (c *BitbucketClient) DeclinePullRequest(ctx, project, repo string, prID, version int) error
func (c *BitbucketClient) ReopenPullRequest(ctx, project, repo string, prID, version int) error
func (c *BitbucketClient) UpdatePullRequest(ctx, project, repo string, prID int, opts UpdatePROptions) error
func (c *BitbucketClient) GetPullRequestCommits(ctx, project, repo string, prID int) ([]Commit, error)
func (c *BitbucketClient) GetPullRequestChanges(ctx, project, repo string, prID int) ([]Change, error)
func (c *BitbucketClient) CanMerge(ctx, project, repo string, prID int) (bool, error)
func (c *BitbucketClient) RebasePullRequest(ctx, project, repo string, prID, version int) error
```

### Command Designs

**pr create**
```
atl pr create [project/repo]
  -t, --title        PR title (required unless --fill)
  -b, --body         PR description
  -B, --base         Target branch (default: main/master)
  -H, --head         Source branch (default: current branch)
  -r, --reviewer     Add reviewer (repeatable)
  -a, --assignee     Add assignee (repeatable)
  -d, --draft        Create as draft
  -w, --web          Open in browser after creation
  --fill             Auto-fill title/body from commits
```

**pr checkout**
```
atl pr checkout <pr-id>
  # git fetch origin pull/{id}/head:pr-{id} && git checkout pr-{id}
  # or for BB: git fetch origin refs/pull-requests/{id}/from:pr-{id}
```

**pr review**
```
atl pr review <pr-id>
  -a, --approve           Approve the PR
  -r, --request-changes   Request changes (sets NEEDS_WORK)
  -c, --comment           Add review comment only
  -b, --body              Review comment text
  -F, --body-file         Read body from file
```

**pr decline**
```
atl pr decline <pr-id>
  -c, --comment    Decline reason
```

**pr edit**
```
atl pr edit <pr-id>
  -t, --title       New title
  -b, --body        New description
  -B, --base        Change target branch
  --add-reviewer    Add reviewer
  --remove-reviewer Remove reviewer
```

### Bitbucket API Notes

**Participant Status Values:**
- `UNAPPROVED` - default, no action taken
- `APPROVED` - approved the PR
- `NEEDS_WORK` - requested changes

**PR Ref Format:**
```
refs/pull-requests/{id}/from   # source branch
refs/pull-requests/{id}/merge  # merge preview
```

**Version Field:**
All mutating operations require `version` from GET response (optimistic locking).

---

## Decision Log

| Date | Decision | Rationale |
|------|----------|-----------|
| 2024-12-03 | Keep flat cmd/ for now, refactor incrementally | Avoid big-bang refactor, ship delete first |
| 2024-12-03 | Add --yes flag, require confirmation by default | gh-cli pattern, prevents accidents |
| 2024-12-03 | Support ID/title/URL for delete target | Consistent with create --parent |
| 2024-12-03 | Skip Factory pattern for now | Overkill at current scale |
| 2024-12-18 | Priority: pr create > checkout > review > decline | Completes daily workflow |
| 2024-12-18 | Adopt gh --json/--web patterns | Scriptability + browser fallback |
| 2024-12-18 | Skip lock/unlock, checks (GitHub-specific) | Not in BB API |
| 2024-12-18 | Implemented all high/medium priority PR commands | 19 total subcommands |

---

## Implementation Summary (2024-12-18)

### New Files Created
- `cmd/pr_create.go` - Create PR with --fill, --reviewer, --web
- `cmd/pr_checkout.go` - Local checkout with alias `co`
- `cmd/pr_review.go` - Review/approve/unapprove commands
- `cmd/pr_lifecycle.go` - Decline, reopen, rebase commands
- `cmd/pr_edit.go` - Edit title/body/reviewers
- `cmd/pr_info.go` - Commits, files, activity, can-merge commands
- `cmd/browser.go` - Cross-platform `--web` opener

### API Methods Added (api/bitbucket.go)
- `ApprovePullRequest`, `UnapprovePullRequest`
- `SetReviewerStatus` (NEEDS_WORK support)
- `DeclinePullRequest`, `ReopenPullRequest`
- `UpdatePullRequest` with UpdatePROptions
- `GetPullRequestCommits`, `GetPullRequestChanges`
- `GetPullRequestActivity`
- `CanMerge`, `RebasePullRequest`
- `AddReviewer`, `RemoveReviewer`
- `GetDefaultBranch`
- `DeleteBranch` (Branch Utils)

### Patterns Applied
- `--json` output on list/view/commits/files/activity/can-merge
- `--web` flag to open in browser
- Consistent arg parsing: `[project/repo] <pr-id>`
- Mutually exclusive flags for review actions

### Post-implementation Fixes
See `docs/devlog/2024-12-18-pr-commands.org` for detailed lessons learned.

Summary:
1. No lying flags - implement or delete
2. Safe git ops - never merge into current branch
3. Exclusive output modes - --web vs --json
4. HTTP hygiene - close bodies, set Accept header
5. Flag validation - enforce required combinations
6. Cross-platform browser - OS detection
7. Branch normalization - handle refs/heads/ prefix

---

---

## Inline PR Comments (2026-07-24)

`atl pr comment` could only post general PR comments; every review had to
spell out `file:line` inside the comment text. Bitbucket Server anchors
comments through `anchor: {path, line, lineType, fileType, diffType,
fromHash, toHash}`, and the anchor must match the diff or the comment lands
orphaned.

**Anchor resolution.** Callers pass a path and a line number as they read it
in the file. `api/bitbucket_diff.go` fetches the structured diff
(`GET .../diff` with `Accept: application/json`) and derives lineType and
fileType from the segment the line falls in. Two facts the fixture encodes,
both observed on the live server:

- Added lines carry a `source` number and removed lines a `destination`
  number, so side selection must filter on segment type, not on which line
  number is populated.
- Only lines inside the diff (changed plus context) can be anchored. The
  resolver reports the file's commentable ranges instead of posting an
  anchor the server would orphan. Resolution retries once with a wider
  context before failing.

**Safety.** Batches resolve every anchor before posting the first comment, so
a bad path cannot half-post a review. `--dry-run` prints resolved anchors.
`--pending` drafts comments that only the author sees, which is the right
default for agent-authored reviews: a public comment notifies every reviewer
and cannot be unsent. Publishing a pending review (`PUT .../review`) is left
to the web UI; the payload is undocumented in the bundled spec and is not
worth guessing against a live team PR.

| Date | Decision | Rationale |
|------|----------|-----------|
| 2026-07-24 | Resolve lineType/fileType from the diff, not from flags | Callers know the file, not the diff hunk shape |
| 2026-07-24 | Fail on unanchorable lines instead of posting | Server silently orphans bad anchors |
| 2026-07-24 | Resolve whole batch before posting any comment | Partial reviews are worse than none |
| 2026-07-24 | Ship `--pending` + `--discard-pending`, not publish | Verifiable without notifying a team PR |
| 2026-07-24 | Read comments via activities, not per-path | One request covers general, inline and replies |

---

## Comment Lifecycle (2026-07-27)

`atl pr comment --edit <id>` and `--delete <id>` close the loop opened by
`--pending`: agent-drafted review comments can now be corrected or withdrawn
without the web UI. Both fetch the comment first for its `version`
(optimistic locking; the server answers 409 on a stale version, and delete
409s on comments with replies). Verified live against a pending review
comment (v0 -> v1 edit).

| Date | Decision | Rationale |
|------|----------|-----------|
| 2026-07-27 | Flags on `pr comment`, not a subcommand | `comment edit` would collide with the positional text argument |
| 2026-07-27 | Edit replaces text only | Severity/state changes are separate concerns; smallest correct surface |

## Next Steps

1. Add `page delete` command with confirmation
2. Add `cmdutil.Confirm()` and spinner helpers
3. Publish pending review (`PUT .../review`) once the payload is confirmed
5. Add `pr watch`/`unwatch` (low priority)
6. Add `pr auto-merge` settings (low priority)
7. Consider restructure after more commands exist
