# Confluence Page Guidelines

## Before Any Edit

1. **Read first**: `atl page view <page-id> --format storage -o page.html`
2. **Preserve structure**: Keep original panels, macros, tables intact
3. **No silent deletions**: Don't remove content without explicit permission
4. **Check duplicates**: Search before creating new pages

## Writing Pages -- Format Choice

`atl page create -f file` accepts markdown (.md) or Confluence storage format (.html).

**Default to storage format** for anything non-trivial. The markdown converter has
known bugs (unescaped `&`, `<`, `>` in code fences cause XML parse errors / HTTP 400)
and limited macro support. Markdown is fine for plain prose; storage format is required
when your page has:

- Code blocks containing `&`, `<`, `>`, or single quotes
- Warning/info/note panels
- Multi-column layouts
- Tables inside macros
- Anything beyond basic headings + paragraphs + lists

### Workflow

```bash
# 1. Write content in storage format
#    (see "Storage Format Reference" below)
vi page.html

# 2. Create
atl page create -s '~eric.wang' -t "Page Title" -f page.html -p PARENT_ID

# 3. Verify in browser
atl page view PAGE_ID --web    # or open the URL manually
```

## Storage Format Reference

### Minimal page skeleton

```html
<ac:structured-macro ac:name="info">
  <ac:rich-text-body><p>Key message at the top.</p></ac:rich-text-body>
</ac:structured-macro>

<h2>Section</h2>
<p>Paragraph text. <a href="http://example.com">Link text</a> for clickable URLs.</p>

<ac:structured-macro ac:name="code">
  <ac:parameter ac:name="language">bash</ac:parameter>
  <ac:plain-text-body><![CDATA[echo "code goes here"
special chars & < > are safe inside CDATA]]></ac:plain-text-body>
</ac:structured-macro>
```

### Links -- always explicit

Bare URLs are NOT auto-linked in Confluence storage format. Always wrap:

```html
<!-- WRONG -- renders as plain text -->
<p>Open http://10.40.8.80/platform/ to check.</p>

<!-- RIGHT -->
<p>Open <a href="http://10.40.8.80/platform/">http://10.40.8.80/platform/</a> to check.</p>
```

### Code blocks

Use `<ac:structured-macro ac:name="code">` with CDATA for the body.
CDATA is the RIGHT way in storage format (it protects `&`, `<`, `>` from XML parsing).
The old guidance "no CDATA" applied only to atl's markdown-to-storage converter path,
which cannot handle CDATA in `-f markdown.md` input -- it's correct in raw storage.

```html
<ac:structured-macro ac:name="code">
  <ac:parameter ac:name="language">bash</ac:parameter>
  <ac:parameter ac:name="title">Optional title</ac:parameter>
  <ac:plain-text-body><![CDATA[REDIS_PASS='0701!1523&SH'
# & < > all safe here]]></ac:plain-text-body>
</ac:structured-macro>
```

Supported `language` values: bash, python, go, java, sql, javascript, html, xml,
json, yaml, toml, text, none.

### Panels (info, warning, note, tip)

```html
<ac:structured-macro ac:name="warning">
  <ac:rich-text-body>
    <p>This page contains lab credentials. Do not forward.</p>
  </ac:rich-text-body>
</ac:structured-macro>

<ac:structured-macro ac:name="info">
  <ac:rich-text-body><p>Informational message.</p></ac:rich-text-body>
</ac:structured-macro>
```

Panel types: `info` (blue), `note` (yellow), `warning` (red), `tip` (green).

### Tables

```html
<table>
  <thead><tr><th>Key</th><th>Value</th><th>Notes</th></tr></thead>
  <tbody>
    <tr><td>VIP</td><td>10.40.8.80</td><td>keepalived VRRP</td></tr>
    <tr><td>Web Active</td><td>10.40.8.81</td><td>MASTER on healthy start</td></tr>
  </tbody>
</table>
```

### Horizontal rule

```html
<hr />
```

## Suggested Page Structure

```
{Warning/info panel if needed}

{Overview table or key-value summary}

---
{Section A -- instructions or content}

{Section B}
---

{Notes / caveats / links}
```

Keep it scannable. Engineers read Confluence pages like dashboards, not novels.
Lead with the actionable content; put context and caveats at the bottom.

## Known Bugs (atl markdown mode)

| Bug | Symptom | Workaround |
|-----|---------|------------|
| Unescaped `&` in code fences | HTTP 400, empty error body | Use storage format with CDATA |
| Unescaped `<` in code fences | Malformed XML, partial render | Same |
| No macro support in markdown | Panels/columns silently dropped | Use storage format |

These are converter bugs in atl, not Confluence issues. Filed upstream.

## Shell Gotchas

### Tilde expansion

```bash
# WRONG - tilde expands to home dir
atl page create -s ~john.doe -t "Title" -f page.html

# RIGHT - quote the space key
atl page create -s '~john.doe' -t "Title" -f page.html
```

### Positional arguments

```bash
# WRONG - page list takes positional space arg
atl page list --space '~john.doe'

# RIGHT
atl page list '~john.doe'
```

### Duplicate titles

Confluence rejects duplicate titles in same space (400 error).
Check first: `atl page search --title "Meeting Notes" -s SPACE`

## Limitations

- No `atl page move` -- use Confluence UI
- Personal spaces use `~username` format (must be quoted in shell)
- Markdown mode unreliable for pages with code blocks or macros -- use storage format
