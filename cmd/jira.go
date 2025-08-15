package cmd

import (
	"fmt"
	"os"
	"strings"
	"text/tabwriter"

	"github.com/lroolle/atlas-cli/api"
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
		jql := ""
		if len(args) > 0 {
			jql = strings.Join(args, " ")
		}
		
		assignee, _ := cmd.Flags().GetString("assignee")
		var conditions []string
		
		if assignee != "" {
			if assignee == "me" {
				assignee = "currentUser()"
			}
			conditions = append(conditions, fmt.Sprintf("assignee = %s", assignee))
		}
		
		project, _ := cmd.Flags().GetString("project")
		if project == "" {
			project = viper.GetString("jira.default_project")
		}
		if project != "" {
			conditions = append(conditions, fmt.Sprintf("project = %s", project))
		}
		
		status, _ := cmd.Flags().GetString("status")
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
		
		limit, _ := cmd.Flags().GetInt("limit")
		
		client := getJiraClient()
		issues, err := client.SearchIssues(jql, limit)
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
				truncate(issue.Fields.Summary, 50),
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
		client := getJiraClient()
		issue, err := client.GetIssue(args[0])
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
		
		return nil
	},
}

var issueTransitionCmd = &cobra.Command{
	Use:   "transition [issue-key] [transition-name]",
	Short: "Transition a JIRA issue to a new status",
	Args:  cobra.RangeArgs(1, 2),
	RunE: func(cmd *cobra.Command, args []string) error {
		client := getJiraClient()
		issueKey := args[0]
		
		transitions, err := client.GetTransitions(issueKey)
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
		
		err = client.TransitionIssue(issueKey, targetTransition.ID)
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
		client := getJiraClient()
		err := client.AddComment(args[0], args[1])
		if err != nil {
			return err
		}
		
		fmt.Printf("Comment added to %s\n", args[0])
		return nil
	},
}

func init() {
	rootCmd.AddCommand(issueCmd)
	issueCmd.AddCommand(issueListCmd)
	issueCmd.AddCommand(issueViewCmd)
	issueCmd.AddCommand(issueTransitionCmd)
	issueCmd.AddCommand(issueCommentCmd)
	
	issueListCmd.Flags().String("assignee", "", "Filter by assignee (use 'me' for current user)")
	issueListCmd.Flags().String("project", "", "Filter by project")
	issueListCmd.Flags().String("status", "", "Filter by status")
	issueListCmd.Flags().Int("limit", 25, "Maximum number of results")
}

func getJiraClient() *api.JiraClient {
	client, err := api.GetJiraClient()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		fmt.Fprintln(os.Stderr, "Set in config file under 'jira' section")
		os.Exit(1)
	}
	return client
}