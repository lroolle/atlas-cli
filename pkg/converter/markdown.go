package converter

import (
	"strings"

	md "github.com/JohannesKaufmann/html-to-markdown/v2"
)

// HTMLToMarkdown converts HTML to markdown
func HTMLToMarkdown(html string) (string, error) {
	markdown, err := md.ConvertString(html)
	if err != nil {
		return "", err
	}
	return strings.TrimSpace(markdown), nil
}
