package cmd

import (
	"fmt"
	"os"
	"strings"
	"text/tabwriter"

	"github.com/lroolle/atlas-cli/api"
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
		
		fmt.Printf("Page: %s\n", page.Title)
		fmt.Printf("Space: %s\n", page.Space.Key)
		fmt.Printf("Status: %s\n", page.Status)
		fmt.Printf("Version: %d\n", page.Version.Number)
		fmt.Printf("URL: %s%s\n", viper.GetString("confluence.server"), page.Links["webui"])
		
		showContent, _ := cmd.Flags().GetBool("content")
		if showContent && page.Body.View.Value != "" {
			fmt.Println("\nContent (HTML):")
			fmt.Println(page.Body.View.Value)
		}
		
		return nil
	},
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
	
	pageViewCmd.Flags().Bool("content", false, "Show page content")
	
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