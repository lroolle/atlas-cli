package cmd

import (
	"fmt"
	"os"
	"os/exec"
	"strconv"

	"github.com/spf13/cobra"
)

var prCheckoutCmd = &cobra.Command{
	Use:   "checkout <pr-id>",
	Short: "Check out a pull request branch locally",
	Long: `Check out the source branch of a pull request locally.

This fetches the PR's source branch and creates a local branch named pr-<id>.`,
	Aliases: []string{"co"},
	Args:    cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		ctx := cmd.Context()

		prIDStr := args[0]
		prID, err := strconv.Atoi(prIDStr)
		if err != nil {
			return fmt.Errorf("invalid PR ID: %v", err)
		}

		project, repo, err := parseRepoArg("")
		if err != nil {
			return fmt.Errorf("could not determine repository: %w", err)
		}

		client, err := getClient()
		if err != nil {
			return err
		}

		pr, err := client.GetPullRequest(ctx, project, repo, prID)
		if err != nil {
			return fmt.Errorf("fetching PR: %w", err)
		}

		sourceBranch := pr.FromRef.DisplayID
		localBranch := fmt.Sprintf("pr-%d", prID)

		detach, _ := cmd.Flags().GetBool("detach")
		branchName, _ := cmd.Flags().GetString("branch")
		if branchName != "" {
			localBranch = branchName
		}

		force, _ := cmd.Flags().GetBool("force")

		fmt.Printf("Fetching PR #%d: %s\n", prID, pr.Title)
		fmt.Printf("  %s -> %s\n", pr.FromRef.DisplayID, pr.ToRef.DisplayID)

		fetchCmd := exec.Command("git", "fetch", "origin", sourceBranch)
		fetchCmd.Stdout = os.Stdout
		fetchCmd.Stderr = os.Stderr
		if err := fetchCmd.Run(); err != nil {
			return fmt.Errorf("fetching branch: %w", err)
		}

		if detach {
			checkoutCmd := exec.Command("git", "checkout", "--detach", "FETCH_HEAD")
			checkoutCmd.Stdout = os.Stdout
			checkoutCmd.Stderr = os.Stderr
			if err := checkoutCmd.Run(); err != nil {
				return fmt.Errorf("checking out: %w", err)
			}
			fmt.Printf("Checked out PR #%d in detached HEAD state\n", prID)
			return nil
		}

		if force {
			checkoutCmd := exec.Command("git", "checkout", "-B", localBranch, "FETCH_HEAD")
			checkoutCmd.Stdout = os.Stdout
			checkoutCmd.Stderr = os.Stderr
			if err := checkoutCmd.Run(); err != nil {
				return fmt.Errorf("checking out: %w", err)
			}
			fmt.Printf("Switched to branch '%s'\n", localBranch)
			return nil
		}

		if branchExists(localBranch) {
			checkoutCmd := exec.Command("git", "checkout", localBranch)
			checkoutCmd.Stdout = os.Stdout
			checkoutCmd.Stderr = os.Stderr
			if err := checkoutCmd.Run(); err != nil {
				return fmt.Errorf("checking out: %w", err)
			}

			mergeCmd := exec.Command("git", "merge", "--ff-only", "FETCH_HEAD")
			mergeCmd.Stdout = os.Stdout
			mergeCmd.Stderr = os.Stderr
			if err := mergeCmd.Run(); err != nil {
				return fmt.Errorf("could not fast-forward %q; use --force to overwrite: %w", localBranch, err)
			}

			fmt.Printf("Updated branch '%s'\n", localBranch)
			return nil
		}

		checkoutCmd := exec.Command("git", "checkout", "-b", localBranch, "FETCH_HEAD")
		checkoutCmd.Stdout = os.Stdout
		checkoutCmd.Stderr = os.Stderr
		if err := checkoutCmd.Run(); err != nil {
			return fmt.Errorf("checking out: %w", err)
		}
		fmt.Printf("Switched to branch '%s'\n", localBranch)
		return nil
	},
}

func branchExists(name string) bool {
	err := exec.Command("git", "rev-parse", "--verify", name).Run()
	return err == nil
}

func init() {
	prCmd.AddCommand(prCheckoutCmd)

	prCheckoutCmd.Flags().StringP("branch", "b", "", "Local branch name (default: pr-<id>)")
	prCheckoutCmd.Flags().Bool("detach", false, "Checkout in detached HEAD state")
	prCheckoutCmd.Flags().BoolP("force", "f", false, "Force overwrite local branch")
}
