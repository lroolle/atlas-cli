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
| `atl pr comment <project/repo> <id> <text>` | Add comment |
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
