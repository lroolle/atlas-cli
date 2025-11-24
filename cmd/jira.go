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
	Use:   "issue",
	Short: "Manage JIRA issues",
	Long:  `Commands for listing, viewing, and managing JIRA issues`,
	Aliases: []string{"jira"},
}

var issueListCmd = &cobra.Command{
	Use:   "list [jql]",
	Short: "List JIRA issues",
	Args:  cobra.MinimumNArgs(0),
	RunE: func(cmd *cobra.Command, args []string) error {
		ctx := cmd.Context()

		jql := ""
		if len(args) > 0 {
			jql = strings.Join(args, " ")
		}

		assignee, err := cmd.Flags().GetString("assignee")
		cmdutil.ExitIfError(err)
		var conditions []string

		if assignee != "" {
			if assignee == "me" {
				assignee = "currentUser()"
			}
			conditions = append(conditions, fmt.Sprintf("assignee = %s", assignee))
		}

		project, err := cmd.Flags().GetString("project")
		cmdutil.ExitIfError(err)
		if project == "" {
			project = viper.GetString("jira.default_project")
		}
		if project != "" {
			conditions = append(conditions, fmt.Sprintf("project = %s", project))
		}

		status, err := cmd.Flags().GetString("status")
		cmdutil.ExitIfError(err)
		if status != "" {
			conditions = append(conditions, fmt.Sprintf("status = '%s'", status))
		}

		if len(conditions) > 0 {
			if jql != "" {
				jql = strings.Join(conditions, " AND ") + " AND " + jql
			} else {
				jql = strings.Join(conditions, " AND ")
			}
		}

		if jql == "" {
			jql = "ORDER BY created DESC"
		} else if !strings.Contains(strings.ToUpper(jql), "ORDER BY") {
			jql += " ORDER BY created DESC"
		}

		limit, err := cmd.Flags().GetInt("limit")
		cmdutil.ExitIfError(err)

		client, err := api.GetJiraClient()
		cmdutil.ExitIfError(err)

		issues, err := client.SearchIssues(ctx, jql, limit)
		if err != nil {
			return err
		}

		w := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
		fmt.Fprintln(w, "KEY\tSTATUS\tPRIORITY\tSUMMARY\tASSIGNEE")

		for _, issue := range issues {
			assignee := "Unassigned"
			if issue.Fields.Assignee != nil {
				assignee = issue.Fields.Assignee.DisplayName
			}

			fmt.Fprintf(w, "%s\t%s\t%s\t%s\t%s\n",
				issue.Key,
				issue.Fields.Status.Name,
				issue.Fields.Priority.Name,
				cmdutil.Truncate(issue.Fields.Summary, cmdutil.TitleTruncateNormal),
				assignee,
			)
		}

		return w.Flush()
	},
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
	
	issueListCmd.Flags().String("assignee", "", "Filter by assignee (use 'me' for current user)")
	issueListCmd.Flags().String("project", "", "Filter by project")
	issueListCmd.Flags().String("status", "", "Filter by status")
	issueListCmd.Flags().Int("limit", cmdutil.DefaultLimit, "Maximum number of results")
}

