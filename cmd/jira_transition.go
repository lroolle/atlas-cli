package cmd

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"strconv"
	"strings"

	"github.com/lroolle/atlas-cli/api"
	"github.com/lroolle/atlas-cli/internal/cmdutil"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

var issueTransitionCmd = &cobra.Command{
	Use:   "transition [issue-key] [transition]",
	Short: "Transition a JIRA issue to a new status",
	Long: `Transition a JIRA issue through its workflow.

Without a transition argument, lists available transitions including
required fields and their allowed values.

The transition argument matches (case-insensitive) the transition name,
the target status name, the transition ID, or a unique substring:
  atl issue transition MYPROJ-123 "Start Progress"
  atl issue transition MYPROJ-123 resolve

Fields on the transition screen can be set by display name or field ID;
values are coerced to the right JSON shape from the field schema:
  -F "Root Cause=config" -F "Test Level=unit,integration"
  -F "Team Zone=Backend / Platform"            # cascading select
  -F 'customfield_10001={"value":"config"}'    # raw JSON still works

Team constants belong in config, not on the command line. Declare them
under jira.transition_defaults keyed by issue type then transition name;
they merge beneath CLI flags and skip fields not on the screen:

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

Use --dry-run to inspect the exact payload without transitioning, and
--no-defaults to ignore the config block.`,
	Example: `  atl issue transition MYPROJ-123
  atl issue transition MYPROJ-123 --json
  atl issue transition MYPROJ-123 "Start Progress"
  atl issue transition MYPROJ-123 resolve -R Done --fix-version 1.2.0 -T 2h -m "fixed in abc123"
  atl issue transition MYPROJ-123 close -R "Won't Fix" -F "Closure Reason=stale"

Note: workflow validators may require fields beyond the ones marked
required in list mode (e.g. Time Spent, Fix Version/s, checklist
fields) — the server names them in the error; set them with -F/-T
and retry.`,
	Aliases: []string{"move", "mv"},
	Args:    cobra.RangeArgs(1, 2),
	RunE:    runIssueTransition,
}

func runIssueTransition(cmd *cobra.Command, args []string) error {
	ctx := cmd.Context()

	client, err := api.GetJiraClient()
	cmdutil.ExitIfError(err)

	issueKey := args[0]

	transitions, err := client.GetTransitions(ctx, issueKey)
	if err != nil {
		return err
	}

	if len(args) == 1 {
		if asJSON, _ := cmd.Flags().GetBool("json"); asJSON {
			return json.NewEncoder(os.Stdout).Encode(transitions)
		}
		printTransitions(ctx, client, issueKey, transitions)
		return nil
	}

	target, err := matchTransition(transitions, args[1])
	if err != nil {
		return err
	}

	opts := transitionOptions{}
	opts.Resolution, _ = cmd.Flags().GetString("resolution")
	opts.FixVersions, _ = cmd.Flags().GetStringSlice("fix-version")
	opts.RawFields, _ = cmd.Flags().GetStringArray("field")

	update := api.TransitionUpdate{}
	update.Comment, _ = cmd.Flags().GetString("comment")
	update.TimeSpent, _ = cmd.Flags().GetString("time-spent")

	if noDefaults, _ := cmd.Flags().GetBool("no-defaults"); !noDefaults {
		issue, err := client.GetIssue(ctx, issueKey)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Warning: skipping transition defaults, cannot read issue: %v\n", err)
		} else {
			opts.Defaults = transitionDefaults(issue.Fields.IssueType.Name, target.Name)
		}
	}
	if ts, ok := popDefault(opts.Defaults, "Time Spent"); ok && update.TimeSpent == "" {
		update.TimeSpent = ts
	}
	if est, ok := popDefault(opts.Defaults, "Original Estimate"); ok {
		opts.Defaults["timetracking"] = est
	}

	fields, err := buildTransitionFields(opts, target)
	if err != nil {
		return err
	}

	if dryRun, _ := cmd.Flags().GetBool("dry-run"); dryRun {
		out := map[string]interface{}{
			"issue": issueKey,
			"to":    target.To.Name,
			"body":  api.TransitionBody(target.ID, fields, update),
		}
		enc := json.NewEncoder(os.Stdout)
		enc.SetIndent("", "  ")
		return enc.Encode(out)
	}

	err = client.TransitionIssue(ctx, issueKey, target.ID, fields, update)
	if err != nil {
		printRequiredFieldsHint(target)
		return err
	}

	fmt.Printf("Issue %s transitioned to %s\n", issueKey, target.To.Name)
	return nil
}

// transitionDefaults reads jira.transition_defaults.<issue type>.<transition>
// from config: a map of field display name (or id) -> value applied before
// CLI flags. Viper lowercases keys; matching is case-insensitive throughout.
func transitionDefaults(issueType, transitionName string) map[string]string {
	root := viper.GetStringMap("jira.transition_defaults")
	byType, ok := root[strings.ToLower(issueType)].(map[string]interface{})
	if !ok {
		if byType, ok = root["*"].(map[string]interface{}); !ok {
			return nil
		}
	}
	byTransition, ok := byType[strings.ToLower(transitionName)].(map[string]interface{})
	if !ok {
		return nil
	}
	defaults := make(map[string]string, len(byTransition))
	for k, v := range byTransition {
		defaults[k] = fmt.Sprintf("%v", v)
	}
	return defaults
}

func popDefault(defaults map[string]string, name string) (string, bool) {
	for k, v := range defaults {
		if strings.EqualFold(k, name) {
			delete(defaults, k)
			return v, true
		}
	}
	return "", false
}

func printTransitions(ctx context.Context, client *api.JiraClient, issueKey string, transitions []api.Transition) {
	header := fmt.Sprintf("Transitions for %s", issueKey)
	if issue, err := client.GetIssue(ctx, issueKey); err == nil {
		header = fmt.Sprintf("Transitions for %s [%s / %s]", issueKey, issue.Fields.IssueType.Name, issue.Fields.Status.Name)
	}
	fmt.Println(header)

	for _, t := range transitions {
		fmt.Printf("  %-4s %-28s -> %s\n", t.ID, t.Name, t.To.Name)
		for id, meta := range t.Fields {
			if !meta.Required {
				continue
			}
			line := fmt.Sprintf("       requires %s (%s)", meta.Name, id)
			if vals := formatAllowedValues(meta.AllowedValues); vals != "" {
				line += ": " + vals
			}
			fmt.Println(line)
		}
	}
}

type transitionOptions struct {
	Resolution  string
	FixVersions []string
	RawFields   []string
	// Defaults come from jira.transition_defaults config, lowest
	// precedence, silently skipped when not on the transition screen.
	Defaults map[string]string
}

// matchTransition finds a transition by exact name, target status name, or
// ID (case-insensitive), falling back to a unique substring match.
func matchTransition(transitions []api.Transition, arg string) (*api.Transition, error) {
	needle := strings.ToLower(arg)

	for i := range transitions {
		t := &transitions[i]
		if strings.ToLower(t.Name) == needle || strings.ToLower(t.To.Name) == needle || t.ID == arg {
			return t, nil
		}
	}

	var candidates []*api.Transition
	for i := range transitions {
		t := &transitions[i]
		if strings.Contains(strings.ToLower(t.Name), needle) || strings.Contains(strings.ToLower(t.To.Name), needle) {
			candidates = append(candidates, t)
		}
	}
	if len(candidates) == 1 {
		return candidates[0], nil
	}
	if len(candidates) > 1 {
		names := make([]string, len(candidates))
		for i, t := range candidates {
			names[i] = fmt.Sprintf("%q", t.Name)
		}
		return nil, fmt.Errorf("transition %q is ambiguous: matches %s", arg, strings.Join(names, ", "))
	}

	available := make([]string, len(transitions))
	for i, t := range transitions {
		available[i] = fmt.Sprintf("%q -> %s", t.Name, t.To.Name)
	}
	return nil, fmt.Errorf("transition %q not found, available: %s", arg, strings.Join(available, ", "))
}

// buildTransitionFields assembles the fields payload. Keys in --field
// entries may be field IDs or display names from the transition screen;
// plain values are coerced to the JSON shape the field schema expects.
func buildTransitionFields(opts transitionOptions, t *api.Transition) (map[string]interface{}, error) {
	fields := make(map[string]interface{})

	fromDefaults := make(map[string]bool)
	for key, val := range opts.Defaults {
		fieldID, meta, err := resolveTransitionField(t, key)
		if err != nil || meta == nil {
			continue // default's field is not on this transition's screen
		}
		fields[fieldID] = coerceTransitionValue(*meta, val)
		fromDefaults[fieldID] = true
	}

	if opts.Resolution != "" {
		fields["resolution"] = map[string]string{"name": opts.Resolution}
	}

	if len(opts.FixVersions) > 0 {
		versions := make([]map[string]string, len(opts.FixVersions))
		for i, v := range opts.FixVersions {
			versions[i] = map[string]string{"name": v}
		}
		fields["fixVersions"] = versions
	}

	for _, f := range opts.RawFields {
		parts := strings.SplitN(f, "=", 2)
		if len(parts) != 2 {
			return nil, fmt.Errorf("invalid --field format %q, expected key=value", f)
		}
		key, val := parts[0], parts[1]

		fieldID, meta, err := resolveTransitionField(t, key)
		if err != nil {
			return nil, err
		}

		var value interface{}
		if strings.HasPrefix(val, "[") || strings.HasPrefix(val, "{") {
			var parsed interface{}
			if err := json.Unmarshal([]byte(val), &parsed); err == nil {
				value = parsed
			} else {
				value = val
			}
		} else if meta != nil {
			value = coerceTransitionValue(*meta, val)
		} else {
			value = val
		}

		// Repeated flags for the same array field accumulate, but the
		// first explicit flag replaces a config default entirely.
		if prev, ok := fields[fieldID].([]interface{}); ok && !fromDefaults[fieldID] {
			if next, ok := value.([]interface{}); ok {
				fields[fieldID] = append(prev, next...)
				continue
			}
		}
		delete(fromDefaults, fieldID)
		fields[fieldID] = value
	}

	return fields, nil
}

// resolveTransitionField maps a --field key to a field ID using the
// transition screen metadata. Display names match case-insensitively.
func resolveTransitionField(t *api.Transition, key string) (string, *api.TransitionField, error) {
	if t != nil {
		if meta, ok := t.Fields[key]; ok {
			return key, &meta, nil
		}
		for id, meta := range t.Fields {
			if strings.EqualFold(meta.Name, key) {
				m := meta
				return id, &m, nil
			}
		}
	}

	// Field IDs pass through untouched; an unresolved display name is an
	// error because Jira would reject it with an opaque message anyway.
	if strings.Contains(key, " ") {
		var names []string
		if t != nil {
			for _, meta := range t.Fields {
				names = append(names, meta.Name)
			}
		}
		return "", nil, fmt.Errorf("field %q is not on the %q transition screen (available: %s)",
			key, t.Name, strings.Join(names, ", "))
	}
	return key, nil, nil
}

// coerceTransitionValue converts a plain string value into the JSON shape
// the field schema expects (option -> {"value": v}, version -> {"name": v},
// arrays of those -> lists, comma-separated for multi-select fields).
func coerceTransitionValue(meta api.TransitionField, raw string) interface{} {
	switch meta.Schema.Type {
	case "option":
		return map[string]string{"value": raw}
	case "option-with-child":
		// Cascading select: "Parent / Child" (e.g. "Backend / Platform")
		parent, child, hasChild := strings.Cut(raw, "/")
		v := map[string]interface{}{"value": strings.TrimSpace(parent)}
		if hasChild {
			v["child"] = map[string]string{"value": strings.TrimSpace(child)}
		}
		return v
	case "timetracking":
		return map[string]string{"originalEstimate": raw}
	case "resolution", "priority", "user", "version", "issuetype":
		return map[string]string{"name": raw}
	case "number":
		if n, err := strconv.ParseFloat(raw, 64); err == nil {
			return n
		}
		return raw
	case "array":
		// Only split on commas for fields with enumerated values;
		// free-text arrays (e.g. labels-like) keep the raw string whole.
		parts := []string{raw}
		if len(meta.AllowedValues) > 0 || meta.Schema.Items == "version" {
			parts = strings.Split(raw, ",")
		}
		out := make([]interface{}, 0, len(parts))
		for _, p := range parts {
			p = strings.TrimSpace(p)
			switch meta.Schema.Items {
			case "option":
				out = append(out, map[string]string{"value": p})
			case "version", "user", "group", "component":
				out = append(out, map[string]string{"name": p})
			default:
				out = append(out, p)
			}
		}
		return out
	default:
		return raw
	}
}

const maxAllowedValuesShown = 12

func formatAllowedValues(vals []api.AllowedValue) string {
	if len(vals) == 0 {
		return ""
	}
	shown := vals
	extra := ""
	if len(vals) > maxAllowedValuesShown {
		shown = vals[:maxAllowedValuesShown]
		extra = fmt.Sprintf(", … (+%d more)", len(vals)-maxAllowedValuesShown)
	}
	names := make([]string, len(shown))
	for i, v := range shown {
		names[i] = v.Display()
	}
	return strings.Join(names, ", ") + extra
}

func printRequiredFieldsHint(t *api.Transition) {
	var lines []string
	for id, meta := range t.Fields {
		if !meta.Required {
			continue
		}
		line := fmt.Sprintf("  %s (%s)", meta.Name, id)
		if vals := formatAllowedValues(meta.AllowedValues); vals != "" {
			line += ": " + vals
		}
		lines = append(lines, line)
	}
	if len(lines) == 0 {
		return
	}
	fmt.Fprintf(os.Stderr, "Transition %q requires:\n%s\n", t.Name, strings.Join(lines, "\n"))
}
