# Command Reference

Full reference for every atlas-cli command. If you just want to get started, read the [main README](../README.md) instead.

## Global Flags

These work on any command:

```bash
--config FILE    Use config file (default: ~/.config/atlas/config.yaml)
--format FORMAT  Output format: text, json (where applicable)
--help, -h       Show help
```

---

## Confluence Pages

The main attraction. Full CRUD for Confluence pages.

### atl page list

List recent pages in a space.

```bash
atl page list MYSPACE
atl page list MYSPACE --limit 50
atl page list '~username'           # Personal space (quote the ~)
atl page list MYSPACE --format json
```

**Flags:**
- `--limit N` - Max pages to return (default: 25)
- `--format` - Output format: text, json

**Gotcha:** Takes space as positional arg, not `--space` flag.

### atl page search

Search pages by text, title, author, or modification date.

```bash
# Text search
atl page search "API design" -s MYSPACE
atl page search "meeting notes" -s MYSPACE --limit 10

# Title search (exact match)
atl page search --title "2024-12" -s MYSPACE

# Filter by author/date
atl page search --creator john.doe -s MYSPACE
atl page search --modified week -s MYSPACE
atl page search --modified month -s MYSPACE

# Combine filters
atl page search "architecture" --creator alice --modified week -s MYSPACE
```

**Flags:**
- `-s, --space SPACE` - Space to search (required)
- `--title TEXT` - Search by page title (exact match)
- `--creator USERNAME` - Filter by author
- `--modified WHEN` - Filter by date: week, month
- `--limit N` - Max results (default: 25)
- `--format` - Output format: text, json

### atl page view

View or export a page.

```bash
# View in terminal
atl page view 12345678

# Export as markdown
atl page view 12345678 --format markdown
atl page view 12345678 -o backup.md --format markdown

# Export with images downloaded
atl page view 12345678 -o full-backup.md --with-images

# Export as HTML (Confluence storage format)
atl page view 12345678 --format storage -o template.html

# Metadata only
atl page view 12345678 --info
```

**Flags:**
- `--format FORMAT` - Output format: text, markdown, storage
- `-o, --output FILE` - Write to file instead of stdout
- `--with-images` - Download embedded images (slow for large pages)
- `--info` - Show metadata only (title, author, URL, etc.)

**Note:** "storage" format is Confluence's XHTML. Use it to grab templates.

### atl page create

Create a new page from file or inline content.

```bash
# From HTML file
atl page create -s MYSPACE -t "New Page" -f content.html

# From inline content
atl page create -s MYSPACE -t "Quick Note" -c "<p>Content here</p>"

# With parent page (by title)
atl page create -s MYSPACE -t "Child Page" -f child.html -p "Parent Title"

# With parent page (by ID)
atl page create -s MYSPACE -t "Child Page" -f child.html -p 12345678

# With parent page (by URL)
atl page create -s MYSPACE -t "Child" -f child.html \
  -p "https://confluence.company.com/display/MYSPACE/Parent"
```

**Flags:**
- `-s, --space SPACE` - Space to create page in (required)
- `-t, --title TITLE` - Page title (required)
- `-f, --file FILE` - Read content from file
- `-c, --content HTML` - Inline content
- `-p, --parent ID|TITLE|URL` - Parent page (optional)

**Parent Resolution:**
- By ID: `-p 12345678`
- By title: `-p "Parent Page"` (searches in same space)
- By URL: `-p "https://confluence.../display/SPACE/Page"`

**Content Format:**
Confluence uses "storage format" (XHTML). Simple HTML works:
```html
<h1>Title</h1>
<p>Text with <strong>bold</strong> and <em>italic</em></p>
<ul><li>List item</li></ul>
<pre>Code block</pre>
```

**Avoid:**
- `<![CDATA[...]]>` - gets mangled by shell
- Duplicate titles in same space - API returns 400

### atl page edit

Update an existing page.

```bash
# Update title only
atl page edit 12345678 -t "New Title"

# Update content from file
atl page edit 12345678 -f updated.html

# Update both
atl page edit 12345678 -t "New Title" -f updated.html
```

**Flags:**
- `-t, --title TITLE` - New title (optional)
- `-f, --file FILE` - New content from file (optional)
- `-c, --content HTML` - New content inline (optional)

**Note:** At least one of `-t`, `-f`, or `-c` required.

### atl page delete

Delete a page.

```bash
# With confirmation prompt
atl page delete 12345678

# By title
atl page delete "Page Title" -s MYSPACE

# Skip confirmation (dangerous)
atl page delete 12345678 --yes
```

**Aliases:** `rm`, `del`, `remove`

**Flags:**
- `-s, --space SPACE` - Space (required if using title)
- `--yes, -y` - Skip confirmation prompt

**Warning:** Deletes are permanent on Confluence Server (Cloud has trash).

### atl page spaces

List available spaces.

```bash
atl page spaces
atl page spaces --limit 50
```

**Flags:**
- `--limit N` - Max spaces to return (default: 25)

---

## Bitbucket Pull Requests

Work with PRs without the slow web UI.

### atl pr list

List pull requests in a repository.

```bash
# All PRs
atl pr list PROJ/repo

# Open PRs only
atl pr list PROJ/repo --state OPEN

# Merged PRs
atl pr list PROJ/repo --state MERGED

# Use defaults from config
atl pr list
```

**Flags:**
- `--state STATE` - Filter by state: OPEN, MERGED, DECLINED, ALL (default: ALL)
- `--limit N` - Max PRs to return (default: 25)

### atl pr view

View PR details.

```bash
atl pr view PROJ/repo 123
atl pr view PROJ/repo 123 --format json
```

**Flags:**
- `--format` - Output format: text, json

### atl pr diff

Show PR diff.

```bash
atl pr diff PROJ/repo 123
```

### atl pr merge

Merge a pull request.

```bash
atl pr merge PROJ/repo 123
```

**Note:** Requires write permissions on the repository.

### atl pr status

Show PR status for current git repository.

```bash
atl pr status
```

Must be run inside a git repository with Bitbucket remote.

---

## JIRA Issues

Read-only JIRA viewing. For real JIRA work, use [jira-cli](https://github.com/ankitpokhrel/jira-cli).

### atl issue view

View issue details.

```bash
atl issue view PROJ-123
atl issue view PROJ-123 --format json
```

**Flags:**
- `--format` - Output format: text, json

### atl issue list

List issues.

```bash
# Your issues
atl issue list --assignee me

# Someone else's issues
atl issue list --assignee john.doe --project PROJ

# Filter by status
atl issue list --assignee me --status "In Progress"
```

**Flags:**
- `--assignee USER` - Filter by assignee (use "me" for yourself)
- `--project PROJ` - Filter by project
- `--status STATUS` - Filter by status
- `--limit N` - Max issues (default: 25)

### atl issue prs

Show PRs linked to an issue.

```bash
atl issue prs PROJ-123
```

**Note:** This is the killer feature for standup - see all PRs for a JIRA ticket.

### atl issue transition

Change issue status.

```bash
atl issue transition PROJ-123 "Start Progress"
atl issue transition PROJ-123 "Resolve Issue"
```

**Note:** Available transitions depend on your JIRA workflow. Use the web UI for complex workflows.

---

## Config Management

### atl init

Generate config file template.

```bash
atl init
# Creates ~/.config/atlas/config.yaml
```

Edit the generated file with your credentials. See [CONFIGURATION.md](CONFIGURATION.md) for details.

---

## Output Formats

Where `--format` is supported:

**text** (default)
- Human-readable tables
- Truncated for terminal width
- Uses tabwriter for alignment

**json**
- Full API response
- Useful for scripting
- Pipe to `jq` for processing

Example:
```bash
atl page search "meeting" -s MYSPACE --format json | \
  jq -r '.[].id' | \
  xargs -I {} atl page view {} -o "export/{}.md" --format markdown
```

---

## Common Patterns

### Bulk export
```bash
for id in $(atl page list MYSPACE --format json | jq -r '.[].id'); do
  atl page view $id -o "backup/$id.md" --format markdown
done
```

### Export with images
```bash
atl page view 12345678 -o full-doc.md --with-images --format markdown
```

### Create from template
```bash
# 1. Export template
atl page view 11111111 -o template.html --format storage

# 2. Edit template
vim template.html

# 3. Create new page
atl page create -s MYSPACE -t "2024-12-04: New Doc" -f template.html -p "Parent"
```

### Search and filter
```bash
# Find all pages by author from last week
atl page search --creator john.doe --modified week -s MYSPACE
```

---

## Shell Completion

Generate shell completion scripts:

```bash
# Bash
atl completion bash > /etc/bash_completion.d/atl

# Zsh
atl completion zsh > /usr/local/share/zsh/site-functions/_atl

# Fish
atl completion fish > ~/.config/fish/completions/atl.fish
```

Then restart your shell.

---

## Exit Codes

- `0` - Success
- `1` - Generic error (bad args, API failure, etc.)
- `2` - Config error (missing token, invalid config)

---

## See Also

- [Configuration](CONFIGURATION.md) - Config file, API tokens
- [Workflows](WORKFLOWS.md) - Real-world examples
- [Troubleshooting](TROUBLESHOOTING.md) - Common errors, gotchas
