---
name: atl-cli
description: >
  Atlassian Server CLI (Jira/Confluence/Bitbucket). This skill should be used
  when working with Jira issues, Confluence page operations, Bitbucket pull
  requests, or any Atlassian Server tasks. Triggers on: issue tracking, wiki
  pages, PR management, CQL queries.
---

# ATL CLI - Atlassian Server CLI

Command-line interface for Atlassian Server REST APIs: Jira, Confluence, Bitbucket.

## Jira Issues

```bash
atl issue list --assignee me --project MYPROJ
atl issue list --status "In Progress" --limit 20
atl issue view MYPROJ-123
atl issue transition MYPROJ-123 "Start Progress"
atl issue transition MYPROJ-123              # List available transitions
atl issue comment MYPROJ-123 "Comment text"  # Add comment
atl issue comments MYPROJ-123                # List comments
atl issue prs MYPROJ-456                     # Show linked PRs
```

## Confluence Pages

### List & Search
```bash
atl page list '~john.doe'                         # List pages in space
atl page search "meeting notes" -s '~john.doe'    # Text search
atl page search --title "Atlas CLI" -s DOCS       # By title
atl page search --creator john.doe --modified month
atl page search -q 'type=page AND title~"API"'    # Raw CQL
atl page find "rust" -s '~john.doe'               # Alias: find, query
atl page spaces --limit 20
atl page children 110330423
```

### View & Export
```bash
atl page view 252088815 --info                    # Metadata only
atl page view 252088815 --format markdown         # As markdown
atl page view 252088815 --format storage          # Confluence XHTML
atl page view 252088815 -o notes.md               # Save to file
atl page view 252088815 -o notes.md --with-images # Export with images
atl page view 252088815 -o notes.md --with-toc    # Add table of contents
```

### Create & Edit
```bash
atl page create -s '~john.doe' -t "Title" -c "<p>content</p>"
atl page create -s '~john.doe' -t "Title" -f content.html -p "Parent Page"
atl page edit 252088815 -t "New Title" -c "<p>new content</p>"
atl page edit 252088815 -f updated.html
```

### Delete
```bash
atl page delete 252088815 --yes                   # By ID
atl page delete "Page Title" -s '~john.doe' -y    # By title (needs -s)
atl page delete "https://confluence.../page" -y   # By URL
atl page delete 252088815 --cascade -y            # Delete with children
atl page rm 252088815 -y                          # Aliases: rm, del, remove
```

### Parent Resolution
`--parent` / `-p` accepts: numeric ID, page title (needs space context), or full URL.

### Search Flags
| Flag | Description |
|------|-------------|
| `-s, --space` | Space key |
| `-t, --type` | page, blogpost, comment, attachment |
| `--title` | Title contains |
| `--creator` | Creator username |
| `--modified` | today, yesterday, week, month, year |
| `-q, --cql` | Raw CQL (overrides others) |
| `--order-by` | created, lastmodified, title |

## Bitbucket PRs

```bash
atl pr list PROJ/repo --state OPEN
atl pr list PROJ/repo --author @me
atl pr view PROJ/repo 140
atl pr diff PROJ/repo 85
atl pr comment PROJ/repo 140 "LGTM"
atl pr merge PROJ/repo 140
atl pr merge PROJ/repo 140 --force             # Merge without approval
atl pr status                                   # Your PRs & review requests
```

## Critical Pitfalls

```bash
# WRONG - unquoted tilde triggers shell expansion
atl page create -s ~john.doe -t "Title" -c "<p>x</p>"
# RIGHT - quote the tilde
atl page create -s '~john.doe' -t "Title" -c "<p>x</p>"

# WRONG - page list uses positional arg, not --space
atl page list --space '~john.doe'
# RIGHT
atl page list '~john.doe'

# WRONG - CDATA breaks in file upload
<ac:plain-text-body><![CDATA[code]]></ac:plain-text-body>
# RIGHT - use <pre> for code blocks
<pre>code here</pre>
```

## Confluence Content Guidelines

Before editing pages, read `references/confluence-guidelines.md` for:
- Page structure templates
- Content rules (no timelines, no phases)
- Storage format gotchas
- Common mistakes

**Always read before edit:** `atl page view <id> --format storage -o page.html`
