package spaces

import (
	"fmt"
	"os"
	"text/tabwriter"

	"github.com/lroolle/atlas-cli/internal/cmdutil"
	"github.com/lroolle/atlas-cli/pkg/cmd/page/shared"
	"github.com/spf13/cobra"
)

func NewCmdSpaces() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "spaces",
		Short: "List Confluence spaces",
		RunE:  runSpaces,
	}

	cmd.Flags().Int("limit", cmdutil.DefaultLimit, "Maximum number of results")

	return cmd
}

func runSpaces(cmd *cobra.Command, args []string) error {
	ctx := cmd.Context()

	client, err := shared.GetConfluenceClient()
	if err != nil {
		return err
	}

	limit, _ := cmd.Flags().GetInt("limit")

	spaces, err := client.GetSpaces(ctx, limit)
	if err != nil {
		return err
	}

	if len(spaces) == 0 {
		fmt.Println("No spaces found")
		return nil
	}

	w := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
	fmt.Fprintln(w, "KEY\tNAME\tTYPE\tSTATUS")

	for _, space := range spaces {
		fmt.Fprintf(w, "%s\t%s\t%s\t%s\n",
			space.Key,
			cmdutil.Truncate(space.Name, cmdutil.TitleTruncateShort),
			space.Type,
			space.Status,
		)
	}

	return w.Flush()
}
