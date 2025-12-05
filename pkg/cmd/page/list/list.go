package list

import (
	"fmt"
	"os"
	"text/tabwriter"

	"github.com/lroolle/atlas-cli/internal/cmdutil"
	"github.com/lroolle/atlas-cli/pkg/cmd/page/shared"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

func NewCmdList() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "list [space]",
		Short: "List pages in a Confluence space",
		Args:  cobra.MaximumNArgs(1),
		RunE:  runList,
	}

	cmd.Flags().Int("limit", cmdutil.DefaultLimit, "Maximum number of results")
	cmd.Flags().String("type", "page", "Content type (page, blogpost)")

	return cmd
}

func runList(cmd *cobra.Command, args []string) error {
	ctx := cmd.Context()

	var spaceKey string
	if len(args) > 0 {
		spaceKey = args[0]
	} else {
		spaceKey = viper.GetString("confluence.default_space")
		if spaceKey == "" {
			return fmt.Errorf("space required: provide space key or set confluence.default_space in config")
		}
	}

	client, err := shared.GetConfluenceClient()
	if err != nil {
		return err
	}

	limit, err := cmd.Flags().GetInt("limit")
	if err != nil {
		return fmt.Errorf("reading limit flag: %w", err)
	}
	contentType, err := cmd.Flags().GetString("type")
	if err != nil {
		return fmt.Errorf("reading type flag: %w", err)
	}

	pages, err := client.GetContent(ctx, spaceKey, contentType, limit)
	if err != nil {
		return err
	}

	if len(pages) == 0 {
		fmt.Printf("No %s found in space %s\n", contentType, spaceKey)
		return nil
	}

	w := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
	fmt.Fprintln(w, "ID\tTITLE\tSTATUS\tVERSION")

	for _, page := range pages {
		fmt.Fprintf(w, "%s\t%s\t%s\tv%d\n",
			page.ID,
			cmdutil.Truncate(page.Title, cmdutil.TitleTruncateNormal),
			page.Status,
			page.Version.Number,
		)
	}

	return w.Flush()
}
