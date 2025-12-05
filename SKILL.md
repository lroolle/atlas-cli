## ATL - CLI for Atlassian Server (Jira/Confluence/Bitbucket)

### Jira
atl issue view MYPROJ-123
atl issue list --assignee me --project MYPROJ
atl issue transition MYPROJ-123 "Start Progress"  # or "Resolve Issue", "Close Issue"
atl issue prs MYPROJ-456                          # Show linked PRs

### Confluence

#### Page Commands
```bash
# List & Search
atl page list '~john.doe'                         # List pages in space
atl page search "meeting notes" -s '~john.doe'    # Text search
atl page search --title "Atlas CLI" -s DOCS       # Search by title
atl page search --creator john.doe --modified month  # Filter by creator/date
atl page search -q 'type=page AND title~"API"'    # Raw CQL query
atl page find "rust" -s '~john.doe'               # Alias for search
atl page spaces --limit 20                         # List spaces
atl page children 110330423                        # List child pages

# View
atl page view 252088815 --info                     # Page metadata
atl page view 252088815 --format markdown          # View as markdown
atl page view 252088815 -o notes.md --with-images  # Export with images

# Create & Edit
atl page create -s '~john.doe' -t "Title" -c "<p>content</p>"
atl page create -s '~john.doe' -t "Title" -f content.html -p "Notes"  # Under parent
atl page create -s DOCS -t "Design Doc" -f design.html
atl page edit 252088815 -t "New Title" -c "<p>new content</p>"
atl page edit 252088815 -f updated.html

# Delete (NEW)
atl page delete 252088815 --yes                    # By ID
atl page delete "Page Title" -s '~john.doe' -y   # By title
atl page delete "https://confluence.../display/~john.doe/Notes" -y  # By URL
atl page rm 252088815 -y                           # Alias: rm, del, remove
```

#### Parent Page Resolution
The `--parent` / `-p` flag accepts:
- Numeric ID: `-p 110330423`
- Page title: `-p "Notes"` (requires space context)
- Full URL: `-p "https://confluence.example.com/display/~john.doe/Notes"`

#### Search Filters
```bash
--space, -s      # Space key
--type, -t       # page, blogpost, comment, attachment (default: page)
--title          # Title contains
--creator        # Creator username
--contributor    # Contributor username
--modified       # today, yesterday, week, month, year
--created        # today, yesterday, week, month, year
--cql, -q        # Raw CQL query (overrides other filters)
--order-by       # created, lastmodified, title (default: lastmodified)
--reverse        # Ascending order
```

#### Confluence Page Guidelines

**MUST DO before editing:**
1. MUST read the page before any update: `atl page view <page-id> --format storage -o page.html`
2. MUST maintain the original page structure (panels, macros, tables)
3. MUST not delete content without explicit permission
4. For new pages, check if similar page exists first

**NOTES PAGE**
- Title format: `YYYY-MM-DD: Title` (consistent with existing notes)
- URL: https://confluence.example.com/display/~john.doe/Notes

**Standard Page Structure:**
```
{Page TOC}

{Overview / Background / Context / Current State}

---
{Section A}

{Section B}

{Section C} ...

---

{Summary / Conclusion}

---

{Reference Links}

---
```

- Keep it practical and concise - no bureaucratic overhead
- Focus on essential information team needs
- Use zh-hans for internal documentation
- Avoid excessive subsections and checklists unless explicitly requested

**Content Rules - What NOT to include:**
- NO planning phases (Phase 1, Phase 2, etc.) unless explicitly requested
- NO timelines, deadlines, or finish date estimates
- NO working hours assumptions (e.g., "this will take 2 weeks")
- NO progress tracking sections (sprints, milestones, etc.)
- NO project management overhead (RACI, approval flows, etc.)
- Focus on WHAT needs to be done, not WHEN or HOW LONG
- Let users decide their own scheduling and effort estimates

**Technical Notes:**
- Confluence storage format: use `<pre>` for code blocks (CDATA in file upload causes issues)
- No duplicate titles in same space
- atl has no move command - move pages manually in UI if needed

**Common atl CLI mistakes to avoid:**

```bash
# ❌ WRONG - Tilde without quotes (shell expansion)
atl page create -s ~john.doe -t "Title" -c "<p>test</p>"

# ✅ RIGHT - Quote the tilde space key
atl page create -s '~john.doe' -t "Title" -c "<p>test</p>"

# ❌ WRONG - Space as flag for page list (it's positional)
atl page list --space '~john.doe'

# ✅ RIGHT - Space as positional argument
atl page list '~john.doe'

# ❌ WRONG - Piping with tilde causes shell expansion issues
atl page list '~john.doe' | grep "notes"

# ✅ RIGHT - Use search instead of grep
atl page search "notes" -s '~john.doe'

# ❌ WRONG - CDATA in structured-macro fails with -f flag
<ac:structured-macro ac:name="code">
<ac:plain-text-body><![CDATA[code]]></ac:plain-text-body>
</ac:structured-macro>

# ✅ RIGHT - Use simple <pre> tags for code blocks
<pre>code here</pre>

# ❌ WRONG - Duplicate titles in same space (causes 400)
# If page "2025-11-24: Notes" exists, creating another fails

# ✅ RIGHT - Check for existing pages first
atl page search --title "2025-11-24" -s '~john.doe'
```

**Why these fail:**
- Tilde `~` triggers shell home directory expansion if unquoted
- CDATA `<![CDATA[...]]>` gets mangled in bash heredocs/files
- Confluence API rejects duplicate titles (400 error)
- Some atl commands use positional args, not flags
- Pipe with grep can trigger shell expansion on unquoted vars

### Bitbucket PRs
atl pr list PROJ/my-repo --state OPEN
atl pr view PROJ/my-repo 140
atl pr diff PROJ/other-repo 85
atl pr list OTHER/core-lib

#### Common repos: my-repo, other-repo, web-app, api-service, infra-tools
