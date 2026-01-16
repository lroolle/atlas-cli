package cmd

import (
	"fmt"
	"strconv"

	"github.com/lroolle/atlas-cli/api"
	"github.com/spf13/cobra"
)

var prEditCmd = &cobra.Command{
	Use:   "edit [project/repo] <pr-id>",
	Short: "Edit pull request properties",
	Long: `Edit a pull request's title, description, base branch, or reviewers.

Examples:
  atl pr edit 123 --title "New title"
  atl pr edit 123 --add-reviewer alice --add-reviewer bob
  atl pr edit 123 --remove-reviewer charlie`,
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

		pr, err := client.GetPullRequest(ctx, project, repo, prID)
		if err != nil {
			return fmt.Errorf("fetching PR: %w", err)
		}

		title, _ := cmd.Flags().GetString("title")
		body, _ := cmd.Flags().GetString("body")
		base, _ := cmd.Flags().GetString("base")
		addReviewers, _ := cmd.Flags().GetStringSlice("add-reviewer")
		removeReviewers, _ := cmd.Flags().GetStringSlice("remove-reviewer")

		hasUpdates := title != "" || body != "" || base != ""

		for _, r := range removeReviewers {
			if err := client.RemoveReviewer(ctx, project, repo, prID, r); err != nil {
				fmt.Printf("Warning: could not remove reviewer %s: %v\n", r, err)
			} else {
				fmt.Printf("Removed reviewer: %s\n", r)
			}
		}

		for _, r := range addReviewers {
			if err := client.AddReviewer(ctx, project, repo, prID, r); err != nil {
				fmt.Printf("Warning: could not add reviewer %s: %v\n", r, err)
			} else {
				fmt.Printf("Added reviewer: %s\n", r)
			}
		}

		if hasUpdates {
			opts := api.UpdatePROptions{
				Title:       title,
				Description: body,
				ToRef:       base,
				Version:     pr.Version,
			}

			updatedPR, err := client.UpdatePullRequest(ctx, project, repo, prID, opts)
			if err != nil {
				return fmt.Errorf("updating PR: %w", err)
			}

			fmt.Printf("Updated PR #%d: %s\n", prID, updatedPR.Title)
		} else if len(addReviewers) == 0 && len(removeReviewers) == 0 {
			return fmt.Errorf("no changes specified")
		}

		return nil
	},
}

func init() {
	prCmd.AddCommand(prEditCmd)

	prEditCmd.Flags().StringP("title", "t", "", "New title")
	prEditCmd.Flags().StringP("body", "b", "", "New description")
	prEditCmd.Flags().StringP("base", "B", "", "Change target branch")
	prEditCmd.Flags().StringSlice("add-reviewer", nil, "Add reviewer (can be repeated)")
	prEditCmd.Flags().StringSlice("remove-reviewer", nil, "Remove reviewer (can be repeated)")
}
