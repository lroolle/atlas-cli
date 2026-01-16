package cmd

import (
	"fmt"
	"os"
	"os/exec"
	"strings"

	"github.com/spf13/cobra"
)

var prCreateCmd = &cobra.Command{
	Use:   "create [project/repo]",
	Short: "Create a pull request",
	Long: `Create a new pull request from the current branch.

If no title is provided and --fill is used, the title will be derived from
the first commit message on the branch.`,
	Args: cobra.MaximumNArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		ctx := cmd.Context()

		var arg string
		if len(args) > 0 {
			arg = args[0]
		}

		project, repo, err := parseRepoArg(arg)
		if err != nil {
			return err
		}

		client, err := getClient()
		if err != nil {
			return err
		}

		title, _ := cmd.Flags().GetString("title")
		body, _ := cmd.Flags().GetString("body")
		base, _ := cmd.Flags().GetString("base")
		head, _ := cmd.Flags().GetString("head")
		reviewers, _ := cmd.Flags().GetStringSlice("reviewer")
		fill, _ := cmd.Flags().GetBool("fill")
		web, _ := cmd.Flags().GetBool("web")

		if head == "" {
			head, err = getCurrentGitBranch()
			if err != nil {
				return fmt.Errorf("could not determine current branch: %w", err)
			}
		}

		if base == "" {
			base, err = client.GetDefaultBranch(ctx, project, repo)
			if err != nil {
				base = "main"
			}
		}

		if fill && title == "" {
			title, body = fillFromCommits(base, head)
		}

		if title == "" {
			return fmt.Errorf("--title is required (or use --fill to derive from commits)")
		}

		pr, err := client.CreatePullRequest(ctx, project, repo, title, body, head, base, reviewers)
		if err != nil {
			return fmt.Errorf("creating pull request: %w", err)
		}

		prURL := getPRURL(client.Client.BaseURL, project, repo, pr.ID)

		if web {
			if err := openBrowser(prURL); err != nil {
				fmt.Fprintf(os.Stderr, "Warning: could not open browser: %v\n", err)
			}
		}

		fmt.Printf("Created PR #%d: %s\n", pr.ID, pr.Title)
		fmt.Printf("%s -> %s\n", pr.FromRef.DisplayID, pr.ToRef.DisplayID)
		fmt.Printf("%s\n", prURL)

		return nil
	},
}

func getCurrentGitBranch() (string, error) {
	out, err := exec.Command("git", "rev-parse", "--abbrev-ref", "HEAD").Output()
	if err != nil {
		return "", err
	}
	return strings.TrimSpace(string(out)), nil
}

func fillFromCommits(base, head string) (title, body string) {
	out, err := exec.Command("git", "log", "--format=%s", "--reverse", base+".."+head).Output()
	if err != nil {
		return "", ""
	}

	lines := strings.Split(strings.TrimSpace(string(out)), "\n")
	if len(lines) == 0 {
		return "", ""
	}

	title = lines[0]
	if len(lines) > 1 {
		body = strings.Join(lines[1:], "\n")
	}

	return title, body
}

func getPRURL(baseURL, project, repo string, prID int) string {
	return fmt.Sprintf("%s/projects/%s/repos/%s/pull-requests/%d", baseURL, project, repo, prID)
}

func init() {
	prCmd.AddCommand(prCreateCmd)

	prCreateCmd.Flags().StringP("title", "t", "", "Title for the pull request")
	prCreateCmd.Flags().StringP("body", "b", "", "Body/description for the pull request")
	prCreateCmd.Flags().StringP("base", "B", "", "Base branch (default: repo default branch)")
	prCreateCmd.Flags().StringP("head", "H", "", "Head branch (default: current branch)")
	prCreateCmd.Flags().StringSliceP("reviewer", "r", nil, "Request review from these users")
	prCreateCmd.Flags().Bool("fill", false, "Use commit messages to fill title and body")
	prCreateCmd.Flags().BoolP("web", "w", false, "Open the PR in browser after creation")
}
