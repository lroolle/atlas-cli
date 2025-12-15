package cmd

import (
	"fmt"
	"os"
	"strings"
	"text/tabwriter"

	"github.com/lroolle/atlas-cli/api"
	"github.com/lroolle/atlas-cli/internal/cmdutil"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

var issueCmd = &cobra.Command{
	Use:     "issue",
	Short:   "Manage JIRA issues",
	Long:    `Commands for listing, viewing, and managing JIRA issues`,
	Aliases: []string{"jira"},
}

var issueListCmd = &cobra.Command{
	Use:   "list [search text]",
	Short: "List JIRA issues",
	Long: `List JIRA issues with various filters.

You can combine flags to create complex queries:
  atl issue list -t Bug -s Open -e GAUSS-18421
  atl issue list -a me -y High --label backend

Use ~ prefix for negation (quote to prevent shell expansion):
  atl issue list -s '~Done'            # status != Done
  atl issue list --label '~wontfix'    # exclude label`,
	Example: `  atl issue list
  atl issue list -t Bug -s Open
  atl issue list -e GAUSS-18421 -a me
  atl issue list -e 18421              # auto-prefix with default project
  atl issue list -q "created >= -7d"
  atl issue list --order-by updated --reverse`,
	Aliases: []string{"ls", "search"},
	Args:    cobra.MaximumNArgs(1),
	RunE:    runIssueList,
}

func runIssueList(cmd *cobra.Command, args []string) error {
	ctx := cmd.Context()
	var conditions []string

	project, _ := cmd.Flags().GetString("project")
	if project == "" {
		project = viper.GetString("jira.default_project")
	}
	if project != "" {
		conditions = append(conditions, fmt.Sprintf("project = '%s'", escapeJQL(project)))
	}

	if val, _ := cmd.Flags().GetString("type"); val != "" {
		conditions = append(conditions, formatCondition("type", val))
	}

	if vals, _ := cmd.Flags().GetStringArray("status"); len(vals) > 0 {
		cond := formatMultiCondition("status", vals)
		if cond != "" {
			conditions = append(conditions, cond)
		}
	}

	if val, _ := cmd.Flags().GetString("priority"); val != "" {
		conditions = append(conditions, formatCondition("priority", val))
	}

	if val, _ := cmd.Flags().GetString("assignee"); val != "" {
		if val == "me" {
			conditions = append(conditions, "assignee = currentUser()")
		} else if val == "none" || val == "x" {
			conditions = append(conditions, "assignee IS EMPTY")
		} else {
			conditions = append(conditions, formatCondition("assignee", val))
		}
	}

	if val, _ := cmd.Flags().GetString("reporter"); val != "" {
		if val == "me" {
			conditions = append(conditions, "reporter = currentUser()")
		} else {
			conditions = append(conditions, formatCondition("reporter", val))
		}
	}

	if val, _ := cmd.Flags().GetString("epic"); val != "" {
		if !strings.Contains(val, "-") && project != "" {
			val = project + "-" + val
		}
		conditions = append(conditions, fmt.Sprintf(`"Epic Link" = '%s'`, escapeJQL(val)))
	}

	if val, _ := cmd.Flags().GetString("component"); val != "" {
		conditions = append(conditions, formatCondition("component", val))
	}

	if vals, _ := cmd.Flags().GetStringArray("label"); len(vals) > 0 {
		cond := formatMultiCondition("labels", vals)
		if cond != "" {
			conditions = append(conditions, cond)
		}
	}

	if val, _ := cmd.Flags().GetString("jql"); val != "" {
		if strings.Contains(strings.ToUpper(val), "ORDER BY") {
			return fmt.Errorf("--jql should not contain ORDER BY, use --order-by flag instead")
		}
		conditions = append(conditions, val)
	}

	if len(args) > 0 {
		conditions = append(conditions, fmt.Sprintf("text ~ '%s'", escapeJQL(args[0])))
	}

	jql := strings.Join(conditions, " AND ")

	orderBy, _ := cmd.Flags().GetString("order-by")
	validOrderFields := map[string]bool{
		"created": true, "updated": true, "priority": true, "status": true,
		"key": true, "assignee": true, "reporter": true, "summary": true,
	}
	if !validOrderFields[orderBy] {
		return fmt.Errorf("invalid --order-by field %q, valid: created, updated, priority, status, key, assignee, reporter, summary", orderBy)
	}

	reverse, _ := cmd.Flags().GetBool("reverse")
	direction := "DESC"
	if reverse {
		direction = "ASC"
	}
	if jql == "" {
		jql = fmt.Sprintf("ORDER BY %s %s", orderBy, direction)
	} else {
		jql += fmt.Sprintf(" ORDER BY %s %s", orderBy, direction)
	}

	limit, _ := cmd.Flags().GetInt("limit")
	if limit <= 0 {
		limit = cmdutil.DefaultLimit
	}

	client, err := api.GetJiraClient()
	cmdutil.ExitIfError(err)

	issues, err := client.SearchIssues(ctx, jql, limit)
	if err != nil {
		return err
	}

	if len(issues) == 0 {
		fmt.Println("No issues found")
		return nil
	}

	w := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
	fmt.Fprintln(w, "KEY\tSTATUS\tPRIORITY\tSUMMARY\tASSIGNEE")

	for _, issue := range issues {
		assignee := "Unassigned"
		if issue.Fields.Assignee != nil {
			assignee = issue.Fields.Assignee.DisplayName
		}
		status := "Unknown"
		if issue.Fields.Status.Name != "" {
			status = issue.Fields.Status.Name
		}
		priority := "None"
		if issue.Fields.Priority.Name != "" {
			priority = issue.Fields.Priority.Name
		}

		fmt.Fprintf(w, "%s\t%s\t%s\t%s\t%s\n",
			issue.Key,
			status,
			priority,
			cmdutil.Truncate(issue.Fields.Summary, cmdutil.TitleTruncateNormal),
			assignee,
		)
	}

	return w.Flush()
}

func escapeJQL(val string) string {
	return strings.ReplaceAll(val, "'", "''")
}

func formatCondition(field, val string) string {
	if strings.HasPrefix(val, "~") {
		return fmt.Sprintf("%s != '%s'", field, escapeJQL(val[1:]))
	}
	return fmt.Sprintf("%s = '%s'", field, escapeJQL(val))
}

func formatMultiCondition(field string, vals []string) string {
	var positive, negative []string
	for _, v := range vals {
		if strings.HasPrefix(v, "~") {
			negative = append(negative, v[1:])
		} else {
			positive = append(positive, v)
		}
	}

	var parts []string
	if len(positive) == 1 {
		parts = append(parts, fmt.Sprintf("%s = '%s'", field, escapeJQL(positive[0])))
	} else if len(positive) > 1 {
		quoted := make([]string, len(positive))
		for i, p := range positive {
			quoted[i] = fmt.Sprintf("'%s'", escapeJQL(p))
		}
		parts = append(parts, fmt.Sprintf("%s IN (%s)", field, strings.Join(quoted, ", ")))
	}

	for _, n := range negative {
		parts = append(parts, fmt.Sprintf("%s != '%s'", field, escapeJQL(n)))
	}

	return strings.Join(parts, " AND ")
}

var issueViewCmd = &cobra.Command{
	Use:   "view [issue-key]",
	Short: "View a JIRA issue",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		ctx := cmd.Context()

		client, err := api.GetJiraClient()
		cmdutil.ExitIfError(err)

		issue, err := client.GetIssue(ctx, args[0])
		if err != nil {
			return err
		}

		fmt.Printf("Issue: %s\n", issue.Key)
		fmt.Printf("Summary: %s\n", issue.Fields.Summary)
		fmt.Printf("Status: %s\n", issue.Fields.Status.Name)
		fmt.Printf("Priority: %s\n", issue.Fields.Priority.Name)
		fmt.Printf("Type: %s\n", issue.Fields.IssueType.Name)
		fmt.Printf("Reporter: %s\n", issue.Fields.Reporter.DisplayName)

		if issue.Fields.Assignee != nil {
			fmt.Printf("Assignee: %s\n", issue.Fields.Assignee.DisplayName)
		} else {
			fmt.Println("Assignee: Unassigned")
		}

		fmt.Printf("Created: %s\n", issue.Fields.Created)
		fmt.Printf("Updated: %s\n", issue.Fields.Updated)

		if issue.Fields.Resolution != nil {
			fmt.Printf("Resolution: %s\n", issue.Fields.Resolution.Name)
		}

		if issue.Fields.Description != "" {
			fmt.Printf("\nDescription:\n%s\n", issue.Fields.Description)
		}

		if len(issue.Fields.IssueLinks) > 0 {
			fmt.Printf("\nIssue Links:\n")
			for _, link := range issue.Fields.IssueLinks {
				if link.OutwardIssue != nil {
					fmt.Printf("  %s %s (%s)\n", link.Type.Outward, link.OutwardIssue.Key, link.OutwardIssue.Fields.Summary)
				}
				if link.InwardIssue != nil {
					fmt.Printf("  %s %s (%s)\n", link.Type.Inward, link.InwardIssue.Key, link.InwardIssue.Fields.Summary)
				}
			}
		}

		return nil
	},
}

var issueTransitionCmd = &cobra.Command{
	Use:   "transition [issue-key] [transition-name]",
	Short: "Transition a JIRA issue to a new status",
	Args:  cobra.RangeArgs(1, 2),
	RunE: func(cmd *cobra.Command, args []string) error {
		ctx := cmd.Context()

		client, err := api.GetJiraClient()
		cmdutil.ExitIfError(err)

		issueKey := args[0]

		transitions, err := client.GetTransitions(ctx, issueKey)
		if err != nil {
			return err
		}

		if len(args) == 1 {
			fmt.Println("Available transitions:")
			for _, t := range transitions {
				fmt.Printf("  %s -> %s\n", t.Name, t.To.Name)
			}
			return nil
		}

		transitionName := strings.ToLower(args[1])
		var targetTransition *api.Transition

		for _, t := range transitions {
			if strings.ToLower(t.Name) == transitionName || strings.ToLower(t.To.Name) == transitionName {
				targetTransition = &t
				break
			}
		}

		if targetTransition == nil {
			return fmt.Errorf("transition '%s' not found", args[1])
		}

		err = client.TransitionIssue(ctx, issueKey, targetTransition.ID)
		if err != nil {
			return err
		}

		fmt.Printf("Issue %s transitioned to %s\n", issueKey, targetTransition.To.Name)
		return nil
	},
}

var issueCommentCmd = &cobra.Command{
	Use:   "comment [issue-key] [comment]",
	Short: "Add a comment to a JIRA issue",
	Args:  cobra.ExactArgs(2),
	RunE: func(cmd *cobra.Command, args []string) error {
		ctx := cmd.Context()

		client, err := api.GetJiraClient()
		cmdutil.ExitIfError(err)

		err = client.AddComment(ctx, args[0], args[1])
		if err != nil {
			return err
		}

		fmt.Printf("Comment added to %s\n", args[0])
		return nil
	},
}

var issueCommentsCmd = &cobra.Command{
	Use:   "comments [issue-key]",
	Short: "Show comments for a JIRA issue",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		ctx := cmd.Context()

		client, err := api.GetJiraClient()
		cmdutil.ExitIfError(err)

		comments, err := client.GetComments(ctx, args[0])
		if err != nil {
			return err
		}

		if len(comments) == 0 {
			fmt.Printf("No comments found for %s\n", args[0])
			return nil
		}

		fmt.Printf("Comments for %s:\n\n", args[0])
		for i, comment := range comments {
			fmt.Printf("--- Comment %d ---\n", i+1)
			fmt.Printf("Author: %s\n", comment.Author.DisplayName)
			fmt.Printf("Created: %s\n", comment.Created)
			fmt.Printf("Body:\n%s\n\n", comment.Body)
		}

		return nil
	},
}

var issuePrsCmd = &cobra.Command{
	Use:   "prs [issue-key]",
	Short: "Show pull requests for a JIRA issue",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		ctx := cmd.Context()

		client, err := api.GetJiraClient()
		cmdutil.ExitIfError(err)

		issue, err := client.GetIssue(ctx, args[0])
		if err != nil {
			return err
		}

		devInfo, err := client.GetDevelopmentInfo(ctx, issue.ID)
		if err != nil {
			return err
		}

		totalPRs := 0
		for _, detail := range devInfo.Detail {
			totalPRs += len(detail.PullRequests)
		}

		if totalPRs == 0 {
			fmt.Printf("No pull requests found for %s\n", args[0])
			return nil
		}

		fmt.Printf("Pull Requests for %s (%d total):\n\n", args[0], totalPRs)

		for _, detail := range devInfo.Detail {
			for _, pr := range detail.PullRequests {
				fmt.Printf("PR %s: %s\n", pr.ID, pr.Name)
				fmt.Printf("URL: %s\n", pr.URL)
				fmt.Printf("Status: %s\n", pr.Status)
				fmt.Printf("Author: %s\n", pr.Author.Name)
				fmt.Printf("Last Updated: %s\n", pr.LastUpdate)
				fmt.Printf("Source: %s → %s\n", pr.Source.Branch, pr.Destination.Branch)
				fmt.Printf("Repository: %s\n", pr.Source.Repository.Name)

				if len(pr.Reviewers) > 0 {
					fmt.Printf("Reviewers:\n")
					for _, reviewer := range pr.Reviewers {
						status := "X"
						if reviewer.Approved {
							status = "✓"
						}
						fmt.Printf("  [%s] %s\n", status, reviewer.Name)
					}
				}
				fmt.Println("---")
			}
		}

		return nil
	},
}

func init() {
	rootCmd.AddCommand(issueCmd)
	issueCmd.AddCommand(issueListCmd)
	issueCmd.AddCommand(issueViewCmd)
	issueCmd.AddCommand(issueTransitionCmd)
	issueCmd.AddCommand(issueCommentCmd)
	issueCmd.AddCommand(issueCommentsCmd)
	issueCmd.AddCommand(issuePrsCmd)

	f := issueListCmd.Flags()
	f.SortFlags = false

	f.StringP("type", "t", "", "Filter by issue type (Bug, Story, Task, Epic)")
	f.StringArrayP("status", "s", nil, "Filter by status (use ~ for negation, e.g., '~Done')")
	f.StringP("priority", "y", "", "Filter by priority (Blocker, Critical, Major, Minor, Trivial)")
	f.StringP("assignee", "a", "", "Filter by assignee (use 'me' or 'none'/'x' for unassigned)")
	f.StringP("reporter", "r", "", "Filter by reporter (use 'me' for current user)")
	f.StringP("epic", "e", "", "Filter by epic link (issue key, auto-prefixes project if needed)")
	f.StringP("component", "C", "", "Filter by component")
	f.StringArrayP("label", "l", nil, "Filter by label (use ~ for negation)")
	f.StringP("project", "p", "", "Filter by project (default from config)")
	f.StringP("jql", "q", "", "Additional JQL conditions (no ORDER BY)")
	f.String("order-by", "created", "Order by field (created, updated, priority, status)")
	f.Bool("reverse", false, "Reverse sort order (ASC instead of DESC)")
	f.Int("limit", cmdutil.DefaultLimit, "Maximum number of results")
}
