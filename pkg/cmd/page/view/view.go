package view

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/lroolle/atlas-cli/api"
	"github.com/lroolle/atlas-cli/internal/cmdutil"
	"github.com/lroolle/atlas-cli/pkg/cmd/page/shared"
	"github.com/lroolle/atlas-cli/pkg/converter"
	"github.com/lroolle/atlas-cli/pkg/extractor"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

func NewCmdView() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "view [page-id]",
		Short: "View a Confluence page",
		Args:  cobra.ExactArgs(1),
		RunE:  runView,
	}

	cmd.Flags().StringP("output", "o", "", "Save output to file")
	cmd.Flags().String("format", "markdown", "Output format: markdown (md), storage (Confluence XHTML), or html")
	cmd.Flags().Bool("with-toc", false, "Add table of contents to output")
	cmd.Flags().Bool("with-images", false, "Download images and fix paths (requires -o)")
	cmd.Flags().Bool("info", false, "Show metadata summary only")

	return cmd
}

func runView(cmd *cobra.Command, args []string) error {
	ctx := cmd.Context()

	client, err := shared.GetConfluenceClient()
	if err != nil {
		return err
	}

	page, err := client.GetPage(ctx, args[0])
	if err != nil {
		return err
	}

	infoMode, err := cmd.Flags().GetBool("info")
	if err != nil {
		return fmt.Errorf("reading info flag: %w", err)
	}
	if infoMode {
		fmt.Printf("Page: %s\n", page.Title)
		fmt.Printf("Space: %s\n", page.Space.Key)
		fmt.Printf("Status: %s\n", page.Status)
		fmt.Printf("Version: %d\n", page.Version.Number)
		fmt.Printf("URL: %s%s\n", viper.GetString("confluence.server"), page.Links["webui"])
		return nil
	}

	outputFile, err := cmd.Flags().GetString("output")
	if err != nil {
		return fmt.Errorf("reading output flag: %w", err)
	}
	format, err := cmd.Flags().GetString("format")
	if err != nil {
		return fmt.Errorf("reading format flag: %w", err)
	}

	content, err := formatContent(page, format)
	if err != nil {
		return err
	}

	withTOC, err := cmd.Flags().GetBool("with-toc")
	if err != nil {
		return fmt.Errorf("reading with-toc flag: %w", err)
	}
	if withTOC && format == "markdown" {
		content = addTOC(page, content)
	}

	withImages, err := cmd.Flags().GetBool("with-images")
	if err != nil {
		return fmt.Errorf("reading with-images flag: %w", err)
	}
	if withImages {
		if outputFile == "" {
			return fmt.Errorf("--with-images requires -o to specify output file")
		}
		if format != "markdown" && format != "md" {
			return fmt.Errorf("--with-images only works with markdown format (current: %s)", format)
		}
		content, err = downloadAndFixImages(ctx, client, page, outputFile, content)
		if err != nil {
			return err
		}
	}

	if outputFile != "" {
		if err := os.WriteFile(outputFile, []byte(content), cmdutil.FilePermRW); err != nil {
			return fmt.Errorf("failed to write output file: %w", err)
		}
		fmt.Printf("Saved to %s\n", outputFile)
	} else {
		fmt.Println(content)
	}

	return nil
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

	baseName := strings.TrimSuffix(filepath.Base(outputFile), filepath.Ext(outputFile))
	outputDir := filepath.Dir(outputFile)
	imageDir := filepath.Join(outputDir, baseName+"_images")

	if err := os.MkdirAll(imageDir, cmdutil.DirPermStandard); err != nil {
		return "", fmt.Errorf("failed to create image directory: %w", err)
	}

	imageDirName := filepath.Base(imageDir)
	replacements := make(map[string]string)

	for _, img := range images {
		ext := strings.ToLower(filepath.Ext(img.Filename))
		if !cmdutil.IsValidImageExt(ext) {
			continue
		}

		imagePath := filepath.Join(imageDir, img.Filename)

		data, err := client.GetAttachment(ctx, page.ID, img.Filename)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Warning: failed to download %s: %v\n", img.Filename, err)
			continue
		}

		if err := os.WriteFile(imagePath, data, cmdutil.FilePermRW); err != nil {
			fmt.Fprintf(os.Stderr, "Warning: failed to save %s: %v\n", imagePath, err)
			continue
		}

		fmt.Printf("Downloaded: %s\n", imagePath)
		replacements[img.Filename] = filepath.Join(imageDirName, img.Filename)
	}

	for filename, relPath := range replacements {
		pattern := regexp.MustCompile(`(!\[[^\]]*\]\()([^)]*` + regexp.QuoteMeta(filename) + `[^)]*)(\))`)
		content = pattern.ReplaceAllString(content, `${1}`+relPath+`${3}`)
	}

	return content, nil
}
