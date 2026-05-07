package bookmarks

import (
	"bufio"
	"io"
	"regexp"
	"strings"

	"holetab/internal/favicon"
	"holetab/internal/model"
)

var (
	// Very simple regex to find <A HREF="...">Name</A>
	// It's not perfect for all HTML, but Netscape format is usually predictable.
	linkRegex = regexp.MustCompile(`(?i)<A\s+[^>]*HREF=["']([^"']+)["'][^>]*>([^<]*)</A>`)
)

// Import parses the bookmarks from the given reader in Netscape Bookmark File Format.
func Import(r io.Reader) ([]model.Link, error) {
	var links []model.Link
	scanner := bufio.NewScanner(r)

	for scanner.Scan() {
		line := scanner.Text()
		matches := linkRegex.FindAllStringSubmatch(line, -1)
		for _, match := range matches {
			if len(match) >= 3 {
				href := match[1]
				name := match[2]
				if name == "" {
					name = href
				}
				links = append(links, model.Link{
					Name: strings.TrimSpace(name),
					Href: href,
					Img:  favicon.GetFaviconURL(href),
				})
			}
		}
	}

	if err := scanner.Err(); err != nil {
		return nil, err
	}

	return links, nil
}
