package extractor

import (
	"regexp"
	"strings"

	"golang.org/x/net/html"
)

type Metadata struct {
	Links    []Link
	Images   []Image
	Headings []Heading
}

type Link struct {
	Text string
	URL  string
}

type Image struct {
	Alt      string
	Filename string
}

type Heading struct {
	Level int
	Text  string
}

// ExtractLinks extracts all links from HTML or storage format
func ExtractLinks(content string) []Link {
	var links []Link

	// Parse HTML
	doc, err := html.Parse(strings.NewReader(content))
	if err != nil {
		return links
	}

	var extract func(*html.Node)
	extract = func(n *html.Node) {
		if n.Type == html.ElementNode && n.Data == "a" {
			var href, text string
			for _, attr := range n.Attr {
				if attr.Key == "href" {
					href = attr.Val
				}
			}
			if n.FirstChild != nil && n.FirstChild.Type == html.TextNode {
				text = n.FirstChild.Data
			}
			if href != "" {
				links = append(links, Link{Text: text, URL: href})
			}
		}
		for c := n.FirstChild; c != nil; c = c.NextSibling {
			extract(c)
		}
	}
	extract(doc)

	// Also extract Confluence-specific links from storage format
	// <ac:link><ri:page ri:content-title="Page Title" /></ac:link>
	pageLinkRe := regexp.MustCompile(`<ri:page ri:content-title="([^"]+)"`)
	matches := pageLinkRe.FindAllStringSubmatch(content, -1)
	for _, match := range matches {
		if len(match) > 1 {
			links = append(links, Link{Text: match[1], URL: "confluence://" + match[1]})
		}
	}

	return links
}

// ExtractImages extracts all images from HTML or storage format
func ExtractImages(content string) []Image {
	var images []Image

	// Parse HTML for regular img tags
	doc, err := html.Parse(strings.NewReader(content))
	if err != nil {
		return images
	}

	var extract func(*html.Node)
	extract = func(n *html.Node) {
		if n.Type == html.ElementNode && n.Data == "img" {
			var src, alt string
			for _, attr := range n.Attr {
				if attr.Key == "src" {
					src = attr.Val
				}
				if attr.Key == "alt" {
					alt = attr.Val
				}
			}
			if src != "" {
				images = append(images, Image{Alt: alt, Filename: src})
			}
		}
		for c := n.FirstChild; c != nil; c = c.NextSibling {
			extract(c)
		}
	}
	extract(doc)

	// Extract Confluence attachments
	// <ri:attachment ri:filename="image.png" />
	attachmentRe := regexp.MustCompile(`<ri:attachment ri:filename="([^"]+)"`)
	matches := attachmentRe.FindAllStringSubmatch(content, -1)
	for _, match := range matches {
		if len(match) > 1 {
			// Strip query parameters from filename
			filename := strings.Split(match[1], "?")[0]
			images = append(images, Image{Filename: filename})
		}
	}

	return images
}

// ExtractHeadings extracts all headings to build a TOC
func ExtractHeadings(content string) []Heading {
	var headings []Heading

	doc, err := html.Parse(strings.NewReader(content))
	if err != nil {
		return headings
	}

	var extract func(*html.Node)
	extract = func(n *html.Node) {
		if n.Type == html.ElementNode {
			level := 0
			switch n.Data {
			case "h1":
				level = 1
			case "h2":
				level = 2
			case "h3":
				level = 3
			case "h4":
				level = 4
			case "h5":
				level = 5
			case "h6":
				level = 6
			}
			if level > 0 {
				text := extractText(n)
				if text != "" {
					headings = append(headings, Heading{Level: level, Text: text})
				}
			}
		}
		for c := n.FirstChild; c != nil; c = c.NextSibling {
			extract(c)
		}
	}
	extract(doc)

	return headings
}

func extractText(n *html.Node) string {
	if n.Type == html.TextNode {
		return n.Data
	}
	var text string
	for c := n.FirstChild; c != nil; c = c.NextSibling {
		text += extractText(c)
	}
	return text
}

// Extract extracts all metadata from content
func Extract(content string) Metadata {
	return Metadata{
		Links:    ExtractLinks(content),
		Images:   ExtractImages(content),
		Headings: ExtractHeadings(content),
	}
}
