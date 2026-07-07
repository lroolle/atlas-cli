# Jira Workflow Transitions — discovery and defaults

How to move issues through any Jira Server workflow with
`atl issue transition` without hitting blind 400s. This is the generic
methodology; encode your instance's specifics (state graph, per-type
required fields, team constants) in your own config and team knowledge
base, not in this skill.

## Two kinds of "required"

1. **Screen-required fields** — visible via the API. List mode shows
   them per transition with allowed values:

   ```bash
   atl issue transition MYPROJ-123          # human-readable
   atl issue transition MYPROJ-123 --json   # full schema + allowed values
   ```

2. **Validator-required fields** — enforced by workflow validators,
   INVISIBLE to the REST API. They only surface when a transition POST
   fails; the server names them ("Time Spent is required",
   "customfield_10001: Root Cause is required"). No amount of metadata
   introspection reveals them up front.

So the fluent loop is: attempt -> read the named fields from the error ->
add flags -> retry. The tool optimizes that loop rather than pretending
it can pre-validate.

## Agent guardrails

Transitions mutate team-visible state. Working agents follow these:

1. List first (`atl issue transition <key>`) — see the real current
   status and transitions. Never assume state; never hardcode IDs.
2. `--dry-run` before the first real transition of a session or any
   unfamiliar transition/type combo — inspect the merged payload.
3. Only transition issues assigned to you or explicitly requested.
4. On 400, add exactly the fields the error names and retry ONCE; if it
   fails again, report instead of iterating blindly.
5. Team constants belong in `jira.transition_defaults` config. If the
   dry-run payload lacks expected defaults, the config block is missing
   — say so rather than hand-typing values.
6. Judgment fields (resolution, analysis text, checklist choices) must
   reflect the actual work, not copied examples.

## Discovering your instance's workflow

One-time, per project (agents: do this when the knowledge base has no
workflow map yet, and record the results there):

1. `atl issue transition <key>` on issues of each type in each status —
   maps the state graph and screen-required fields. Transition IDs
   usually differ by source status; match by name.
2. Attempt a transition on a throwaway/test issue with no fields — the
   400 lists the validator-required fields for that type. This mutates
   nothing when it fails.
3. Sample recently resolved issues of each type and compare which
   fields are consistently populated — that's the de-facto required
   set, including pure team conventions validators don't enforce.

## Transitioning

```bash
atl issue transition MYPROJ-123 "Start Progress"   # exact name
atl issue transition MYPROJ-123 resolve            # unique substring
atl issue transition MYPROJ-123 resolve -R Done --fix-version 1.2.0 -T 2h \
  -F "Root Cause=config error" -m "fixed in abc123"
```

- `-R` resolution, `--fix-version` fix versions, `-T` logs work,
  `-m` comments — all in the same transition POST.
- `-F "name=value"` sets any screen field by display name or ID; values
  are coerced from the field schema: options, multi-selects
  (ASCII-comma separated), versions, users, numbers, cascading selects
  (`"Parent / Child"`). Free-text array fields are never comma-split.
  Repeat `-F` for the same field to accumulate. Raw JSON passes through.

## Team constants belong in config

Values that are the same on every transition (team/zone fields, story
points conventions, estimates, current fix version) go in
`~/.config/atlas/config.yaml`, keyed by issue type then transition name:

```yaml
jira:
  transition_defaults:
    Bug:
      start progress:
        Team Zone: Backend / Platform
        Story Points: 0
        Original Estimate: 4h
      resolve issue:
        Fix Version/s: 1.2.0
        Time Spent: 2h
    '*':                      # applies to types without their own block
      resolve issue:
        Fix Version/s: 1.2.0
```

Rules:
- Defaults merge beneath CLI flags; an explicit flag replaces the
  default for that field entirely.
- Fields not on the target transition's screen are skipped silently, so
  a `resolve issue` block is safe even when screens differ per type.
- Special keys: `Time Spent` logs work, `Original Estimate` sets
  timetracking.
- `--no-defaults` ignores the block; `--dry-run` prints the exact POST
  body (defaults merged) without transitioning — use it to verify a new
  config block before first live use.

The command line then carries only what genuinely varies per issue
(resolution, analysis text, actual time spent).
