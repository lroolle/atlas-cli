---
name: atl-cli
description: >
  Atlassian Server CLI (Jira/Confluence/Bitbucket). This skill should be used
  when working with Jira issues, Confluence page operations, Bitbucket pull
  requests, or any Atlassian Server tasks. Triggers on: issue tracking, wiki
  pages, PR management, CQL queries.
---

# ATL CLI

```
atl
├── init                              # Setup config
├── issue (jira)
│   ├── list (ls, search) [text]      # -t -s[] -y -a -r -e -C -l[] -p -q --order-by --reverse --limit
│   ├── view [key]
│   ├── create                        # -t -s -P -y -a -e --sprint --story-points --field --json
│   ├── comment [key] [text]
│   ├── comments [key]
│   ├── transition (move, mv) [key] [name?]  # -R -F[] -T -m --fix-version --json
│   └── prs [key]                     # Linked pull requests
├── page (confluence)
│   ├── list [space]                  # --type --limit
│   ├── search (find, query) [text]   # -s -t --title --creator --contributor --created --modified -q --order-by --reverse --limit
│   ├── view [id]                     # --format --info -o --with-images --with-toc
│   ├── create                        # -s -t -c -f -p
│   ├── edit [id]                     # -t -c -f
│   ├── delete (rm, del) [id|title|url]  # -s --cascade -y
│   ├── children [id]                 # --limit
│   └── spaces                        # --limit
└── pr
    ├── list [proj/repo]              # --state --author --base --head --limit
    ├── view [proj/repo] [id]
    ├── diff [proj/repo] [id]
    ├── comment [proj/repo] [id] [text]  # -b -F -f -L --side --reply --blocker --pending --batch --dry-run
    ├── comments [proj/repo] [id]     # -f --pending --limit --json
    ├── review [proj/repo] [id]       # -a -r -c -b --discard-pending
    ├── merge [proj/repo] [id]        # --force --delete-branch
    └── status                        # Your PRs & reviews
```

Global: `--config`, `--username`

## Jira Issue List

```bash
atl issue list -a me -t Bug -s Open   # My open bugs
atl issue list -s '~Done' -e 123      # Not Done, epic auto-prefixed
atl issue list -q "created >= -7d"    # Raw JQL
```

| Flag | Values |
|------|--------|
| `-t` | Bug, Story, Task, Epic |
| `-s` | Status (multi, `~` negates) |
| `-y` | Blocker, Critical, Major, Minor, Trivial |
| `-a` | me, none, x, username |
| `-e` | Epic key (auto-prefixes project) |
| `--order-by` | created, updated, priority, status, key, assignee, reporter, summary |

## Jira Issue Create

```bash
# Story with epic/sprint/points (agile field IDs auto-resolved per instance)
atl issue create -t Story -s "story title" -e MYPROJ-100 --sprint 1946 --story-points 3

# Sub-task under a story (subtask type names are server-specific)
atl issue create -t Sub-task -P MYPROJ-123 -y Major -s "dev subtask"

atl issue create -t Bug -s "crash on empty input" -y Critical -a me
atl issue create -t Task -s "x" --field 'customfield_10001={"value":"internal"}' --json
```

Pitfalls: sub-tasks INHERIT sprint from parent (passing `--sprint` gets a
400 — omit it). Dedup first: `atl issue view <parent>` shows existing
subtasks. `--json` prints `{key,id,url}` for scripting.

## Jira Workflow Transitions

```bash
atl issue transition MYPROJ-123                    # List transitions + required fields
atl issue transition MYPROJ-123 --json             # Full field metadata (schema, allowed values)
atl issue transition MYPROJ-123 "Start Progress"   # By name, target status, id, or unique substring
atl issue transition MYPROJ-123 resolve -R Done --fix-version 1.2.0 -T 2h \
  -F "Root Cause=config error" -m "merged in PR #123"
```

| Flag | Purpose |
|------|---------|
| `-R` | Resolution (required on Resolve/Close) |
| `--fix-version` | Fix Version/s |
| `-T` | Log work, e.g. 2h, 30m (validators often demand Time Spent) |
| `-F` | Screen field by display name or id; values coerced from schema; repeatable; ASCII comma = multi-select separator; cascading selects as "Parent / Child" |
| `-m` | Comment posted with the transition |
| `--dry-run` | Print the exact POST body without transitioning |
| `--no-defaults` | Ignore jira.transition_defaults from config |

Workflow validators require fields the API never marks required — the
400 names them; add the flags and retry. Put team-constant field values
in `jira.transition_defaults` (config) so the CLI only carries what
varies per issue. Read `references/jira-workflow.md` for the discovery
methodology and config semantics BEFORE transitioning issues on an
instance whose workflow you haven't mapped.

## Confluence Page Search

```bash
atl page search "notes" -s '~john.doe'
atl page search --title "CLI" --creator john.doe --modified month
atl page search -q 'type=page AND title~"API"'
```

| Flag | Values |
|------|--------|
| `-t` | page, blogpost, comment, attachment |
| `--modified` | today, yesterday, week, month, year |
| `--order-by` | created, lastmodified, title |

## Confluence Page View/Create/Edit

```bash
atl page view 12345 --format storage -o page.html  # Fetch before edit
atl page create -s SPACE -t "Title" -f content.html -p "Parent"
atl page edit 12345 -f updated.html
atl page delete 12345 --cascade -y
```

`-p/--parent`: ID, title (needs space), or URL

## Bitbucket PR

```bash
atl pr list PROJ/repo --state ALL --author @me
atl pr merge PROJ/repo 140 --force --delete-branch
atl pr status
```

| `--state` | OPEN, MERGED, DECLINED, ALL |

## Inline Review Comments

Anchor a comment to a line of the diff. `--line` counts in the NEW file;
`--side old` targets a line the PR deletes. ADDED/REMOVED/CONTEXT and the
file side are resolved from the diff -- never hand-write them.

```bash
atl pr comment PROJ/repo 140 -f src/app.py -L 42 -b "this leaks"       # inline
atl pr comment PROJ/repo 140 -f src/app.py -L 42 --side old -b "why?"  # deleted line
atl pr comment PROJ/repo 140 -f src/app.py -b "file-level note"        # whole file
atl pr comment PROJ/repo 140 --reply 331 -b "fixed in a1b2c3d"     # thread reply
atl pr comment PROJ/repo 140 --blocker -f src/app.py -L 42 -b "..."    # merge-blocking task
atl pr comments PROJ/repo 140                                          # read the review back
```

Only lines the diff touches (changed lines plus surrounding context) can be
anchored -- the error lists the file's commentable ranges. Reply IDs come from
`atl pr comments`.

### Agent review workflow

Post a whole review from JSON. Anchors are all resolved before anything is
posted, so a bad path or line cannot half-post a batch.

```json
[
  {"file": "src/app.py", "line": 42, "body": "leaks here", "blocker": true},
  {"file": "src/app.py", "line": 17, "side": "old", "body": "why drop this?"},
  {"body": "NAK, see inline"}
]
```

```bash
atl pr comment PROJ/repo 140 --batch findings.json --dry-run   # check anchors first
atl pr comment PROJ/repo 140 --batch findings.json --pending   # draft, nobody notified
atl pr comments PROJ/repo 140 --pending                        # review the draft
atl pr review PROJ/repo 140 --discard-pending                  # drop the draft
```

`--pending` comments stay invisible to everyone else until published from the
PR page in the browser. Default to `--pending` for agent-authored reviews: a
public comment notifies every reviewer and cannot be unsent.

## Pitfalls

```bash
# Quote tilde - shell expansion
atl page list '~john.doe'      # RIGHT
atl page list ~john.doe        # WRONG

# Positional arg, not --space
atl page list '~john.doe'      # RIGHT
atl page list --space SPACE    # WRONG
```

## Confluence Pages -- Format Choice

**Default to storage format (.html) for pages with code blocks or macros.**
The markdown converter has known bugs: unescaped `&`, `<`, `>` in code fences
cause HTTP 400. Use CDATA in storage format to protect special chars:

```html
<ac:structured-macro ac:name="code">
  <ac:parameter ac:name="language">bash</ac:parameter>
  <ac:plain-text-body><![CDATA[PASS='foo&bar']]></ac:plain-text-body>
</ac:structured-macro>
```

Bare URLs are NOT auto-linked -- always use `<a href="...">text</a>`.

Read `references/confluence-guidelines.md` for full layout patterns, panel macros,
tables, and known bugs before creating or editing pages.
