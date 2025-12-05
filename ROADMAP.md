# Roadmap

## Done (v0.1.0)

- Confluence: CRUD, search (CQL), markdown export, image download, TOC
- JIRA: View issues, transitions, comments, linked PRs
- Bitbucket: PR list/view/diff/merge/comment
- Cross-service: `atl issue prs`
- Config: YAML, env vars, bearer auth

---

## P0 - Blocking Adoption

- [ ] Shell completion (bash/zsh/fish)
- [ ] `--json` output flag
- [ ] `atl open` - browser shortcuts
- [ ] `atl pr create`
- [ ] Better error messages

---

## P1 - Quality of Life

- [ ] `atl page comment` add/list
- [ ] `atl init` wizard
- [ ] `$EDITOR` integration
- [ ] Color output
- [ ] Progress spinners
- [ ] `--web` flag on create/edit

---

## P2 - Feature Completeness

- [ ] Page labels
- [ ] Page watch/unwatch
- [ ] Attachment upload
- [ ] Page history/versions
- [ ] Page clone

---

## P3 - Production Grade

- [ ] Keyring token storage
- [ ] 60%+ test coverage
- [ ] API stability guarantee
- [ ] Homebrew formula
- [ ] Man pages

---

## Not Building

- Interactive TUI - overkill
- CSV output - use JSON + jq
- Deep JIRA workflows - use jira-cli
- Extension system - no demand
- Space admin - use web UI
