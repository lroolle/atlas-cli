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

## Next Steps

1. Add `page delete` command with confirmation
2. Add `cmdutil.Confirm()` and spinner helpers
3. Consider restructure after more commands exist
