package search

import (
	"fmt"
	"os"
	"strings"
	"text/tabwriter"

	"github.com/lroolle/atlas-cli/internal/cmdutil"
	"github.com/lroolle/atlas-cli/pkg/cmd/page/shared"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

const (
	helpText = `Search for Confluence pages using text search or CQL.

The search query is a simple text search by default. Use --cql for raw CQL queries.`

	examples = `# Simple text search
$ atl page search "meeting notes"

# Search in specific space
$ atl page search "API documentation" --space PROJ

# Search by title
$ atl page search --title "Release Notes"

# Search pages created by user
$ atl page search --creator eric.wang

# Search pages modified recently
$ atl page search --modified today

# Raw CQL query
$ atl page search --cql 'type=page AND space=PROJ AND title~"release"'

# Combine filters
$ atl page search "backend" --space PROJ --type page --creator eric.wang`
)

func NewCmdSearch() *cobra.Command {
	cmd := &cobra.Command{
		Use:     "search [text]",
		Short:   "Search for Confluence pages",
		Long:    helpText,
		Example: examples,
		Aliases: []string{"find", "query"},
		Args:    cobra.MaximumNArgs(1),
		RunE:    runSearch,
	}

	cmd.Flags().Int("limit", cmdutil.DefaultLimit, "Maximum number of results")
	cmd.Flags().StringP("space", "s", "", "Limit search to specific space")
	cmd.Flags().StringP("type", "t", "page", "Content type: page, blogpost, comment, attachment")
	cmd.Flags().String("title", "", "Search by title (contains)")
	cmd.Flags().String("creator", "", "Filter by creator username")
	cmd.Flags().String("contributor", "", "Filter by contributor username")
	cmd.Flags().String("modified", "", "Modified date: today, yesterday, week, month, year")
	cmd.Flags().String("created", "", "Created date: today, yesterday, week, month, year")
	cmd.Flags().StringP("cql", "q", "", "Raw CQL query (overrides other filters)")
	cmd.Flags().String("order-by", "lastmodified", "Order by: created, lastmodified, title")
	cmd.Flags().Bool("reverse", false, "Reverse sort order (ascending)")

	return cmd
}

func runSearch(cmd *cobra.Command, args []string) error {
	ctx := cmd.Context()

	client, err := shared.GetConfluenceClient()
	if err != nil {
		return err
	}

	limit, _ := cmd.Flags().GetInt("limit")

	cql, err := buildCQL(cmd, args)
	if err != nil {
		return err
	}

	pages, err := client.SearchContent(ctx, cql, limit)
	if err != nil {
		return fmt.Errorf("search failed: %w\nCQL: %s", err, cql)
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
}

func buildCQL(cmd *cobra.Command, args []string) (string, error) {
	// Raw CQL takes precedence
	rawCQL, _ := cmd.Flags().GetString("cql")
	if rawCQL != "" {
		return rawCQL, nil
	}

	var conditions []string

	// Content type
	contentType, _ := cmd.Flags().GetString("type")
	if contentType != "" {
		conditions = append(conditions, fmt.Sprintf("type=%s", contentType))
	}

	// Space filter
	space, _ := cmd.Flags().GetString("space")
	if space == "" {
		space = viper.GetString("confluence.default_space")
	}
	if space != "" {
		conditions = append(conditions, fmt.Sprintf("space=%q", space))
	}

	// Text search (positional argument)
	if len(args) > 0 && args[0] != "" {
		text := args[0]
		conditions = append(conditions, fmt.Sprintf("text~%q", text))
	}

	// Title filter
	title, _ := cmd.Flags().GetString("title")
	if title != "" {
		conditions = append(conditions, fmt.Sprintf("title~%q", title))
	}

	// Creator filter
	creator, _ := cmd.Flags().GetString("creator")
	if creator != "" {
		conditions = append(conditions, fmt.Sprintf("creator=%q", creator))
	}

	// Contributor filter
	contributor, _ := cmd.Flags().GetString("contributor")
	if contributor != "" {
		conditions = append(conditions, fmt.Sprintf("contributor=%q", contributor))
	}

	// Modified date filter
	modified, _ := cmd.Flags().GetString("modified")
	if modified != "" {
		dateExpr := parseDateFilter(modified)
		if dateExpr != "" {
			conditions = append(conditions, fmt.Sprintf("lastmodified>=%s", dateExpr))
		}
	}

	// Created date filter
	created, _ := cmd.Flags().GetString("created")
	if created != "" {
		dateExpr := parseDateFilter(created)
		if dateExpr != "" {
			conditions = append(conditions, fmt.Sprintf("created>=%s", dateExpr))
		}
	}

	if len(conditions) == 0 {
		return "", fmt.Errorf("no search criteria provided (use text argument or filters)")
	}

	cql := strings.Join(conditions, " AND ")

	// Order by
	orderBy, _ := cmd.Flags().GetString("order-by")
	reverse, _ := cmd.Flags().GetBool("reverse")
	if orderBy != "" {
		direction := "desc"
		if reverse {
			direction = "asc"
		}
		cql = fmt.Sprintf("%s ORDER BY %s %s", cql, orderBy, direction)
	}

	return cql, nil
}

func parseDateFilter(value string) string {
	switch strings.ToLower(value) {
	case "today":
		return "startOfDay()"
	case "yesterday":
		return "startOfDay(-1d)"
	case "week":
		return "startOfWeek()"
	case "month":
		return "startOfMonth()"
	case "year":
		return "startOfYear()"
	default:
		// Assume it's a date like "2024-01-01"
		if strings.Contains(value, "-") || strings.Contains(value, "/") {
			return fmt.Sprintf("%q", value)
		}
		return ""
	}
}
