package children

import (
	"fmt"
	"os"
	"text/tabwriter"

	"github.com/lroolle/atlas-cli/internal/cmdutil"
	"github.com/lroolle/atlas-cli/pkg/cmd/page/shared"
	"github.com/spf13/cobra"
)

func NewCmdChildren() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "children [page-id]",
		Short: "List child pages of a parent page",
		Args:  cobra.ExactArgs(1),
		RunE:  runChildren,
	}

	cmd.Flags().Int("limit", cmdutil.DefaultChildrenLimit, "Maximum number of child pages to retrieve")

	return cmd
}

func runChildren(cmd *cobra.Command, args []string) error {
	ctx := cmd.Context()

	client, err := shared.GetConfluenceClient()
	if err != nil {
		return err
	}

	pageID := args[0]
	limit, _ := cmd.Flags().GetInt("limit")

	children, err := client.GetChildPages(ctx, pageID, limit)
	if err != nil {
		return err
	}

	if len(children) == 0 {
		fmt.Printf("No child pages found for page %s\n", pageID)
		return nil
	}

	w := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
	fmt.Fprintln(w, "ID\tTITLE\tSTATUS\tVERSION")

	for _, page := range children {
		fmt.Fprintf(w, "%s\t%s\t%s\tv%d\n",
			page.ID,
			cmdutil.Truncate(page.Title, cmdutil.TitleTruncateLong),
			page.Status,
			page.Version.Number,
		)
	}

	return w.Flush()
}
