package delete

import (
	"bufio"
	"context"
	"fmt"
	"os"
	"strings"

	"github.com/lroolle/atlas-cli/api"
	"github.com/lroolle/atlas-cli/pkg/cmd/page/shared"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
	"golang.org/x/term"
)

func NewCmdDelete() *cobra.Command {
	cmd := &cobra.Command{
		Use:     "delete {<id> | <title> | <url>}",
		Short:   "Delete a Confluence page",
		Aliases: []string{"rm", "del", "remove"},
		Args:    cobra.ExactArgs(1),
		RunE:    runDelete,
	}

	cmd.Flags().BoolP("yes", "y", false, "Skip confirmation prompt")
	cmd.Flags().StringP("space", "s", "", "Space key (required when deleting by title)")
	cmd.Flags().Bool("cascade", false, "Delete page and all child pages recursively")

	return cmd
}

func runDelete(cmd *cobra.Command, args []string) error {
	ctx := cmd.Context()

	client, err := shared.GetConfluenceClient()
	if err != nil {
		return err
	}

	spaceKey, err := cmd.Flags().GetString("space")
	if err != nil {
		return fmt.Errorf("reading space flag: %w", err)
	}
	if spaceKey == "" {
		spaceKey = viper.GetString("confluence.default_space")
	}

	pageID, err := shared.ResolvePage(ctx, client, args[0], spaceKey)
	if err != nil {
		return err
	}

	page, err := client.GetPage(ctx, pageID)
	if err != nil {
		return fmt.Errorf("failed to fetch page: %w", err)
	}

	cascade, err := cmd.Flags().GetBool("cascade")
	if err != nil {
		return fmt.Errorf("reading cascade flag: %w", err)
	}

	children, err := client.GetChildPages(ctx, pageID, 0)
	if err == nil && len(children) > 0 {
		if !cascade {
			return fmt.Errorf("page %q has %d child page(s) - use --cascade to delete recursively, or delete children first", page.Title, len(children))
		}
		fmt.Fprintf(os.Stderr, "Will delete %d child page(s) recursively\n", len(children))
	}

	yes, err := cmd.Flags().GetBool("yes")
	if err != nil {
		return fmt.Errorf("reading yes flag: %w", err)
	}
	if !yes && isInteractive() {
		fmt.Fprintf(os.Stderr, "Deleted pages cannot be recovered.\n")
		fmt.Fprintf(os.Stderr, "Type %q to confirm deletion: ", page.Title)

		reader := bufio.NewReader(os.Stdin)
		input, err := reader.ReadString('\n')
		if err != nil {
			return fmt.Errorf("failed to read input: %w", err)
		}

		if strings.TrimSpace(input) != page.Title {
			return fmt.Errorf("deletion cancelled")
		}
	}

	if cascade && len(children) > 0 {
		if err := deleteRecursive(ctx, client, children); err != nil {
			return fmt.Errorf("failed to delete child pages: %w", err)
		}
	}

	if err := client.DeletePage(ctx, pageID); err != nil {
		return err
	}

	fmt.Printf("Deleted page %q (%s)\n", page.Title, pageID)
	return nil
}

func deleteRecursive(ctx context.Context, client *api.ConfluenceClient, pages []api.Content) error {
	for _, page := range pages {
		children, err := client.GetChildPages(ctx, page.ID, 0)
		if err == nil && len(children) > 0 {
			if err := deleteRecursive(ctx, client, children); err != nil {
				return err
			}
		}
		if err := client.DeletePage(ctx, page.ID); err != nil {
			return fmt.Errorf("failed to delete %q: %w", page.Title, err)
		}
		fmt.Printf("Deleted child page %q (%s)\n", page.Title, page.ID)
	}
	return nil
}

func isInteractive() bool {
	return term.IsTerminal(int(os.Stdin.Fd()))
}
