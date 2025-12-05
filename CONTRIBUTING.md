# Contributing to atlas-cli

Get set up and ship code in 5 minutes. No ceremony.

## Quick Start

```bash
git clone https://github.com/lroolle/atlas-cli.git
cd atlas-cli
make build      # Builds ./atl
make test       # Runs tests
./atl --help    # Verify it works
```

Requirements: Go 1.22+

## Running Tests

```bash
make test                    # All tests
go test ./pkg/cmd/page/...  # Specific package
go test -v -run TestFoo     # Specific test
```

If tests need auth, copy `test-config.yaml` to `~/.config/atlas/config.yaml` with real credentials.
Most tests don't - we test logic, not API integration.

## Code Structure

```
cmd/                # Legacy flat structure (being phased out)
├── root.go
├── pr.go          # Bitbucket PR commands
└── jira.go        # Basic JIRA commands

pkg/cmd/           # NEW structure - put new commands here
└── page/
    ├── page.go           # Root command
    ├── create/create.go  # Each action in own package
    ├── delete/delete.go
    ├── view/view.go
    └── shared/
        ├── lookup.go     # Common utilities
        └── client.go

api/               # API clients for Atlassian services
├── confluence.go  # Confluence REST API
├── bitbucket.go   # Bitbucket Cloud API
└── jira.go        # JIRA API (minimal - not our focus)

internal/cmdutil/  # Shared command utilities
├── errors.go      # Error handling
└── constants.go   # Shared constants
```

**Pattern:**
- New commands go in `pkg/cmd/{resource}/{action}/`
- API calls stay in `api/`
- Shared logic for a resource goes in `pkg/cmd/{resource}/shared/`
- Don't touch `cmd/` - legacy code, refactor later

## Adding a New Command

Example: Adding `atl page archive`

1. Create the package:
```bash
mkdir -p pkg/cmd/page/archive
touch pkg/cmd/page/archive/archive.go
```

2. Write the command:
```go
package archive

import (
	"github.com/lroolle/atlas-cli/api"
	"github.com/lroolle/atlas-cli/pkg/cmd/page/shared"
	"github.com/spf13/cobra"
)

func NewCmdArchive() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "archive {<id> | <title> | <url>}",
		Short: "Archive a Confluence page",
		Args:  cobra.ExactArgs(1),
		RunE:  runArchive,
	}
	return cmd
}

func runArchive(cmd *cobra.Command, args []string) error {
	client, err := shared.GetConfluenceClient()
	if err != nil {
		return err
	}

	pageID, err := shared.LookupPageID(cmd.Context(), client, args[0])
	if err != nil {
		return err
	}

	// Do the thing
	return client.ArchivePage(cmd.Context(), pageID)
}
```

3. Register in parent command (`pkg/cmd/page/page.go`):
```go
import "github.com/lroolle/atlas-cli/pkg/cmd/page/archive"

func NewCmdPage() *cobra.Command {
	// ... existing code ...
	cmd.AddCommand(archive.NewCmdArchive())
	return cmd
}
```

4. Add API method if needed (`api/confluence.go`):
```go
func (c *ConfluenceClient) ArchivePage(ctx context.Context, pageID string) error {
	// API implementation
}
```

5. Write a test (`pkg/cmd/page/archive/archive_test.go`):
```go
func TestArchive_ValidID(t *testing.T) {
	// Test logic, not API calls
}
```

Done. Run `make build && ./atl page archive 12345`.

## PR Guidelines

**Before submitting:**
- `make test` passes
- Tested manually (we don't have CI yet, your word is gold)
- If adding a command, update `docs/USAGE.md` with the new command

**PR description:**
```
What: Added `atl page archive` command
Why: Users need to archive pages without opening browser
Tested: Archived test page 12345 in my sandbox space
```

No templates. Just tell us what you did and why.

**We merge fast.** If tests pass and it makes sense, it goes in.

## What We Want

### High Priority
- Confluence features (archive, move, permissions, labels)
- Bitbucket improvements (view PR, approve, comments)
- Better output formatting (colors, tables)
- Markdown → Confluence storage format improvements
- Bug fixes

### Nice to Have
- JIRA issue linking (viewing, not workflow)
- Attachments support
- Bulk operations
- Shell completion

### Not Interested
- Full JIRA workflow client (jira-cli does this)
- JIRA Server support (Cloud only)
- Enterprise SSO (too complex, PRs welcome but we won't maintain it)
- Web UI replacement features

## What We Don't Want

- Bloat. This is a CLI tool, not an IDE plugin.
- Breaking changes without migration path.
- Dependencies on heavy frameworks. Keep it lean.
- Code that you wouldn't want to debug at 2am.

## Code Style

Follow Go conventions:
- `gofmt` your code (it's 2024, this should be automatic)
- Meaningful names: `pageID` not `pid`, `client` not `c`
- Comments explain WHY, not WHAT
- If it needs a comment to understand WHAT it does, refactor it

**Error messages:**
```go
BAD:  return fmt.Errorf("error")
GOOD: return fmt.Errorf("failed to archive page %s: %w", pageID, err)
```

**Flags:**
- Long form: `--space`, `--title`
- Short form for common ones: `-s`, `-t`
- Boolean flags default to false

## Testing Philosophy

Test logic, not API calls.

```go
// Good - tests argument parsing logic
func TestLookupPageID_ParsesConfluenceURL(t *testing.T) {
	id := extractPageIDFromURL("https://company.atlassian.net/wiki/spaces/DOC/pages/12345")
	if id != "12345" {
		t.Errorf("expected 12345, got %s", id)
	}
}

// Bad - tests Atlassian's API
func TestGetPage_ReturnsPage(t *testing.T) {
	client := NewClient()
	page, _ := client.GetPage(ctx, "12345")  // Makes real API call
	// ...
}
```

If you need to test API integration, mock it or add `// +build integration` tag.

## Questions?

Open an issue. We're friendly, just direct.

If something in this doc is wrong or unclear, fix it and send a PR.
The best documentation is the one that gets updated.

---

Built by developers who are tired of clicking through web UIs when they should be coding.
