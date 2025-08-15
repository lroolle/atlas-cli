package cmd

import (
	"fmt"
	"os"
	"text/tabwriter"

	"github.com/lroolle/atlas-cli/api"
	"github.com/spf13/cobra"
)

var prStatusCmd = &cobra.Command{
	Use:   "status",
	Short: "Show status of relevant pull requests",
	Long:  `Show status of pull requests relevant to you (created by you, requesting your review, etc.)`,
	RunE: func(cmd *cobra.Command, args []string) error {
		client := getClient()
		
		// Get default repo from config
		project, repo, err := parseRepoArg("")
		if err != nil {
			fmt.Printf("No default repository configured: %v\n", err)
			fmt.Println("Use 'atlas pr list PROJECT/REPO' instead.")
			return nil
		}
		
		fmt.Printf("Relevant pull requests in %s/%s\n\n", project, repo)
		
		// Get PRs created by me
		myPRs, err := client.ListPullRequests(project, repo, "OPEN", 20)
		if err != nil {
			return fmt.Errorf("fetching pull requests: %w", err)
		}
		
		var createdByMe []api.PullRequest
		var requestingReview []api.PullRequest
		
		for _, pr := range myPRs {
			if pr.Author.User.Name == client.Username {
				createdByMe = append(createdByMe, pr)
			} else {
				// Check if I'm a reviewer
				for _, reviewer := range pr.Reviewers {
					if reviewer.User.Name == client.Username {
						requestingReview = append(requestingReview, pr)
						break
					}
				}
			}
		}
		
		// Display created by me
		if len(createdByMe) > 0 {
			fmt.Println("Created by you")
			w := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
			for _, pr := range createdByMe {
				reviewStatus := getReviewStatus(pr)
				fmt.Fprintf(w, "  #%d\t%s\t%s\n", pr.ID, truncate(pr.Title, 50), reviewStatus)
			}
			w.Flush()
			fmt.Println()
		}
		
		// Display requesting review
		if len(requestingReview) > 0 {
			fmt.Println("Requesting your review")
			w := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
			for _, pr := range requestingReview {
				fmt.Fprintf(w, "  #%d\t%s\t%s\n", pr.ID, truncate(pr.Title, 50), pr.Author.User.Name)
			}
			w.Flush()
		}
		
		if len(createdByMe) == 0 && len(requestingReview) == 0 {
			fmt.Println("You have no relevant pull requests")
		}
		
		return nil
	},
}

func getReviewStatus(pr api.PullRequest) string {
	approved := 0
	needsWork := 0
	pending := 0
	
	for _, reviewer := range pr.Reviewers {
		if reviewer.Approved {
			approved++
		} else if reviewer.Status == "NEEDS_WORK" {
			needsWork++
		} else {
			pending++
		}
	}
	
	if needsWork > 0 {
		return fmt.Sprintf("❌ Changes requested")
	}
	if approved > 0 && pending == 0 {
		return fmt.Sprintf("✓ Approved by %d", approved)
	}
	if approved > 0 {
		return fmt.Sprintf("◐ %d approved, %d pending", approved, pending)
	}
	if pending > 0 {
		return fmt.Sprintf("○ %d pending review", pending)
	}
	return "No reviewers"
}

func init() {
	prCmd.AddCommand(prStatusCmd)
}