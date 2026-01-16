package cmd

import (
	"encoding/json"
	"fmt"
	"os"
	"strconv"
	"strings"
	"text/tabwriter"
	"time"

	"github.com/lroolle/atlas-cli/api"
	"github.com/lroolle/atlas-cli/internal/cmdutil"
	"github.com/spf13/cobra"
)

var prCmd = &cobra.Command{
	Use:   "pr",
	Short: "Manage pull requests",
	Long:  `Commands for listing, viewing, and managing pull requests`,
}

var prListCmd = &cobra.Command{
	Use:   "list [project/repo]",
	Short: "List pull requests in a repository",
	Args:  cobra.MaximumNArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		ctx := cmd.Context()
		var project, repo string

		var arg string
		if len(args) > 0 {
			arg = args[0]
		}

		var err error
		project, repo, err = parseRepoArg(arg)
		if err != nil {
			return err
		}

		client, err := getClient()
		if err != nil {
			return err
		}

		jsonOutput, _ := cmd.Flags().GetBool("json")
		web, _ := cmd.Flags().GetBool("web")
		if web {
			if jsonOutput {
				return fmt.Errorf("--web and --json are mutually exclusive")
			}
			url := fmt.Sprintf("%s/projects/%s/repos/%s/pull-requests", client.Client.BaseURL, project, repo)
			if err := openBrowser(url); err != nil {
				return fmt.Errorf("opening browser: %w", err)
			}
			return nil
		}

		state, err := cmd.Flags().GetString("state")
		cmdutil.ExitIfError(err)
		limit, err := cmd.Flags().GetInt("limit")
		cmdutil.ExitIfError(err)
		author, err := cmd.Flags().GetString("author")
		cmdutil.ExitIfError(err)
		base, err := cmd.Flags().GetString("base")
		cmdutil.ExitIfError(err)
		head, err := cmd.Flags().GetString("head")
		cmdutil.ExitIfError(err)

		prs, err := client.ListPullRequests(ctx, project, repo, state, limit)
		if err != nil {
			return err
		}

		if author != "" {
			var filtered []api.PullRequest
			for _, pr := range prs {
				if author == "@me" && pr.Author.User.Name == client.Username ||
					strings.Contains(strings.ToLower(pr.Author.User.Name), strings.ToLower(author)) ||
					strings.Contains(strings.ToLower(pr.Author.User.DisplayName), strings.ToLower(author)) {
					filtered = append(filtered, pr)
				}
			}
			prs = filtered
		}

		base = normalizeBranchName(base)
		head = normalizeBranchName(head)
		if base != "" {
			var filtered []api.PullRequest
			for _, pr := range prs {
				if branchMatches(pr.ToRef, base) {
					filtered = append(filtered, pr)
				}
			}
			prs = filtered
		}
		if head != "" {
			var filtered []api.PullRequest
			for _, pr := range prs {
				if branchMatches(pr.FromRef, head) {
					filtered = append(filtered, pr)
				}
			}
			prs = filtered
		}

		if len(prs) == 0 {
			fmt.Println("No pull requests found")
			return nil
		}

		if jsonOutput {
			return json.NewEncoder(os.Stdout).Encode(prs)
		}

		w := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
		fmt.Fprintln(w, "#\tSTATUS\tTITLE\tBRANCH\tAUTHOR")

		for _, pr := range prs {
			status := pr.State
			if pr.State == "OPEN" {
				status = "OPEN  "
			}
			fmt.Fprintf(w, "#%d\t%s\t%s\t%s\t%s\n",
				pr.ID,
				status,
				cmdutil.Truncate(pr.Title, cmdutil.TitleTruncateNormal),
				cmdutil.Truncate(pr.FromRef.DisplayID, 20),
				pr.Author.User.Name,
			)
		}

		return w.Flush()
	},
}

var prViewCmd = &cobra.Command{
	Use:   "view [project/repo] [pr-id]",
	Short: "View a pull request",
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
				return fmt.Errorf("PR ID or PROJECT/REPO PR-ID required: %w", err)
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

		jsonOutput, _ := cmd.Flags().GetBool("json")
		web, _ := cmd.Flags().GetBool("web")
		if web && jsonOutput {
			return fmt.Errorf("--web and --json are mutually exclusive")
		}

		prURL := getPRURL(client.Client.BaseURL, project, repo, prID)

		if web {
			if err := openBrowser(prURL); err != nil {
				return fmt.Errorf("opening browser: %w", err)
			}
			return nil
		}

		pr, err := client.GetPullRequest(ctx, project, repo, prID)
		if err != nil {
			return err
		}

		if jsonOutput {
			return json.NewEncoder(os.Stdout).Encode(pr)
		}

		fmt.Printf("PR #%d: %s\n", pr.ID, pr.Title)
		fmt.Printf("State: %s\n", pr.State)
		fmt.Printf("Author: %s (%s)\n", pr.Author.User.DisplayName, pr.Author.User.EmailAddress)
		fmt.Printf("From: %s\n", pr.FromRef.DisplayID)
		fmt.Printf("To: %s\n", pr.ToRef.DisplayID)
		fmt.Printf("Created: %s\n", time.Unix(pr.CreatedDate/1000, 0).Format("2006-01-02 15:04:05"))
		fmt.Printf("Updated: %s\n", time.Unix(pr.UpdatedDate/1000, 0).Format("2006-01-02 15:04:05"))
		fmt.Printf("URL: %s\n", prURL)

		if pr.Description != "" {
			fmt.Printf("\nDescription:\n%s\n", pr.Description)
		}

		if len(pr.Reviewers) > 0 {
			fmt.Println("\nReviewers:")
			for _, reviewer := range pr.Reviewers {
				status := "PENDING"
				if reviewer.Approved {
					status = "APPROVED"
				}
				fmt.Printf("  - %s: %s\n", reviewer.User.DisplayName, status)
			}
		}

		return nil
	},
}

var prDiffCmd = &cobra.Command{
	Use:   "diff [project/repo] [pr-id]",
	Short: "View pull request diff",
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
				return fmt.Errorf("PR ID or PROJECT/REPO PR-ID required: %w", err)
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
		diff, err := client.GetPullRequestDiff(ctx, project, repo, prID)
		if err != nil {
			return err
		}

		fmt.Println(diff)
		return nil
	},
}

var prCommentCmd = &cobra.Command{
	Use:   "comment [project/repo] [pr-id] [text]",
	Short: "Add a comment to a pull request",
	Args:  cobra.RangeArgs(2, 3),
	RunE: func(cmd *cobra.Command, args []string) error {
		ctx := cmd.Context()
		var project, repo string
		var prIDStr, text string

		if len(args) == 3 {
			var err error
			project, repo, err = parseRepoArg(args[0])
			if err != nil {
				return err
			}
			prIDStr = args[1]
			text = args[2]
		} else {
			var err error
			project, repo, err = parseRepoArg("")
			if err != nil {
				return fmt.Errorf("PR ID and text required: %w", err)
			}
			prIDStr = args[0]
			text = args[1]
		}

		prID, err := strconv.Atoi(prIDStr)
		if err != nil {
			return fmt.Errorf("invalid PR ID: %v", err)
		}

		client, err := getClient()
		if err != nil {
			return err
		}
		comment, err := client.AddPullRequestComment(ctx, project, repo, prID, text)
		if err != nil {
			return err
		}

		fmt.Printf("Comment added (ID: %d)\n", comment.ID)
		return nil
	},
}

func init() {
	rootCmd.AddCommand(prCmd)
	prCmd.AddCommand(prListCmd)
	prCmd.AddCommand(prViewCmd)
	prCmd.AddCommand(prDiffCmd)
	prCmd.AddCommand(prCommentCmd)

	prListCmd.Flags().String("state", "OPEN", "Filter by state (OPEN, MERGED, DECLINED, ALL)")
	prListCmd.Flags().Int("limit", cmdutil.DefaultLimit, "Maximum number of results")
	prListCmd.Flags().String("author", "", "Filter by author (@me for your PRs)")
	prListCmd.Flags().String("base", "", "Filter by base branch")
	prListCmd.Flags().String("head", "", "Filter by head branch")
	prListCmd.Flags().Bool("json", false, "Output as JSON")
	prListCmd.Flags().BoolP("web", "w", false, "Open in browser")

	prViewCmd.Flags().Bool("json", false, "Output as JSON")
	prViewCmd.Flags().BoolP("web", "w", false, "Open in browser")
}

func getClient() (*api.BitbucketClient, error) {
	client, err := api.GetBitbucketClient()
	if err != nil {
		return nil, fmt.Errorf("failed to get Bitbucket client: %w (hint: set credentials in config file under 'bitbucket' section)", err)
	}
	return client, nil
}

func normalizeBranchName(s string) string {
	s = strings.TrimSpace(s)
	s = strings.TrimPrefix(s, "refs/heads/")
	return s
}

func branchMatches(ref api.Ref, want string) bool {
	want = normalizeBranchName(want)
	if want == "" {
		return true
	}
	if strings.EqualFold(normalizeBranchName(ref.DisplayID), want) {
		return true
	}
	if strings.EqualFold(normalizeBranchName(ref.ID), want) {
		return true
	}
	return false
}
