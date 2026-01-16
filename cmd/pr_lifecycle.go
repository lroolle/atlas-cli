package cmd

import (
	"fmt"
	"strconv"

	"github.com/spf13/cobra"
)

var prDeclineCmd = &cobra.Command{
	Use:     "decline [project/repo] <pr-id>",
	Short:   "Decline a pull request",
	Aliases: []string{"close"},
	Args:    cobra.RangeArgs(1, 2),
	RunE: func(cmd *cobra.Command, args []string) error {
		ctx := cmd.Context()
		var project, repo string
		var prIDStr string

		if len(args) == 2 {
			var err error
			project, repo, err = parseRepoArg(args[0])
			if err != nil {
				return err
			}
			prIDStr = args[1]
		} else {
			var err error
			project, repo, err = parseRepoArg("")
			if err != nil {
				return fmt.Errorf("PR ID required: %w", err)
			}
			prIDStr = args[0]
		}

		prID, err := strconv.Atoi(prIDStr)
		if err != nil {
			return fmt.Errorf("invalid PR ID: %v", err)
		}

		client, err := getClient()
		if err != nil {
			return err
		}

		pr, err := client.GetPullRequest(ctx, project, repo, prID)
		if err != nil {
			return fmt.Errorf("fetching PR: %w", err)
		}

		if pr.State != "OPEN" {
			return fmt.Errorf("PR #%d is not open (state: %s)", prID, pr.State)
		}

		comment, _ := cmd.Flags().GetString("comment")
		if comment != "" {
			if _, err := client.AddPullRequestComment(ctx, project, repo, prID, comment); err != nil {
				return fmt.Errorf("adding comment: %w", err)
			}
		}

		if err := client.DeclinePullRequest(ctx, project, repo, prID, pr.Version); err != nil {
			return fmt.Errorf("declining PR: %w", err)
		}

		fmt.Printf("✗ Declined PR #%d: %s\n", prID, pr.Title)
		return nil
	},
}

var prReopenCmd = &cobra.Command{
	Use:   "reopen [project/repo] <pr-id>",
	Short: "Reopen a declined pull request",
	Args:  cobra.RangeArgs(1, 2),
	RunE: func(cmd *cobra.Command, args []string) error {
		ctx := cmd.Context()
		var project, repo string
		var prIDStr string

		if len(args) == 2 {
			var err error
			project, repo, err = parseRepoArg(args[0])
			if err != nil {
				return err
			}
			prIDStr = args[1]
		} else {
			var err error
			project, repo, err = parseRepoArg("")
			if err != nil {
				return fmt.Errorf("PR ID required: %w", err)
			}
			prIDStr = args[0]
		}

		prID, err := strconv.Atoi(prIDStr)
		if err != nil {
			return fmt.Errorf("invalid PR ID: %v", err)
		}

		client, err := getClient()
		if err != nil {
			return err
		}

		pr, err := client.GetPullRequest(ctx, project, repo, prID)
		if err != nil {
			return fmt.Errorf("fetching PR: %w", err)
		}

		if pr.State != "DECLINED" {
			return fmt.Errorf("PR #%d is not declined (state: %s)", prID, pr.State)
		}

		if err := client.ReopenPullRequest(ctx, project, repo, prID, pr.Version); err != nil {
			return fmt.Errorf("reopening PR: %w", err)
		}

		fmt.Printf("✓ Reopened PR #%d: %s\n", prID, pr.Title)
		return nil
	},
}

var prRebaseCmd = &cobra.Command{
	Use:   "rebase [project/repo] <pr-id>",
	Short: "Rebase a pull request (server-side)",
	Long:  `Rebase the pull request's source branch on top of the target branch using server-side rebase.`,
	Args:  cobra.RangeArgs(1, 2),
	RunE: func(cmd *cobra.Command, args []string) error {
		ctx := cmd.Context()
		var project, repo string
		var prIDStr string

		if len(args) == 2 {
			var err error
			project, repo, err = parseRepoArg(args[0])
			if err != nil {
				return err
			}
			prIDStr = args[1]
		} else {
			var err error
			project, repo, err = parseRepoArg("")
			if err != nil {
				return fmt.Errorf("PR ID required: %w", err)
			}
			prIDStr = args[0]
		}

		prID, err := strconv.Atoi(prIDStr)
		if err != nil {
			return fmt.Errorf("invalid PR ID: %v", err)
		}

		client, err := getClient()
		if err != nil {
			return err
		}

		pr, err := client.GetPullRequest(ctx, project, repo, prID)
		if err != nil {
			return fmt.Errorf("fetching PR: %w", err)
		}

		if pr.State != "OPEN" {
			return fmt.Errorf("PR #%d is not open (state: %s)", prID, pr.State)
		}

		if err := client.RebasePullRequest(ctx, project, repo, prID, pr.Version); err != nil {
			return fmt.Errorf("rebasing PR: %w", err)
		}

		fmt.Printf("✓ Rebased PR #%d: %s\n", prID, pr.Title)
		return nil
	},
}

func init() {
	prCmd.AddCommand(prDeclineCmd)
	prCmd.AddCommand(prReopenCmd)
	prCmd.AddCommand(prRebaseCmd)

	prDeclineCmd.Flags().StringP("comment", "c", "", "Add a comment when declining")
}
