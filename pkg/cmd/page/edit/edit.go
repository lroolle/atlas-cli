package edit

import (
	"fmt"
	"os"

	"github.com/lroolle/atlas-cli/pkg/cmd/page/shared"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

func NewCmdEdit() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "edit [page-id]",
		Short: "Edit an existing Confluence page",
		Long:  `Update an existing page's title and/or content`,
		Args:  cobra.ExactArgs(1),
		RunE:  runEdit,
	}

	cmd.Flags().StringP("title", "t", "", "New page title (keeps current if not specified)")
	cmd.Flags().StringP("content", "c", "", "New page content in Confluence storage format")
	cmd.Flags().StringP("content-file", "f", "", "File containing new page content")

	return cmd
}

func runEdit(cmd *cobra.Command, args []string) error {
	ctx := cmd.Context()

	client, err := shared.GetConfluenceClient()
	if err != nil {
		return err
	}

	pageID := args[0]

	currentPage, err := client.GetPage(ctx, pageID)
	if err != nil {
		return fmt.Errorf("failed to fetch page: %w", err)
	}

	title, _ := cmd.Flags().GetString("title")
	if title == "" {
		title = currentPage.Title
	}

	content, _ := cmd.Flags().GetString("content")
	contentFile, _ := cmd.Flags().GetString("content-file")

	if contentFile != "" {
		data, err := os.ReadFile(contentFile)
		if err != nil {
			return fmt.Errorf("failed to read content file: %w", err)
		}
		content = string(data)
	}

	if content == "" {
		content = currentPage.Body.Storage.Value
	}

	page, err := client.UpdatePage(ctx, pageID, title, content, currentPage.Version.Number)
	if err != nil {
		return err
	}

	fmt.Printf("Page updated successfully\n")
	fmt.Printf("ID: %s\n", page.ID)
	fmt.Printf("Title: %s\n", page.Title)
	fmt.Printf("Version: %d\n", page.Version.Number)
	if webUI, ok := page.Links["webui"]; ok {
		fmt.Printf("URL: %s%s\n", viper.GetString("confluence.server"), webUI)
	}

	return nil
}
