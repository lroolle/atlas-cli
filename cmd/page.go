package cmd

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"text/tabwriter"

	"github.com/lroolle/atlas-cli/api"
	"github.com/lroolle/atlas-cli/internal/cmdutil"
	"github.com/lroolle/atlas-cli/pkg/converter"
	"github.com/lroolle/atlas-cli/pkg/extractor"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

var pageCmd = &cobra.Command{
	Use:     "page",
	Short:   "Manage Confluence pages",
	Long:    `Commands for listing, viewing, and managing Confluence pages`,
	Aliases: []string{"confluence"},
}

var pageListCmd = &cobra.Command{
	Use:   "list [space]",
	Short: "List pages in a Confluence space",
	Args:  cobra.MaximumNArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
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

		client, err := getConfluenceClient()
		if err != nil {
			return err
		}

		limit, err := cmd.Flags().GetInt("limit")
		cmdutil.ExitIfError(err)
		contentType, err := cmd.Flags().GetString("type")
		cmdutil.ExitIfError(err)

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
	},
}

var pageSearchCmd = &cobra.Command{
	Use:   "search [query]",
	Short: "Search for Confluence pages",
	Args:  cobra.MinimumNArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		ctx := cmd.Context()
		query := strings.Join(args, " ")

		client, err := getConfluenceClient()
		if err != nil {
			return err
		}

		limit, err := cmd.Flags().GetInt("limit")
		cmdutil.ExitIfError(err)
		space, err := cmd.Flags().GetString("space")
		cmdutil.ExitIfError(err)

		cql := query
		if space != "" {
			cql = fmt.Sprintf("space = %s AND (%s)", space, query)
		}

		pages, err := client.SearchContent(ctx, cql, limit)
		if err != nil {
			return err
		}

		if len(pages) == 0 {
			fmt.Println("No pages found")
			return nil
		}

		w := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
		fmt.Fprintln(w, "ID\tSPACE\tTITLE\tVERSION")

		for _, page := range pages {
			fmt.Fprintf(w, "%s\t%s\t%s\tv%d\n",
				page.ID,
				page.Space.Key,
				cmdutil.Truncate(page.Title, cmdutil.TitleTruncateNormal),
				page.Version.Number,
			)
		}

		return w.Flush()
	},
}

var pageViewCmd = &cobra.Command{
	Use:   "view [page-id]",
	Short: "View a Confluence page",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		ctx := cmd.Context()
		client, err := getConfluenceClient()
		if err != nil {
			return err
		}

		page, err := client.GetPage(ctx, args[0])
		if err != nil {
			return err
		}

		infoMode, err := cmd.Flags().GetBool("info")
		cmdutil.ExitIfError(err)
		if infoMode {
			fmt.Printf("Page: %s\n", page.Title)
			fmt.Printf("Space: %s\n", page.Space.Key)
			fmt.Printf("Status: %s\n", page.Status)
			fmt.Printf("Version: %d\n", page.Version.Number)
			fmt.Printf("URL: %s%s\n", viper.GetString("confluence.server"), page.Links["webui"])
			return nil
		}

		outputFile, err := cmd.Flags().GetString("output")
		cmdutil.ExitIfError(err)
		format, err := cmd.Flags().GetString("format")
		cmdutil.ExitIfError(err)

		content, err := formatContent(page, format)
		if err != nil {
			return err
		}

		withTOC, err := cmd.Flags().GetBool("with-toc")
		cmdutil.ExitIfError(err)
		if withTOC && format == "markdown" {
			content = addTOC(page, content)
		}

		withImages, err := cmd.Flags().GetBool("with-images")
		cmdutil.ExitIfError(err)
		if withImages {
			if outputFile == "" {
				return fmt.Errorf("--with-images requires -o to specify output file")
			}
			content, err = downloadAndFixImages(ctx, client, page, outputFile, content)
			if err != nil {
				return err
			}
		}

		if outputFile != "" {
			if err := os.WriteFile(outputFile, []byte(content), 0644); err != nil {
				return fmt.Errorf("failed to write output file: %w", err)
			}
			fmt.Printf("Saved to %s\n", outputFile)
		} else {
			fmt.Println(content)
		}

		return nil
	},
}

func formatContent(page *api.Content, format string) (string, error) {
	switch format {
	case "html":
		return page.Body.View.Value, nil
	case "storage":
		if page.Body.Storage.Value != "" {
			return page.Body.Storage.Value, nil
		}
		return page.Body.View.Value, nil
	case "markdown", "md":
		markdown, err := converter.HTMLToMarkdown(page.Body.View.Value)
		if err != nil {
			return "", fmt.Errorf("failed to convert to markdown: %w", err)
		}
		return markdown, nil
	default:
		return "", fmt.Errorf("unsupported format: %s (supported: html, markdown, md, storage)", format)
	}
}

func addTOC(page *api.Content, content string) string {
	htmlContent := page.Body.Storage.Value
	if htmlContent == "" {
		htmlContent = page.Body.View.Value
	}

	headings := extractor.ExtractHeadings(htmlContent)
	if len(headings) == 0 {
		return content
	}

	var toc strings.Builder
	toc.WriteString("## Table of Contents\n\n")
	for _, h := range headings {
		indent := strings.Repeat("  ", h.Level-1)
		toc.WriteString(fmt.Sprintf("%s- %s\n", indent, h.Text))
	}
	toc.WriteString("\n---\n\n")
	toc.WriteString(content)

	return toc.String()
}

func downloadAndFixImages(ctx context.Context, client *api.ConfluenceClient, page *api.Content, outputFile string, content string) (string, error) {
	htmlContent := page.Body.Storage.Value
	if htmlContent == "" {
		htmlContent = page.Body.View.Value
	}

	images := extractor.ExtractImages(htmlContent)
	if len(images) == 0 {
		return content, nil
	}

	outputDir := strings.TrimSuffix(outputFile, ".md")
	if outputDir == outputFile {
		outputDir = "."
	}
	imageDir := filepath.Join(outputDir, filepath.Base(outputDir)+"_images")

	if err := os.MkdirAll(imageDir, 0755); err != nil {
		return "", fmt.Errorf("failed to create image directory: %w", err)
	}

	for _, img := range images {
		if !strings.Contains(img.Filename, ".png") && !strings.Contains(img.Filename, ".jpg") && !strings.Contains(img.Filename, ".jpeg") && !strings.Contains(img.Filename, ".gif") {
			continue
		}

		imagePath := filepath.Join(imageDir, img.Filename)

		data, err := client.GetAttachment(ctx, page.ID, img.Filename)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Warning: failed to download %s: %v\n", img.Filename, err)
			continue
		}

		if err := os.WriteFile(imagePath, data, 0644); err != nil {
			fmt.Fprintf(os.Stderr, "Warning: failed to save %s: %v\n", imagePath, err)
			continue
		}

		fmt.Printf("Downloaded: %s\n", imagePath)
	}

	imageDirName := filepath.Base(imageDir)
	for _, img := range images {
		relPath := filepath.Join(imageDirName, img.Filename)

		pattern := regexp.MustCompile(`(!\[[^\]]*\]\()([^)]*` + regexp.QuoteMeta(img.Filename) + `[^)]*)(\))`)
		content = pattern.ReplaceAllString(content, `${1}`+relPath+`${3}`)
	}

	return content, nil
}

var pageCreateCmd = &cobra.Command{
	Use:   "create",
	Short: "Create a new Confluence page",
	Long:  `Create a new page in a Confluence space with specified title and content`,
	RunE: func(cmd *cobra.Command, args []string) error {
		ctx := cmd.Context()
		client, err := getConfluenceClient()
		if err != nil {
			return err
		}

		spaceKey, err := cmd.Flags().GetString("space")
		cmdutil.ExitIfError(err)
		if spaceKey == "" {
			spaceKey = viper.GetString("confluence.default_space")
			if spaceKey == "" {
				return fmt.Errorf("space required: use --space or set confluence.default_space in config")
			}
		}

		title, err := cmd.Flags().GetString("title")
		cmdutil.ExitIfError(err)
		if title == "" {
			return fmt.Errorf("--title is required")
		}

		content, err := cmd.Flags().GetString("content")
		cmdutil.ExitIfError(err)
		contentFile, err := cmd.Flags().GetString("content-file")
		cmdutil.ExitIfError(err)

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

		parentID, err := cmd.Flags().GetString("parent")
		cmdutil.ExitIfError(err)

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
	},
}

var pageEditCmd = &cobra.Command{
	Use:   "edit [page-id]",
	Short: "Edit an existing Confluence page",
	Long:  `Update an existing page's title and/or content`,
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		ctx := cmd.Context()
		client, err := getConfluenceClient()
		if err != nil {
			return err
		}

		pageID := args[0]

		currentPage, err := client.GetPage(ctx, pageID)
		if err != nil {
			return fmt.Errorf("failed to fetch page: %w", err)
		}

		title, err := cmd.Flags().GetString("title")
		cmdutil.ExitIfError(err)
		if title == "" {
			title = currentPage.Title
		}

		content, err := cmd.Flags().GetString("content")
		cmdutil.ExitIfError(err)
		contentFile, err := cmd.Flags().GetString("content-file")
		cmdutil.ExitIfError(err)

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
	},
}

var pageChildrenCmd = &cobra.Command{
	Use:   "children [page-id]",
	Short: "List child pages of a parent page",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		ctx := cmd.Context()
		client, err := getConfluenceClient()
		if err != nil {
			return err
		}

		pageID := args[0]
		limit, err := cmd.Flags().GetInt("limit")
		cmdutil.ExitIfError(err)

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
	},
}

var spaceListCmd = &cobra.Command{
	Use:   "spaces",
	Short: "List Confluence spaces",
	RunE: func(cmd *cobra.Command, args []string) error {
		ctx := cmd.Context()
		client, err := getConfluenceClient()
		if err != nil {
			return err
		}

		limit, err := cmd.Flags().GetInt("limit")
		cmdutil.ExitIfError(err)

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
	},
}

func init() {
	rootCmd.AddCommand(pageCmd)
	pageCmd.AddCommand(pageListCmd)
	pageCmd.AddCommand(pageSearchCmd)
	pageCmd.AddCommand(pageViewCmd)
	pageCmd.AddCommand(pageCreateCmd)
	pageCmd.AddCommand(pageEditCmd)
	pageCmd.AddCommand(pageChildrenCmd)
	pageCmd.AddCommand(spaceListCmd)

	pageListCmd.Flags().Int("limit", cmdutil.DefaultLimit, "Maximum number of results")
	pageListCmd.Flags().String("type", "page", "Content type (page, blogpost)")

	pageSearchCmd.Flags().Int("limit", cmdutil.DefaultLimit, "Maximum number of results")
	pageSearchCmd.Flags().String("space", "", "Limit search to specific space")

	pageViewCmd.Flags().StringP("output", "o", "", "Save output to file")
	pageViewCmd.Flags().String("format", "markdown", "Output format: markdown (md), storage (Confluence XHTML), or html")
	pageViewCmd.Flags().Bool("with-toc", false, "Add table of contents to output")
	pageViewCmd.Flags().Bool("with-images", false, "Download images and fix paths (requires -o)")
	pageViewCmd.Flags().Bool("info", false, "Show metadata summary only")

	pageCreateCmd.Flags().StringP("space", "s", "", "Space key (uses default_space from config if not specified)")
	pageCreateCmd.Flags().StringP("title", "t", "", "Page title (required)")
	pageCreateCmd.Flags().StringP("content", "c", "", "Page content in Confluence storage format")
	pageCreateCmd.Flags().StringP("content-file", "f", "", "File containing page content")
	pageCreateCmd.Flags().StringP("parent", "p", "", "Parent page ID (optional)")

	pageEditCmd.Flags().StringP("title", "t", "", "New page title (keeps current if not specified)")
	pageEditCmd.Flags().StringP("content", "c", "", "New page content in Confluence storage format")
	pageEditCmd.Flags().StringP("content-file", "f", "", "File containing new page content")

	pageChildrenCmd.Flags().Int("limit", cmdutil.DefaultChildrenLimit, "Maximum number of child pages to retrieve")

	spaceListCmd.Flags().Int("limit", cmdutil.DefaultLimit, "Maximum number of results")
}

func getConfluenceClient() (*api.ConfluenceClient, error) {
	client, err := api.GetConfluenceClient()
	if err != nil {
		return nil, fmt.Errorf("%w\nHint: Set credentials in config file under 'confluence' section", err)
	}
	return client, nil
}
