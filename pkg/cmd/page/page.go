package page

import (
	"github.com/lroolle/atlas-cli/pkg/cmd/page/children"
	"github.com/lroolle/atlas-cli/pkg/cmd/page/create"
	"github.com/lroolle/atlas-cli/pkg/cmd/page/delete"
	"github.com/lroolle/atlas-cli/pkg/cmd/page/edit"
	"github.com/lroolle/atlas-cli/pkg/cmd/page/list"
	"github.com/lroolle/atlas-cli/pkg/cmd/page/search"
	"github.com/lroolle/atlas-cli/pkg/cmd/page/spaces"
	"github.com/lroolle/atlas-cli/pkg/cmd/page/view"
	"github.com/spf13/cobra"
)

func NewCmdPage() *cobra.Command {
	cmd := &cobra.Command{
		Use:     "page",
		Short:   "Manage Confluence pages",
		Long:    `Commands for listing, viewing, and managing Confluence pages`,
		Aliases: []string{"confluence"},
	}

	cmd.AddCommand(list.NewCmdList())
	cmd.AddCommand(search.NewCmdSearch())
	cmd.AddCommand(view.NewCmdView())
	cmd.AddCommand(create.NewCmdCreate())
	cmd.AddCommand(edit.NewCmdEdit())
	cmd.AddCommand(delete.NewCmdDelete())
	cmd.AddCommand(children.NewCmdChildren())
	cmd.AddCommand(spaces.NewCmdSpaces())

	return cmd
}
