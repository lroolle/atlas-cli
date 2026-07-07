package cmd

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/lroolle/atlas-cli/api"
	"github.com/lroolle/atlas-cli/internal/cmdutil"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

const (
	epicLinkSchema = "com.pyxis.greenhopper.jira:gh-epic-link"
	sprintSchema   = "com.pyxis.greenhopper.jira:gh-sprint"
)

type issueCreateOptions struct {
	Project     string
	Type        string
	Summary     string
	Description string
	Parent      string
	Priority    string
	Assignee    string
	Epic        string
	Sprint      int
	StoryPoints float64
	Labels      []string
	FixVersions []string
	Components  []string
	RawFields   []string
}

// agileFieldIDs holds the server-specific custom field IDs for agile fields,
// resolved at runtime from /rest/api/2/field.
type agileFieldIDs struct {
	EpicLink    string
	Sprint      string
	StoryPoints string
}

var issueCreateCmd = &cobra.Command{
	Use:   "create",
	Short: "Create a JIRA issue",
	Long: `Create a JIRA issue (Story, Task, Bug, Sub-dev-task, ...).

Sub-task types require --parent. Sub-tasks inherit sprint from their
parent, so --sprint is rejected by the server for sub-task types.

Epic, sprint, and story points are stored in server-specific custom
fields; they are resolved automatically from the JIRA field registry.`,
	Example: `  atl issue create -t Story -s "story title" -e MYPROJ-100 --sprint 1946 --story-points 3
  atl issue create -t Sub-task -P MYPROJ-123 -s "dev subtask title"
  atl issue create -t Bug -s "crash on empty input" -y Critical -a me
  atl issue create -t Task -s "raw field example" --field 'customfield_10103=5'`,
	Aliases: []string{"new"},
	Args:    cobra.NoArgs,
	RunE:    runIssueCreate,
}

func runIssueCreate(cmd *cobra.Command, args []string) error {
	ctx := cmd.Context()

	opts := issueCreateOptions{}
	opts.Project, _ = cmd.Flags().GetString("project")
	opts.Type, _ = cmd.Flags().GetString("type")
	opts.Summary, _ = cmd.Flags().GetString("summary")
	opts.Description, _ = cmd.Flags().GetString("description")
	opts.Parent, _ = cmd.Flags().GetString("parent")
	opts.Priority, _ = cmd.Flags().GetString("priority")
	opts.Assignee, _ = cmd.Flags().GetString("assignee")
	opts.Epic, _ = cmd.Flags().GetString("epic")
	opts.Sprint, _ = cmd.Flags().GetInt("sprint")
	opts.StoryPoints, _ = cmd.Flags().GetFloat64("story-points")
	opts.Labels, _ = cmd.Flags().GetStringArray("label")
	opts.FixVersions, _ = cmd.Flags().GetStringSlice("fix-version")
	opts.Components, _ = cmd.Flags().GetStringSlice("component")
	opts.RawFields, _ = cmd.Flags().GetStringArray("field")

	if opts.Project == "" {
		opts.Project = viper.GetString("jira.default_project")
	}
	if opts.Project == "" {
		return fmt.Errorf("project required: use --project or set jira.default_project in config")
	}
	if opts.Assignee == "me" {
		opts.Assignee = viper.GetString("jira.username")
		if opts.Assignee == "" {
			opts.Assignee = viper.GetString("username")
		}
	}

	client, err := api.GetJiraClient()
	cmdutil.ExitIfError(err)

	var agile agileFieldIDs
	if opts.Epic != "" || opts.Sprint > 0 || opts.StoryPoints > 0 {
		agile, err = resolveAgileFields(ctx, client)
		if err != nil {
			return err
		}
	}

	fields, err := buildIssueFields(opts, agile)
	if err != nil {
		return err
	}

	created, err := client.CreateIssue(ctx, fields)
	if err != nil {
		return err
	}

	if jsonOut, _ := cmd.Flags().GetBool("json"); jsonOut {
		out := map[string]string{
			"key": created.Key,
			"id":  created.ID,
			"url": client.BaseURL + "/browse/" + created.Key,
		}
		return json.NewEncoder(cmd.OutOrStdout()).Encode(out)
	}

	fmt.Printf("Created %s: %s\n", created.Key, opts.Summary)
	fmt.Printf("URL: %s/browse/%s\n", client.BaseURL, created.Key)
	return nil
}

// buildIssueFields maps create options onto the JIRA create-issue fields
// payload. Agile field IDs are server-specific and passed in resolved.
func buildIssueFields(opts issueCreateOptions, agile agileFieldIDs) (map[string]interface{}, error) {
	if opts.Type == "" {
		return nil, fmt.Errorf("--type required")
	}
	if opts.Summary == "" {
		return nil, fmt.Errorf("--summary required")
	}

	fields := map[string]interface{}{
		"project":   map[string]string{"key": opts.Project},
		"issuetype": map[string]string{"name": opts.Type},
		"summary":   opts.Summary,
	}

	if opts.Description != "" {
		fields["description"] = opts.Description
	}
	if opts.Parent != "" {
		fields["parent"] = map[string]string{"key": prefixIssueKey(opts.Parent, opts.Project)}
	}
	if opts.Priority != "" {
		fields["priority"] = map[string]string{"name": opts.Priority}
	}
	if opts.Assignee != "" {
		fields["assignee"] = map[string]string{"name": opts.Assignee}
	}
	if len(opts.Labels) > 0 {
		fields["labels"] = opts.Labels
	}
	if len(opts.FixVersions) > 0 {
		versions := make([]map[string]string, len(opts.FixVersions))
		for i, v := range opts.FixVersions {
			versions[i] = map[string]string{"name": v}
		}
		fields["fixVersions"] = versions
	}
	if len(opts.Components) > 0 {
		components := make([]map[string]string, len(opts.Components))
		for i, c := range opts.Components {
			components[i] = map[string]string{"name": c}
		}
		fields["components"] = components
	}

	if opts.Epic != "" {
		if agile.EpicLink == "" {
			return nil, fmt.Errorf("epic link field not found on this JIRA instance")
		}
		fields[agile.EpicLink] = prefixIssueKey(opts.Epic, opts.Project)
	}
	if opts.Sprint > 0 {
		if agile.Sprint == "" {
			return nil, fmt.Errorf("sprint field not found on this JIRA instance")
		}
		fields[agile.Sprint] = opts.Sprint
	}
	if opts.StoryPoints > 0 {
		if agile.StoryPoints == "" {
			return nil, fmt.Errorf("story points field not found on this JIRA instance")
		}
		fields[agile.StoryPoints] = opts.StoryPoints
	}

	for _, f := range opts.RawFields {
		parts := strings.SplitN(f, "=", 2)
		if len(parts) != 2 {
			return nil, fmt.Errorf("invalid --field format %q, expected key=value", f)
		}
		fields[parts[0]] = parseFieldValue(parts[1])
	}

	return fields, nil
}

// parseFieldValue decodes JSON-looking values so --field can pass
// objects and arrays; anything else stays a plain string.
func parseFieldValue(val string) interface{} {
	if strings.HasPrefix(val, "[") || strings.HasPrefix(val, "{") {
		var parsed interface{}
		if err := json.Unmarshal([]byte(val), &parsed); err == nil {
			return parsed
		}
	}
	return val
}

// prefixIssueKey turns a bare issue number into PROJECT-NUMBER, matching
// the auto-prefix behavior of `issue list --epic`.
func prefixIssueKey(key, project string) string {
	if !strings.Contains(key, "-") && project != "" {
		return project + "-" + key
	}
	return key
}

func resolveAgileFields(ctx context.Context, client *api.JiraClient) (agileFieldIDs, error) {
	jiraFields, err := client.GetFields(ctx)
	if err != nil {
		return agileFieldIDs{}, fmt.Errorf("resolving agile fields: %w", err)
	}
	return findAgileFields(jiraFields), nil
}

func findAgileFields(jiraFields []api.JiraField) agileFieldIDs {
	var agile agileFieldIDs
	for _, f := range jiraFields {
		switch {
		case f.Schema.Custom == epicLinkSchema:
			agile.EpicLink = f.ID
		case f.Schema.Custom == sprintSchema:
			agile.Sprint = f.ID
		case f.Custom && strings.EqualFold(f.Name, "Story Points"):
			agile.StoryPoints = f.ID
		}
	}
	return agile
}

func init() {
	issueCmd.AddCommand(issueCreateCmd)

	f := issueCreateCmd.Flags()
	f.SortFlags = false

	f.StringP("type", "t", "", "Issue type (Story, Task, Bug, Sub-dev-task, ...) (required)")
	f.StringP("summary", "s", "", "Issue summary (required)")
	f.StringP("project", "p", "", "Project key (default from config)")
	f.StringP("description", "b", "", "Issue description")
	f.StringP("parent", "P", "", "Parent issue key (required for sub-task types)")
	f.StringP("priority", "y", "", "Priority name (Blocker, Critical, Major, Minor, Trivial)")
	f.StringP("assignee", "a", "", "Assignee username (use 'me' for current user)")
	f.StringP("epic", "e", "", "Epic link (issue key, auto-prefixes project if needed)")
	f.Int("sprint", 0, "Sprint ID (not valid for sub-task types)")
	f.Float64("story-points", 0, "Story points estimate")
	f.StringArrayP("label", "l", nil, "Label (repeatable)")
	f.StringSlice("fix-version", nil, "Fix version(s)")
	f.StringSliceP("component", "C", nil, "Component(s)")
	f.StringArray("field", nil, "Additional fields as key=value, value may be JSON (repeatable)")
	f.Bool("json", false, "Output created issue as JSON")

	_ = issueCreateCmd.MarkFlagRequired("type")
	_ = issueCreateCmd.MarkFlagRequired("summary")
}
