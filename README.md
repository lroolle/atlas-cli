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
- **JIRA**: View issues, check linked PRs. Context, not workflow.

Works for humans. Works for AI agents. Same tool.

## Install

One-liner (installs binary + Claude Code skill):

```bash
curl -fsSL https://raw.githubusercontent.com/lroolle/atlas-cli/main/install.sh | bash
```

System-wide (`/usr/local/bin`, requires sudo):

```bash
curl -fsSL https://raw.githubusercontent.com/lroolle/atlas-cli/main/install.sh | bash -s -- --system
```

Custom binary location:

```bash
curl -fsSL https://raw.githubusercontent.com/lroolle/atlas-cli/main/install.sh | bash -s -- --bin-dir /usr/local/bin
```

Custom skill location:

```bash
curl -fsSL https://raw.githubusercontent.com/lroolle/atlas-cli/main/install.sh | bash -s -- --skill-dir ~/my-skills
```

Binary only / Skill only:

```bash
curl -fsSL https://raw.githubusercontent.com/lroolle/atlas-cli/main/install.sh | bash -s -- --bin-only
curl -fsSL https://raw.githubusercontent.com/lroolle/atlas-cli/main/install.sh | bash -s -- --skill-only
```

Via Go:

```bash
go install github.com/lroolle/atlas-cli/cmd/atl@latest
```

From source:

```bash
git clone https://github.com/lroolle/atlas-cli.git && cd atlas-cli
make build && make install
```

## Quick Start

```bash
atl init  # creates ~/.config/atlas/config.yaml
```

```bash
atl page view 12345 --format markdown     # AI can read this
atl page create -s SPACE -t "Doc" -f a.md # AI can write docs
atl page search "API design" -s SPACE     # AI can find stuff
atl issue view PROJ-123                   # AI gets context
atl pr list PROJ/repo                     # AI sees PR status
```

## Documentation

- [Usage](docs/USAGE.md) - All commands
- [Configuration](docs/CONFIGURATION.md) - Setup & tokens
- [Workflows](docs/WORKFLOWS.md) - Real examples
- [Troubleshooting](docs/TROUBLESHOOTING.md) - When things break

## What This Is NOT

- Not a jira-cli replacement - we do viewing, they do workflows
- Not trying to replace Atlassian's ecosystem - just making it CLI-accessible

## Contributing

Want: Confluence features, Bitbucket improvements, bug fixes.
Don't want: Full JIRA client, enterprise bloat.

## License

MIT

## Acknowledgments

- [gh](https://github.com/cli/cli) - The inspiration. CLI done right.
- [jira-cli](https://github.com/ankitpokhrel/jira-cli) - JIRA done right.
- [Claude Code](https://claude.ai/code) - The reason this exists.

---

Built because AI agents deserve good tools too.
