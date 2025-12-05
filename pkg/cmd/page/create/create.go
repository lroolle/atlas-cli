package create

import (
	"fmt"
	"os"
	"strings"

	"github.com/lroolle/atlas-cli/pkg/cmd/page/shared"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

func NewCmdCreate() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "create",
		Short: "Create a new Confluence page",
		Long:  `Create a new page in a Confluence space with specified title and content`,
		RunE:  runCreate,
	}

	cmd.Flags().StringP("space", "s", "", "Space key (uses default_space from config if not specified)")
	cmd.Flags().StringP("title", "t", "", "Page title (required)")
	cmd.Flags().StringP("content", "c", "", "Page content in Confluence storage format")
	cmd.Flags().StringP("content-file", "f", "", "File containing page content")
	cmd.Flags().StringP("parent", "p", "", "Parent page: ID, title, or URL")

	return cmd
}

func runCreate(cmd *cobra.Command, args []string) error {
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
		if spaceKey == "" {
			return fmt.Errorf("space required: use --space or set confluence.default_space in config")
		}
	}

	title, err := cmd.Flags().GetString("title")
	if err != nil {
		return fmt.Errorf("reading title flag: %w", err)
	}
	title = strings.TrimSpace(title)
	if title == "" {
		return fmt.Errorf("--title is required and cannot be empty")
	}

	content, err := cmd.Flags().GetString("content")
	if err != nil {
		return fmt.Errorf("reading content flag: %w", err)
	}
	contentFile, err := cmd.Flags().GetString("content-file")
	if err != nil {
		return fmt.Errorf("reading content-file flag: %w", err)
	}

	if content == "" && contentFile == "" {
		return fmt.Errorf("either --content or --content-file is required")
	}

	if contentFile != "" {
		data, err := os.ReadFile(contentFile)
		if err != nil {
			return fmt.Errorf("failed to read content file: %w", err)
		}
		content = string(data)
	}

	parent, err := cmd.Flags().GetString("parent")
	if err != nil {
		return fmt.Errorf("reading parent flag: %w", err)
	}
	parentID, err := shared.ResolvePage(ctx, client, parent, spaceKey)
	if err != nil {
		return err
	}

	page, err := client.CreatePage(ctx, spaceKey, title, content, parentID)
	if err != nil {
		return err
	}

	fmt.Printf("Page created successfully\n")
	fmt.Printf("ID: %s\n", page.ID)
	fmt.Printf("Title: %s\n", page.Title)
	fmt.Printf("Space: %s\n", page.Space.Key)
	if webUI, ok := page.Links["webui"]; ok {
		fmt.Printf("URL: %s%s\n", viper.GetString("confluence.server"), webUI)
	}

	return nil
}
