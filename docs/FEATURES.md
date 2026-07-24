# Features

What works right now.

---

## Confluence

| Command | Description |
|---------|-------------|
| `atl page list [space]` | List pages in a space |
| `atl page search <query>` | Search with CQL |
| `atl page view <id>` | View page content |
| `atl page create` | Create new page |
| `atl page edit <id>` | Update page |
| `atl page delete <id>` | Delete page (with `--cascade`) |
| `atl page children <id>` | List child pages |
| `atl page spaces` | List all spaces |

**View options:**
- `--format markdown` - Convert to markdown
- `--format storage` - Raw Confluence XHTML
- `--format html` - Rendered HTML
- `--with-images` - Download images locally
- `--with-toc` - Add table of contents
- `-o file.md` - Save to file

**Create/Edit:**
- `-s, --space` - Space key
- `-t, --title` - Page title
- `-c, --content` - Inline content
- `-f, --content-file` - Content from file
- `-p, --parent` - Parent page (ID, title, or URL)

**Delete:**
- `--cascade` - Delete with all children
- `-y, --yes` - Skip confirmation

**Search filters:**
- `--space`, `--type`, `--title`
- `--creator`, `--contributor`
- `--modified`, `--created`
- `--cql` - Raw CQL query

---

## JIRA

| Command | Description |
|---------|-------------|
| `atl issue list` | List issues (JQL) |
| `atl issue view <key>` | View issue details |
| `atl issue transition <key> <status>` | Change issue status |
| `atl issue comment <key> <text>` | Add comment |
| `atl issue comments <key>` | List comments |
| `atl issue prs <key>` | Show linked PRs |

**List filters:**
- `--assignee` - Filter by assignee (`me` for self)
- `--project` - Filter by project
- `--status` - Filter by status
- `--limit` - Max results

**The `issue prs` command** shows all Bitbucket PRs linked to a JIRA issue. Cross-service integration.

---

## Bitbucket

| Command | Description |
|---------|-------------|
| `atl pr list [project/repo]` | List pull requests |
| `atl pr view <project/repo> <id>` | View PR details |
| `atl pr diff <project/repo> <id>` | Show PR diff |
| `atl pr comment <project/repo> <id> <text>` | Add comment, general or inline |
| `atl pr comments <project/repo> <id>` | List comments grouped by file and line |
| `atl pr merge <project/repo> <id>` | Merge PR |
| `atl pr status` | Show PR status summary |

**List filters:**
- `--state` - OPEN, MERGED, DECLINED, ALL
- `--author` - Filter by author (`@me` for self)
- `--base` - Filter by base branch
- `--head` - Filter by head branch
- `--limit` - Max results

**Merge options:**
- `--force` - Merge without approvals
- `--delete-branch` - Delete source branch after merge

### Inline review comments

`atl pr comment` anchors a comment to a line of the diff when given `--file`
and `--line`, the same way the web UI does. The line type (ADDED, REMOVED,
CONTEXT) and the file side are resolved from the pull request diff, so a path
and a line number are enough:

```bash
atl pr comment MYPROJ/myrepo 140 -f src/app.js -L 543 -b "this allocation leaks"
atl pr comment MYPROJ/myrepo 140 -f src/app.js -L 120 --side old -b "why drop this?"
atl pr comment MYPROJ/myrepo 140 --reply 331 -b "fixed in a1b2c3d"
```

- `--line` counts in the new file; `--side old` targets a deleted line.
- Only lines the diff touches can be anchored; the error lists the commentable
  ranges of the file.
- `--blocker` posts a task that blocks the merge.
- `--pending` keeps the comment unpublished, visible only to you until you
  publish the review from the pull request page. `atl pr review
  --discard-pending` drops the drafts.
- `--batch findings.json` posts a whole review at once, and `--dry-run` prints
  the resolved anchors without posting. Every anchor is resolved before the
  first comment is posted, so a bad path or line cannot half-post a batch.

Batch entries are `{body, file, line, side, reply_to, blocker, pending}`:

```json
[
  {"file": "src/app.js", "line": 543, "body": "this allocation leaks", "blocker": true},
  {"file": "src/app.js", "line": 120, "side": "old", "body": "why drop this?"},
  {"body": "NAK, see inline"}
]
```

`atl pr comments` reads a review back, grouped by file and line, with the IDs
`--reply` takes and `[TASK]`, `[PENDING]` and `[ORPHANED]` markers.

---

## Configuration

**File:** `~/.config/atlas/config.yaml`

```yaml
username: your.username

confluence:
  server: https://confluence.company.com
  token: your-bearer-token
  default_space: MYSPACE

jira:
  server: https://jira.company.com
  token: your-bearer-token
  default_project: PROJ

bitbucket:
  server: https://git.company.com
  token: your-api-token
  default_project: PROJ
  default_repo: repo-name
```

**Environment override:** `ATLAS_CONFLUENCE_TOKEN`, `ATLAS_JIRA_SERVER`, etc.

**Init:** `atl init` creates template config.

---

## Cross-Service

- **`atl issue prs`** - JIRA issue → linked Bitbucket PRs
- **Shared config** - One file for all services
- **Flexible resolution** - Page ID, title, or URL all work

---

## Output

- **Table format** - Default for lists
- **Markdown export** - `--format markdown`
- **Image download** - `--with-images` extracts attachments
- **TOC generation** - `--with-toc` adds navigation

---

## Auth

- Bearer token auth (Confluence, JIRA)
- Basic auth with API token (Bitbucket)
- Tokens stored in config YAML (keyring coming in future)
