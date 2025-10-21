package cmd

import (
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"text/tabwriter"

	"github.com/lroolle/atlas-cli/api"
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

		limit, _ := cmd.Flags().GetInt("limit")
		contentType, _ := cmd.Flags().GetString("type")

		pages, err := client.GetContent(spaceKey, contentType, limit)
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
				truncate(page.Title, 50),
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
		query := strings.Join(args, " ")

		client, err := getConfluenceClient()
		if err != nil {
			return err
		}

		limit, _ := cmd.Flags().GetInt("limit")
		space, _ := cmd.Flags().GetString("space")

		// Build CQL query
		cql := query
		if space != "" {
			cql = fmt.Sprintf("space = %s AND (%s)", space, query)
		}

		pages, err := client.SearchContent(cql, limit)
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
				truncate(page.Title, 50),
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
		client, err := getConfluenceClient()
		if err != nil {
			return err
		}

		page, err := client.GetPage(args[0])
		if err != nil {
			return err
		}

		// Info mode - show metadata only
		infoMode, _ := cmd.Flags().GetBool("info")
		if infoMode {
			fmt.Printf("Page: %s\n", page.Title)
			fmt.Printf("Space: %s\n", page.Space.Key)
			fmt.Printf("Status: %s\n", page.Status)
			fmt.Printf("Version: %d\n", page.Version.Number)
			fmt.Printf("URL: %s%s\n", viper.GetString("confluence.server"), page.Links["webui"])
			return nil
		}

		outputFile, _ := cmd.Flags().GetString("output")
		format, _ := cmd.Flags().GetString("format")

		content, err := formatContent(page, format)
		if err != nil {
			return err
		}

		withTOC, _ := cmd.Flags().GetBool("with-toc")
		if withTOC && format == "markdown" {
			content = addTOC(page, content)
		}

		withImages, _ := cmd.Flags().GetBool("with-images")
		if withImages {
			if outputFile == "" {
				return fmt.Errorf("--with-images requires -o to specify output file")
			}
			content, err = downloadAndFixImages(client, page, outputFile, content)
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

func downloadAndFixImages(client *api.ConfluenceClient, page *api.Content, outputFile string, content string) (string, error) {
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
	imageDir := outputDir + "_images"

	if err := os.MkdirAll(imageDir, 0755); err != nil {
		return "", fmt.Errorf("failed to create image directory: %w", err)
	}

	for _, img := range images {
		if !strings.Contains(img.Filename, ".png") && !strings.Contains(img.Filename, ".jpg") && !strings.Contains(img.Filename, ".jpeg") && !strings.Contains(img.Filename, ".gif") {
			continue
		}

		imagePath := fmt.Sprintf("%s/%s", imageDir, img.Filename)

		data, err := client.GetAttachment(page.ID, img.Filename)
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

	// Fix all image references in content
	// Replace patterns in markdown image syntax: ![...](path/to/image.png?params)
	imageDirName := filepath.Base(imageDir)
	for _, img := range images {
		relPath := fmt.Sprintf("%s/%s", imageDirName, img.Filename)

		pattern := regexp.MustCompile(`(!\[[^\]]*\]\()([^)]*` + regexp.QuoteMeta(img.Filename) + `[^)]*)(\))`)
		content = pattern.ReplaceAllString(content, `${1}`+relPath+`${3}`)
	}

	return content, nil
}

var spaceListCmd = &cobra.Command{
	Use:   "spaces",
	Short: "List Confluence spaces",
	RunE: func(cmd *cobra.Command, args []string) error {
		client, err := getConfluenceClient()
		if err != nil {
			return err
		}

		limit, _ := cmd.Flags().GetInt("limit")

		spaces, err := client.GetSpaces(limit)
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
				truncate(space.Name, 40),
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
	pageCmd.AddCommand(spaceListCmd)

	pageListCmd.Flags().Int("limit", 25, "Maximum number of results")
	pageListCmd.Flags().String("type", "page", "Content type (page, blogpost)")

	pageSearchCmd.Flags().Int("limit", 25, "Maximum number of results")
	pageSearchCmd.Flags().String("space", "", "Limit search to specific space")

	pageViewCmd.Flags().StringP("output", "o", "", "Save output to file")
	pageViewCmd.Flags().String("format", "markdown", "Output format: markdown (md), storage (Confluence XHTML), or html")
	pageViewCmd.Flags().Bool("with-toc", false, "Add table of contents to output")
	pageViewCmd.Flags().Bool("with-images", false, "Download images and fix paths (requires -o)")
	pageViewCmd.Flags().Bool("info", false, "Show metadata summary only")

	spaceListCmd.Flags().Int("limit", 25, "Maximum number of results")
}

func getConfluenceClient() (*api.ConfluenceClient, error) {
	client, err := api.GetConfluenceClient()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		fmt.Fprintln(os.Stderr, "Set in config file under 'confluence' section")
		os.Exit(1)
	}
	return client, err
}
