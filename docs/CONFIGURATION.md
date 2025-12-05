# Configuration

How to set up atlas-cli and get your API tokens.

## Quick Setup

```bash
atl init
# Edit ~/.config/atlas/config.yaml
```

That's it. Now get your tokens.

---

## Config File Location

Default: `~/.config/atlas/config.yaml`

Override with `--config`:
```bash
atl --config /path/to/config.yaml page list SPACE
```

---

## Config File Format

```yaml
username: your.username

# Confluence - the main feature
confluence:
  server: https://confluence.company.com
  token: your-bearer-token-here
  default_space: "~username"     # Personal space or team space

# Bitbucket Server (self-hosted) or Cloud
bitbucket:
  server: https://git.company.com
  token: your-api-token-here
  default_project: PROJ
  default_repo: repo-name

# JIRA - read-only viewing
jira:
  server: https://jira.company.com
  token: your-bearer-token-here
  default_project: PROJ
```

All sections are optional. Only configure what you actually use.

---

## Getting API Tokens

### Bitbucket Server (Self-Hosted)

1. Profile -> Manage account -> Personal access tokens
2. Click "Create a token"
3. Name: `atlas-cli`
4. Permissions:
   - **Read** repositories (required for PR list/view)
   - **Write** repositories (required for PR merge)
5. Copy token to config

**Token format:** Usually looks like `MTIzNDU2Nzg5MDEyOmFiY2RlZmdoaWprbG1ub3BxcnN0dXZ3eHl6`

### Bitbucket Cloud

1. Account settings -> App passwords
2. Create app password
3. Label: `atlas-cli`
4. Permissions:
   - Pull requests: Read, Write
   - Repositories: Read
5. Copy token to config

**Token format:** App password string

### Confluence Server/Cloud

**Option 1: Personal Access Token (recommended)**
1. Profile -> Personal Access Tokens -> Create token
2. Name: `atlas-cli`
3. Expiration: Set according to your security policy
4. Copy token to config

**Option 2: Bearer Token from Browser**
1. Open browser DevTools (F12)
2. Go to Network tab
3. Make a request to Confluence (click any page)
4. Find request to `/rest/api/...`
5. Copy `Authorization: Bearer ...` token
6. **Warning:** These expire quickly (hours to days)

**Token format:** `Bearer ODg5MDEyMzQ1Njc4OTAxMjpabWFzZGtqYXNka2phc2Rr...`

### Confluence Cloud

Use Atlassian API token:
1. Account Settings -> Security -> API tokens -> Create API token
2. Label: `atlas-cli`
3. Copy token to config

**Auth format:** Uses Basic Auth with email:token. Config stays the same, we handle it.

### JIRA Server/Cloud

Same as Confluence (they use the same auth system).

---

## Token Security

**DO:**
- Use Personal Access Tokens (they're revocable)
- Set token expiration based on your company policy
- Revoke tokens you're not using
- Keep `~/.config/atlas/` permissions to 700 (owner-only)

**DON'T:**
- Commit config.yaml to git (add to .gitignore)
- Share tokens with others (they're personal)
- Use someone else's token
- Use bearer tokens from browser for long-term (they expire)

**Check permissions:**
```bash
chmod 700 ~/.config/atlas
chmod 600 ~/.config/atlas/config.yaml
```

---

## Default Values

Set defaults in config to avoid typing them repeatedly:

```yaml
confluence:
  default_space: "~username"    # atl page list now uses this

bitbucket:
  default_project: PROJ
  default_repo: repo-name       # atl pr list now uses these

jira:
  default_project: PROJ
```

**With defaults:**
```bash
atl page list              # Uses default_space
atl pr list                # Uses default_project/default_repo
```

**Override defaults:**
```bash
atl page list OTHER_SPACE
atl pr list OTHER_PROJ/other-repo
```

---

## Multiple Configs

Use different configs for different environments:

```bash
atl --config ~/.config/atlas/work.yaml page list SPACE
atl --config ~/.config/atlas/personal.yaml page list SPACE
```

Or use shell aliases:
```bash
alias atl-work='atl --config ~/.config/atlas/work.yaml'
alias atl-personal='atl --config ~/.config/atlas/personal.yaml'
```

---

## Troubleshooting Auth

### 401 Unauthorized

**Confluence/JIRA:**
- Token expired (regenerate it)
- Wrong server URL
- Token doesn't have permissions

**Bitbucket:**
- Token revoked or expired
- Wrong server URL
- Token missing required permissions (Read repositories)

**Fix:**
1. Regenerate token in Atlassian UI
2. Update config.yaml
3. Test with simple command:
   ```bash
   atl page list MYSPACE
   atl pr list PROJ/repo
   ```

### 403 Forbidden

You don't have permissions for that operation.

**Confluence:**
- Can't create pages in that space
- Space is restricted

**Bitbucket:**
- Can't merge PRs (need write permission)
- Repository is restricted

**Fix:** Check your permissions in the web UI or request access.

### 404 Not Found

**Confluence:**
- Space doesn't exist (check space key)
- Page ID is wrong
- Typo in page title

**Bitbucket:**
- Project or repo doesn't exist
- Typo in PROJ/repo format (case-sensitive)

**JIRA:**
- Issue doesn't exist
- Wrong project key

**Fix:** Double-check IDs/keys. They're case-sensitive.

### SSL Certificate Errors

Self-signed certs or corporate proxies can cause issues.

**Bad fix (insecure):**
```bash
export ATLAS_SKIP_VERIFY=true
```

**Good fix:**
- Add corporate CA cert to system trust store
- Or use proper SSL cert on Atlassian servers

---

## Server URL Format

**Correct:**
```yaml
server: https://confluence.company.com
server: https://git.company.com
server: https://jira.company.com
```

**Wrong:**
```yaml
server: https://confluence.company.com/    # Trailing slash causes issues
server: http://confluence.company.com      # Use HTTPS (we auto-upgrade but don't rely on it)
```

Most APIs fail with trailing slashes. Don't use them.

---

## Confluence Cloud vs Server

### Server (Self-Hosted)

```yaml
confluence:
  server: https://confluence.company.com
  token: your-bearer-token
```

### Cloud (Atlassian-Hosted)

```yaml
confluence:
  server: https://yourcompany.atlassian.net
  token: your-api-token
```

We detect Cloud vs Server and handle auth differences automatically.

**Differences:**
- Cloud uses email:token Basic Auth
- Server uses Bearer token
- Some API endpoints differ (file a bug if something breaks)

---

## Environment Variables

**Not currently supported.** Config file only.

If you need this, file an issue. We can add `ATLAS_CONFLUENCE_TOKEN` etc.

---

## Example Configs

### Minimal (Confluence only)

```yaml
username: john.doe

confluence:
  server: https://confluence.company.com
  token: your-token-here
  default_space: "TEAM"
```

### Full Stack

```yaml
username: jane.smith

confluence:
  server: https://confluence.company.com
  token: confluence-token-here
  default_space: "~jane.smith"

bitbucket:
  server: https://git.company.com
  token: bitbucket-token-here
  default_project: WEBAPP
  default_repo: frontend

jira:
  server: https://jira.company.com
  token: jira-token-here
  default_project: WEBAPP
```

### Multiple Environments

**~/.config/atlas/work.yaml:**
```yaml
username: you@company.com
confluence:
  server: https://confluence.company.com
  token: work-token
```

**~/.config/atlas/personal.yaml:**
```yaml
username: yourname
confluence:
  server: https://yourname.atlassian.net
  token: personal-token
```

Use with:
```bash
atl --config ~/.config/atlas/work.yaml page list SPACE
atl --config ~/.config/atlas/personal.yaml page list SPACE
```

---

## Config Validation

No built-in validation yet. If your config is wrong, you'll get API errors.

**Common mistakes:**
- Trailing slash in server URL
- Forgot to quote `~username` space keys (shell expansion)
- Token has wrong permissions
- Wrong token format for Cloud vs Server

**Test your config:**
```bash
atl page spaces       # List spaces (tests Confluence auth)
atl pr list          # List PRs (tests Bitbucket auth)
atl issue view PROJ-1 # View issue (tests JIRA auth)
```

---

## See Also

- [Usage](USAGE.md) - Full command reference
- [Troubleshooting](TROUBLESHOOTING.md) - Common errors and fixes
