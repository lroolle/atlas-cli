# Atlas CLI 🏔️

*Making Atlassian's enterprise software slightly less painful to use.*

GitHub CLI-inspired tool for Bitbucket, JIRA, and Confluence. Born from pure frustration: we needed Claude Code to work with Atlassian's bloated enterprise crap. Because if AI has to suffer through their APIs, at least we can make it less idiotic.

## Quick Start

```sh
make build
./atl init
# Edit ~/.config/atlas/config.yaml with your tokens
```

## Usage

```sh
# Pull Requests (no more pagination hell)
atl pr list                  # List PRs
atl pr view 123              # View PR  
atl pr merge 123             # Merge PR

# JIRA Issues (surprisingly bearable)
atl issue list               # List issues
atl issue view PROJ-123      # View issue

# Confluence Pages (actually fast)
atl page list                # List pages
atl page search "notes"      # Search that works
```

## Config

```yaml
# ~/.config/atlas/config.yaml
username: your.username

bitbucket:
  server: https://git.yourdomain.com
  token: your-api-token
  default_project: PROJ
  default_repo: repo

jira:
  server: https://jira.yourdomain.com  
  token: your-bearer-token
  default_project: PROJ

confluence:
  server: https://confluence.yourdomain.com
  token: your-bearer-token
  default_space: "~username"
```

## License

MIT