package cmd

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/lroolle/atlas-cli/internal/cmdutil"
	"github.com/spf13/cobra"
)

var prMergeCmd = &cobra.Command{
	Use:   "merge [project/repo] [pr-id]",
	Short: "Merge a pull request",
	Args:  cobra.ExactArgs(2),
	RunE: func(cmd *cobra.Command, args []string) error {
		ctx := cmd.Context()
		parts := strings.Split(args[0], "/")
		if len(parts) != 2 {
			return fmt.Errorf("invalid format: use PROJECT/REPO")
		}

		prID, err := strconv.Atoi(args[1])
		if err != nil {
			return fmt.Errorf("invalid PR ID: %v", err)
		}

		client, err := getClient()
		if err != nil {
			return err
		}
		project, repo := parts[0], parts[1]

		// Get PR details first
		pr, err := client.GetPullRequest(ctx, project, repo, prID)
		if err != nil {
			return err
		}

		if pr.State != "OPEN" {
			return fmt.Errorf("PR #%d is not open (state: %s)", prID, pr.State)
		}

		canMerge, err := cmd.Flags().GetBool("force")
		cmdutil.ExitIfError(err)
		if !canMerge {
			// Check for approvals
			hasApproval := false
			for _, reviewer := range pr.Reviewers {
				if reviewer.Approved {
					hasApproval = true
					break
				}
			}

			if !hasApproval {
				return fmt.Errorf("PR #%d has no approvals. Use --force to merge anyway", prID)
			}
		}

		// Merge the PR
		if err := client.MergePullRequest(ctx, project, repo, prID, pr.Version); err != nil {
			return fmt.Errorf("merging PR: %w", err)
		}

		fmt.Printf("✓ Merged PR #%d: %s\n", prID, pr.Title)
		return nil
	},
}

func init() {
	prCmd.AddCommand(prMergeCmd)
	prMergeCmd.Flags().Bool("force", false, "Merge even without approvals")
	prMergeCmd.Flags().Bool("delete-branch", false, "Delete the source branch after merge")
}
