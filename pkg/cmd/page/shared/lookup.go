package shared

import (
	"context"
	"fmt"
	"net/url"
	"strconv"
	"strings"

	"github.com/lroolle/atlas-cli/api"
)

// ResolvePage resolves a page reference to a page ID.
// Accepts: numeric ID, page title (requires spaceKey), or Confluence URL.
func ResolvePage(ctx context.Context, client *api.ConfluenceClient, ref, spaceKey string) (string, error) {
	if ref == "" {
		return "", nil
	}

	if _, err := strconv.Atoi(ref); err == nil {
		return ref, nil
	}

	if strings.HasPrefix(ref, "http://") || strings.HasPrefix(ref, "https://") {
		return ParseConfluenceURL(ctx, client, ref)
	}

	if spaceKey == "" {
		return "", fmt.Errorf("cannot resolve page by title without space (use --space or provide page ID/URL)")
	}

	page, err := client.GetPageByTitle(ctx, spaceKey, ref)
	if err != nil {
		return "", fmt.Errorf("failed to resolve page %q: %w", ref, err)
	}
	return page.ID, nil
}

// ParseConfluenceURL extracts page ID from Confluence URL patterns:
// - /display/SPACE/Page+Title
// - /pages/viewpage.action?pageId=123
func ParseConfluenceURL(ctx context.Context, client *api.ConfluenceClient, rawURL string) (string, error) {
	u, err := url.Parse(rawURL)
	if err != nil {
		return "", fmt.Errorf("invalid URL: %w", err)
	}

	if strings.Contains(u.Path, "viewpage.action") {
		pageID := u.Query().Get("pageId")
		if pageID != "" {
			return pageID, nil
		}
	}

	if strings.HasPrefix(u.Path, "/display/") {
		parts := strings.SplitN(strings.TrimPrefix(u.Path, "/display/"), "/", 2)
		if len(parts) == 2 {
			spaceKey := parts[0]
			title, err := url.PathUnescape(parts[1])
			if err != nil {
				title = parts[1]
			}
			title = strings.ReplaceAll(title, "+", " ")

			page, err := client.GetPageByTitle(ctx, spaceKey, title)
			if err != nil {
				return "", fmt.Errorf("failed to resolve page from URL: %w", err)
			}
			return page.ID, nil
		}
	}

	return "", fmt.Errorf("cannot parse Confluence URL: %s (expected /display/SPACE/Title or /pages/viewpage.action?pageId=ID)", rawURL)
}
