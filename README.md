# atlas-cli

```
    _   _   _               ____ _     ___
   / \ | |_| | __ _ ___    / ___| |   |_ _|
  / _ \| __| |/ _` / __|  | |   | |    | |
 / ___ \ |_| | (_| \__ \  | |___| |___ | |
/_/   \_\__|_|\__,_|___/   \____|_____|___|

```

**Atlassian CLI for humans and AI agents.**

CLI tools work beautifully with AI coding assistants. `gh` for GitHub, now `atl` for Atlassian.

## Why This Exists

Watched Claude Code absolutely nail it with `gh` - reading PRs, checking issues, pushing code. Clean CLI output that AI can parse and act on.

Then tried using Atlassian services... MCP plugins, complex integrations, auth nightmares. Why can't we just have a simple CLI?

So we built one. Learned from `gh` and `jira-cli`, focused on what matters:
- **Confluence**: The killer feature. Full CRUD, markdown conversion, search.
- **Bitbucket**: PRs without the slow web UI.
- **JIRA**: View issues, filter with flags. Context, not workflow.

Works for humans. Works for AI agents. Same tool.

## What This Is NOT

- Not a jira-cli replacement - we do viewing, they do workflows
- Not trying to replace Atlassian's ecosystem - just making it CLI-accessible

## Install

One-liner (binary + Claude Code skill):

```bash
curl -fsSL https://raw.githubusercontent.com/lroolle/atlas-cli/main/install.sh | bash
```

<details>
<summary>Other installation methods</summary>

```bash
# System-wide (/usr/local/bin)
curl -fsSL https://raw.githubusercontent.com/lroolle/atlas-cli/main/install.sh | bash -s -- --system

# Binary only / Skill only
curl -fsSL https://raw.githubusercontent.com/lroolle/atlas-cli/main/install.sh | bash -s -- --bin-only
curl -fsSL https://raw.githubusercontent.com/lroolle/atlas-cli/main/install.sh | bash -s -- --skill-only

```

```bash
# Via Go
go install github.com/lroolle/atlas-cli/cmd/atl@latest

# From source
git clone https://github.com/lroolle/atlas-cli.git && cd atlas-cli && make build && make install
```

See `install.sh --help` for all flags (`--bin-dir`, `--skill-dir`, etc).
</details>

## Quick Start

```bash
atl init  # creates ~/.config/atlas/config.yaml
```

**Jira - Filter issues:**
```bash
atl issue list                         # All issues
atl issue list -a me -t Bug -s Open   # My open bugs
atl issue list -s '~Done' -e 123      # Not Done, epic auto-prefixed
atl issue list "search text"           # Text search
atl issue view PROJ-123                # Details + linked PRs
```

**Confluence - Docs as markdown:**
```bash
atl page view 12345 --format markdown      # Export to markdown
atl page search "API design" -s SPACE      # Find pages
atl page create -s SPACE -t "Doc" -f a.md  # Create from file
atl page delete 12345 --yes                # Delete (ID/title/URL)
```

**Bitbucket - PRs without the browser:**
```bash
atl pr list PROJ/repo --state OPEN
atl pr view PROJ/repo 140
atl pr diff PROJ/repo 85
```

**Full reference:** [skills/atl-cli/SKILL.md](skills/atl-cli/SKILL.md)

## Documentation

- [SKILL.md](skills/atl-cli/SKILL.md) - Complete command reference
- [Configuration](docs/CONFIGURATION.md) - Setup & tokens
- [Workflows](docs/WORKFLOWS.md) - Real examples
- [Troubleshooting](docs/TROUBLESHOOTING.md) - When things break

## Contributing

Want: Confluence features, Bitbucket improvements, bug fixes.
Don't want: Full JIRA client, enterprise bloat.

## Acknowledgments

- [gh](https://github.com/cli/cli) - The inspiration. CLI done right.
- [jira-cli](https://github.com/ankitpokhrel/jira-cli) - JIRA done right.
- [Claude Code](https://claude.ai/code) - The reason this exists.

---

Built because AI agents deserve good tools too.
