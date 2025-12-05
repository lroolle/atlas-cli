# Troubleshooting

When things break. Because they will.

---

## Authentication Errors

### 401 Unauthorized

**Symptoms:**
```
Error: 401 Unauthorized
```

**Causes:**
1. Token expired
2. Wrong token in config
3. Wrong server URL
4. Token doesn't have required permissions

**Fix:**

**For Confluence/JIRA:**
```bash
# 1. Regenerate token
# Go to Profile -> Personal Access Tokens -> Create token

# 2. Update config
vim ~/.config/atlas/config.yaml

# 3. Test
atl page spaces
```

**For Bitbucket:**
```bash
# 1. Check token hasn't been revoked
# Profile -> Manage account -> Personal access tokens

# 2. Verify permissions: Read repositories (minimum)

# 3. Test
atl pr list PROJ/repo
```

### 403 Forbidden

**Symptoms:**
```
Error: 403 Forbidden
```

**You don't have permissions for that operation.**

**Confluence:**
- Can't create pages in that space (space is restricted)
- Can't edit page (not owner or space admin)
- Can't delete page (insufficient permissions)

**Bitbucket:**
- Can't merge PR (need write permission on repo)
- Can't view private repo

**JIRA:**
- Can't view issue (issue is restricted)
- Can't transition issue (workflow doesn't allow it)

**Fix:**
- Request permissions in web UI
- Or give up and do it in the web UI

### 404 Not Found

**Symptoms:**
```
Error: 404 Not Found
```

**Causes:**
1. Wrong page ID
2. Wrong space key (case-sensitive)
3. Page was deleted
4. Typo in project/repo name

**Fix:**
```bash
# Confluence: Verify space exists
atl page spaces | grep MYSPACE

# Confluence: Search by title instead
atl page search --title "Page Title" -s MYSPACE

# Bitbucket: Double-check PROJ/repo (case matters)
atl pr list PROJ/repo  # not proj/repo

# JIRA: Verify issue exists
atl issue view PROJ-123
```

---

## API Errors

### 400 Bad Request

**Symptoms:**
```
Error: 400 Bad Request
```

**Common causes:**

**1. Duplicate page title**
```
Error: A page with this title already exists in the space
```

**Fix:** Change the title or delete the existing page
```bash
atl page search --title "Duplicate Title" -s MYSPACE
# If exists, delete or rename
atl page delete 12345678 --yes
```

**2. Invalid HTML/XML content**
```
Error: Invalid content
```

**Fix:** Check your HTML for unclosed tags, invalid characters
```bash
# Validate HTML before creating
# Use simple tags: <h1>, <p>, <ul>, <li>, <strong>, <em>
# Avoid CDATA, complex macros
```

**3. Parent page doesn't exist**
```
Error: Parent page not found
```

**Fix:** Verify parent page ID/title
```bash
atl page search --title "Parent Title" -s MYSPACE
```

### 500 Internal Server Error

**Symptoms:**
```
Error: 500 Internal Server Error
```

**Atlassian's API just broke. Not your fault.**

**Possible causes:**
- Atlassian's servers are having a bad day
- Request is too large (giant page content)
- API timeout (slow network)

**Fix:**
1. Retry in a minute
2. Check Atlassian status page
3. Reduce request size (split large content)
4. If persistent, file a bug with API response

---

## Content/Formatting Issues

### Tilde Expansion in Space Keys

**Symptoms:**
```bash
atl page list ~username
# Error: Space not found or expanded to home directory
```

**Cause:** Shell expands `~username` to `/home/username`

**Fix:** Quote the space key
```bash
atl page list '~username'
```

### CDATA Gets Mangled

**Symptoms:**
Code blocks with `<![CDATA[...]]>` show up wrong or fail.

**Cause:** Shell interprets `!` and `[]`, Confluence doesn't preserve CDATA.

**Fix:** Use `<pre>` tags instead
```html
<!-- Don't do this -->
<![CDATA[
code here
]]>

<!-- Do this -->
<pre>
code here
</pre>
```

### Images Don't Export

**Symptoms:**
Exported markdown has broken image links.

**Cause:** Didn't use `--with-images` flag.

**Fix:**
```bash
atl page view 12345678 -o backup.md --format markdown --with-images
```

**Warning:** Slow for pages with lots of images.

### Parent Page Resolution Fails

**Symptoms:**
```
Error: Could not resolve parent page
```

**Causes:**
1. Parent title is ambiguous (multiple pages with same title)
2. Parent doesn't exist in that space
3. Typo in parent title

**Fix:**
```bash
# Use page ID instead of title (unambiguous)
atl page create -s MYSPACE -t "Child" -f child.html -p 12345678

# Or verify parent exists
atl page search --title "Parent Title" -s MYSPACE
```

---

## Performance Issues

### Slow API Responses

**Symptoms:**
Commands take forever.

**Cause:** Atlassian's APIs are just slow, especially:
- Confluence search
- Large page exports with images
- Bitbucket diff for large PRs

**Fix:**
```bash
# Use --limit to reduce results
atl page search "query" -s MYSPACE --limit 10

# Export without images first
atl page view 12345678 -o page.md --format markdown  # Fast

# Then with images if needed
atl page view 12345678 -o page.md --format markdown --with-images  # Slow
```

**No magic fix.** Atlassian's servers are what they are.

### Rate Limiting

**Symptoms:**
```
Error: 429 Too Many Requests
```

**Cause:** You hit the API rate limit (usually 100-1000 req/hour depending on your instance).

**Fix:**
Add delays between requests:
```bash
cat ids.txt | while read id; do
  atl page view $id -o "export/$id.md" --format markdown
  sleep 1  # Don't hammer the API
done
```

**We don't implement retry logic yet.** If you hit this regularly, file an issue.

---

## Config Issues

### Config File Not Found

**Symptoms:**
```
Error: Config file not found
```

**Fix:**
```bash
# Generate default config
atl init

# Or specify path
atl --config /path/to/config.yaml page list SPACE
```

### Invalid YAML

**Symptoms:**
```
Error: Failed to parse config
```

**Cause:** Syntax error in YAML file.

**Fix:**
```bash
# Check YAML syntax
yamllint ~/.config/atlas/config.yaml

# Or validate with Python
python3 -c 'import yaml; yaml.safe_load(open("/home/user/.config/atlas/config.yaml"))'
```

**Common YAML mistakes:**
- Inconsistent indentation (use spaces, not tabs)
- Forgot quotes around values with special chars: `~username` should be `"~username"`
- Trailing whitespace

### Server URL Issues

**Symptoms:**
Weird 404 or 400 errors on all requests.

**Cause:** Server URL has trailing slash or wrong protocol.

**Fix:**
```yaml
# Wrong
server: https://confluence.company.com/
server: http://confluence.company.com

# Right
server: https://confluence.company.com
```

---

## Platform-Specific Issues

### macOS: Permission Denied

**Symptoms:**
```
-bash: /Users/you/go/bin/atl: Permission denied
```

**Fix:**
```bash
chmod +x ~/go/bin/atl
```

### Linux: Command Not Found

**Symptoms:**
```
bash: atl: command not found
```

**Cause:** `~/go/bin` not in PATH.

**Fix:**
```bash
# Add to ~/.bashrc or ~/.zshrc
export PATH=$PATH:~/go/bin

# Reload shell
source ~/.bashrc
```

### Windows: Not Tested

We don't test on Windows. It might work. It might not.

If you use Windows, install WSL and run atlas-cli there.

---

## Known Issues

### 1. No Page Move Between Spaces

**Can't move pages between spaces via API.**

**Workaround:**
```bash
# Export from old space
atl page view 12345678 -o /tmp/page.html --format storage

# Get title
title=$(atl page view 12345678 --info --format json | jq -r '.title')

# Create in new space
atl page create -s NEWSPACE -t "$title" -f /tmp/page.html

# Delete from old space
atl page delete 12345678 --yes
```

Images don't transfer. You need to download/upload them manually.

### 2. Confluence Cloud vs Server Differences

Some API endpoints behave differently.

**Known differences:**
- Cloud uses Basic Auth (email:token), Server uses Bearer token
- Delete on Server is permanent, Cloud has trash
- Some search filters differ

**If something breaks on Cloud, file an issue.**

### 3. No Retry Logic

We don't retry failed API requests.

If you hit rate limits or transient errors, you need to retry manually.

**Future:** Add `--retry` flag.

### 4. Large Attachments

Exporting pages with large images can time out or consume lots of memory.

**Workaround:** Export without images first, download images separately if needed.

### 5. Delete Is Permanent (Server)

Confluence Server deletes are final. Cloud has a trash, but Server doesn't.

**We warn you. Don't ignore the warning.**

---

## SSL Certificate Errors

**Symptoms:**
```
Error: x509: certificate signed by unknown authority
```

**Cause:** Self-signed cert or corporate proxy.

**Bad fix (insecure):**
```bash
export ATLAS_SKIP_VERIFY=true  # DON'T DO THIS IN PRODUCTION
```

**Good fix:**
1. Add corporate CA cert to system trust store
2. Or get proper SSL cert on Atlassian servers

---

## Debug Mode

**Not implemented yet.** If you need debug output, file an issue.

**Workaround:** Use `--format json` to see raw API responses
```bash
atl page view 12345678 --format json | jq .
```

---

## Common Error Messages

### "Space not found"

**Fix:**
```bash
# List spaces to verify
atl page spaces | grep MYSPACE

# Check case (space keys are case-sensitive)
atl page list MYSPACE  # not myspace
```

### "Page title already exists"

**Fix:**
```bash
# Find conflicting page
atl page search --title "Duplicate" -s MYSPACE

# Delete or rename it
atl page delete 12345678 --yes
```

### "Invalid parent page"

**Fix:**
```bash
# Use page ID instead of title
atl page create -s SPACE -t "Child" -f file.html -p 12345678

# Or verify parent exists
atl page search --title "Parent" -s SPACE
```

### "Could not parse Confluence URL"

**Fix:**
Check URL format:
```
Correct: https://confluence.company.com/display/SPACE/Page+Title
Wrong:   confluence.company.com/SPACE/Page
```

---

## Getting Help

**Search docs first:**
- [Usage](USAGE.md) - Full command reference
- [Configuration](CONFIGURATION.md) - Auth and config
- [Workflows](WORKFLOWS.md) - Real-world examples

**Still broken?**

File an issue: https://github.com/lroolle/atlas-cli/issues

**Include:**
1. atlas-cli version (`atl --version`)
2. Command you ran (remove sensitive tokens)
3. Full error message
4. Confluence/Bitbucket/JIRA version (Server or Cloud)

**Don't include:**
- API tokens
- Page content with sensitive info
- Your company's server URLs (replace with example.com)

---

## Things That Are Not Bugs

### "CLI is too slow"

Atlassian's APIs are slow. We can't fix that.

### "Can't do X in JIRA"

Use [jira-cli](https://github.com/ankitpokhrel/jira-cli). We do read-only JIRA.

### "Feature X from web UI is missing"

We implement the useful 20%, not the entire 100% of features.

File an issue if the missing feature is actually important.

---

## Emergency: I Deleted the Wrong Page

### Confluence Cloud

**There's a trash.**

Go to: Space Settings -> Content Tools -> Trash -> Restore

### Confluence Server

**No trash. It's gone.**

Options:
1. Restore from backup (if you have one)
2. Check Confluence audit log, maybe an admin can help
3. Recreate from memory (sorry)

**This is why we warn you before deleting.**
