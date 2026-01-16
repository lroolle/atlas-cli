package cmd

import (
	"fmt"
	"strconv"

	"github.com/spf13/cobra"
)

var prMergeCmd = &cobra.Command{
	Use:   "merge [project/repo] <pr-id>",
	Short: "Merge a pull request",
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

		// Get PR details first
		pr, err := client.GetPullRequest(ctx, project, repo, prID)
		if err != nil {
			return err
		}

		if pr.State != "OPEN" {
			return fmt.Errorf("PR #%d is not open (state: %s)", prID, pr.State)
		}

		force, _ := cmd.Flags().GetBool("force")
		deleteBranch, _ := cmd.Flags().GetBool("delete-branch")
		if !force {
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

		if deleteBranch {
			sourceProject := pr.FromRef.Repository.Project.Key
			sourceRepo := pr.FromRef.Repository.Slug
			sourceBranch := pr.FromRef.ID
			if err := client.DeleteBranch(ctx, sourceProject, sourceRepo, sourceBranch); err != nil {
				return fmt.Errorf("merged PR #%d but failed to delete branch %s/%s %s: %w", prID, sourceProject, sourceRepo, sourceBranch, err)
			}
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
