package cmd

import (
	"encoding/json"
	"fmt"
	"os"
	"strconv"
	"text/tabwriter"
	"time"

	"github.com/lroolle/atlas-cli/internal/cmdutil"
	"github.com/spf13/cobra"
)

var prCommitsCmd = &cobra.Command{
	Use:   "commits [project/repo] <pr-id>",
	Short: "List commits in a pull request",
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

		limit, _ := cmd.Flags().GetInt("limit")
		jsonOutput, _ := cmd.Flags().GetBool("json")

		commits, err := client.GetPullRequestCommits(ctx, project, repo, prID, limit)
		if err != nil {
			return fmt.Errorf("fetching commits: %w", err)
		}

		if jsonOutput {
			return json.NewEncoder(os.Stdout).Encode(commits)
		}

		if len(commits) == 0 {
			fmt.Println("No commits found")
			return nil
		}

		w := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
		fmt.Fprintln(w, "SHA\tMESSAGE\tAUTHOR\tDATE")

		for _, c := range commits {
			date := time.Unix(c.AuthorTimestamp/1000, 0).Format("2006-01-02")
			fmt.Fprintf(w, "%s\t%s\t%s\t%s\n",
				c.DisplayID,
				cmdutil.Truncate(firstLine(c.Message), cmdutil.TitleTruncateNormal),
				c.Author.Name,
				date,
			)
		}

		return w.Flush()
	},
}

var prFilesCmd = &cobra.Command{
	Use:     "files [project/repo] <pr-id>",
	Short:   "List files changed in a pull request",
	Aliases: []string{"changes"},
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

		limit, _ := cmd.Flags().GetInt("limit")
		jsonOutput, _ := cmd.Flags().GetBool("json")

		changes, err := client.GetPullRequestChanges(ctx, project, repo, prID, limit)
		if err != nil {
			return fmt.Errorf("fetching changes: %w", err)
		}

		if jsonOutput {
			return json.NewEncoder(os.Stdout).Encode(changes)
		}

		if len(changes) == 0 {
			fmt.Println("No files changed")
			return nil
		}

		for _, c := range changes {
			status := changeTypeSymbol(c.Type)
			path := c.Path.ToString
			if c.SrcPath != nil && c.SrcPath.ToString != path {
				path = fmt.Sprintf("%s -> %s", c.SrcPath.ToString, path)
			}
			fmt.Printf("%s %s\n", status, path)
		}

		return nil
	},
}

var prActivityCmd = &cobra.Command{
	Use:   "activity [project/repo] <pr-id>",
	Short: "Show activity on a pull request",
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

		limit, _ := cmd.Flags().GetInt("limit")
		jsonOutput, _ := cmd.Flags().GetBool("json")

		activities, err := client.GetPullRequestActivity(ctx, project, repo, prID, limit)
		if err != nil {
			return fmt.Errorf("fetching activity: %w", err)
		}

		if jsonOutput {
			return json.NewEncoder(os.Stdout).Encode(activities)
		}

		if len(activities) == 0 {
			fmt.Println("No activity found")
			return nil
		}

		for _, a := range activities {
			date := time.Unix(a.CreatedDate/1000, 0).Format("2006-01-02 15:04")
			action := a.Action
			detail := ""
			if a.Comment != nil {
				detail = cmdutil.Truncate(a.Comment.Text, 60)
			}
			fmt.Printf("%s  %s  %s  %s\n", date, a.User.Name, action, detail)
		}

		return nil
	},
}

var prCanMergeCmd = &cobra.Command{
	Use:   "can-merge [project/repo] <pr-id>",
	Short: "Check if a pull request can be merged",
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

		jsonOutput, _ := cmd.Flags().GetBool("json")

		result, err := client.CanMerge(ctx, project, repo, prID)
		if err != nil {
			return fmt.Errorf("checking merge status: %w", err)
		}

		if jsonOutput {
			return json.NewEncoder(os.Stdout).Encode(result)
		}

		if result.CanMerge {
			fmt.Printf("✓ PR #%d can be merged\n", prID)
		} else {
			fmt.Printf("✗ PR #%d cannot be merged\n", prID)
			if result.Conflicted {
				fmt.Println("  Reason: conflicts detected")
			}
			for _, v := range result.Vetoes {
				fmt.Printf("  Veto: %s\n", v.SummaryMessage)
			}
		}

		return nil
	},
}

func changeTypeSymbol(t string) string {
	switch t {
	case "ADD":
		return "A"
	case "MODIFY":
		return "M"
	case "DELETE":
		return "D"
	case "MOVE":
		return "R"
	case "COPY":
		return "C"
	default:
		return "?"
	}
}

func firstLine(s string) string {
	for i, c := range s {
		if c == '\n' {
			return s[:i]
		}
	}
	return s
}

func init() {
	prCmd.AddCommand(prCommitsCmd)
	prCmd.AddCommand(prFilesCmd)
	prCmd.AddCommand(prActivityCmd)
	prCmd.AddCommand(prCanMergeCmd)

	prCommitsCmd.Flags().Int("limit", cmdutil.DefaultLimit, "Maximum number of commits")
	prCommitsCmd.Flags().Bool("json", false, "Output as JSON")

	prFilesCmd.Flags().Int("limit", 100, "Maximum number of files")
	prFilesCmd.Flags().Bool("json", false, "Output as JSON")

	prActivityCmd.Flags().Int("limit", cmdutil.DefaultLimit, "Maximum number of activities")
	prActivityCmd.Flags().Bool("json", false, "Output as JSON")

	prCanMergeCmd.Flags().Bool("json", false, "Output as JSON")
}
