package cmd

import (
	"fmt"
	"strconv"

	"github.com/spf13/cobra"
)

var prReviewCmd = &cobra.Command{
	Use:   "review [project/repo] <pr-id>",
	Short: "Add a review to a pull request",
	Long: `Add a review to a pull request with optional comment.

Review actions:
  --approve (-a)           Approve the pull request
  --request-changes (-r)   Request changes (sets NEEDS_WORK status)
  --comment (-c)           Add a comment without changing approval status

If no action is specified, --comment is assumed when --body is provided.`,
	Args: cobra.RangeArgs(1, 2),
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

		approve, _ := cmd.Flags().GetBool("approve")
		requestChanges, _ := cmd.Flags().GetBool("request-changes")
		comment, _ := cmd.Flags().GetBool("comment")
		body, _ := cmd.Flags().GetString("body")

		actionCount := 0
		if approve {
			actionCount++
		}
		if requestChanges {
			actionCount++
		}
		if comment {
			actionCount++
		}

		if actionCount > 1 {
			return fmt.Errorf("only one of --approve, --request-changes, or --comment can be specified")
		}

		if actionCount == 0 && body != "" {
			comment = true
		}

		if actionCount == 0 && body == "" {
			return fmt.Errorf("specify --approve, --request-changes, --comment, or provide --body")
		}
		if comment && body == "" {
			return fmt.Errorf("--comment requires --body")
		}

		pr, err := client.GetPullRequest(ctx, project, repo, prID)
		if err != nil {
			return fmt.Errorf("fetching PR: %w", err)
		}

		if approve {
			if err := client.ApprovePullRequest(ctx, project, repo, prID); err != nil {
				return fmt.Errorf("approving PR: %w", err)
			}
			fmt.Printf("✓ Approved PR #%d: %s\n", prID, pr.Title)
		}

		if requestChanges {
			if err := client.SetReviewerStatus(ctx, project, repo, prID, "NEEDS_WORK"); err != nil {
				return fmt.Errorf("requesting changes: %w", err)
			}
			fmt.Printf("✗ Requested changes on PR #%d: %s\n", prID, pr.Title)
		}

		if body != "" {
			if _, err := client.AddPullRequestComment(ctx, project, repo, prID, body); err != nil {
				return fmt.Errorf("adding comment: %w", err)
			}
			if comment && !approve && !requestChanges {
				fmt.Printf("Commented on PR #%d: %s\n", prID, pr.Title)
			}
		}

		return nil
	},
}

var prApproveCmd = &cobra.Command{
	Use:   "approve [project/repo] <pr-id>",
	Short: "Approve a pull request",
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

		if err := client.ApprovePullRequest(ctx, project, repo, prID); err != nil {
			return fmt.Errorf("approving PR: %w", err)
		}

		fmt.Printf("✓ Approved PR #%d: %s\n", prID, pr.Title)
		return nil
	},
}

var prUnapproveCmd = &cobra.Command{
	Use:   "unapprove [project/repo] <pr-id>",
	Short: "Remove approval from a pull request",
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

		if err := client.UnapprovePullRequest(ctx, project, repo, prID); err != nil {
			return fmt.Errorf("removing approval: %w", err)
		}

		fmt.Printf("Removed approval from PR #%d: %s\n", prID, pr.Title)
		return nil
	},
}

func init() {
	prCmd.AddCommand(prReviewCmd)
	prCmd.AddCommand(prApproveCmd)
	prCmd.AddCommand(prUnapproveCmd)

	prReviewCmd.Flags().BoolP("approve", "a", false, "Approve the pull request")
	prReviewCmd.Flags().BoolP("request-changes", "r", false, "Request changes")
	prReviewCmd.Flags().BoolP("comment", "c", false, "Add comment only")
	prReviewCmd.Flags().StringP("body", "b", "", "Comment text")
}
