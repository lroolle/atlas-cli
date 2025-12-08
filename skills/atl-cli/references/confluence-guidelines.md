# Confluence Page Guidelines

## Before Any Edit

1. **Read first**: `atl page view <page-id> --format storage -o page.html`
2. **Preserve structure**: Keep original panels, macros, tables intact
3. **No silent deletions**: Don't remove content without explicit permission
4. **Check duplicates**: Search before creating new pages

## Suggested Page Structure

```
{Page TOC}

{Overview / Background / Context}

---
{Section A}

{Section B}

{Section C}
---

{Summary / Conclusion}

---

{Reference Links}
```

Adapt to your organization's conventions.

## Storage Format Gotchas

### Code Blocks
```html
<!-- WRONG - CDATA breaks in file upload -->
<ac:structured-macro ac:name="code">
  <ac:plain-text-body><![CDATA[code]]></ac:plain-text-body>
</ac:structured-macro>

<!-- RIGHT - simple pre tags -->
<pre>code here</pre>
```

### Shell Expansion
```bash
# WRONG - tilde expands to home dir
atl page create -s ~john.doe -t "Title" -c "<p>x</p>"

# RIGHT - quote the space key
atl page create -s '~john.doe' -t "Title" -c "<p>x</p>"
```

### Positional Arguments
```bash
# WRONG - page list takes positional space arg
atl page list --space '~john.doe'

# RIGHT
atl page list '~john.doe'

# Use search for filtering, not grep
atl page search "notes" -s '~john.doe'
```

### Duplicate Titles
Confluence rejects duplicate titles in same space (400 error).
Check first: `atl page search --title "Meeting Notes" -s SPACE`

## Limitations

- No `atl page move` command - use Confluence UI
- Personal spaces use `~username` format (must be quoted in shell)
