package bookmarks

import (
	"fmt"
	"io"
	"time"

	"holetab/internal/model"
)

// BookmarkFile represents the Netscape Bookmark File Format.
// Reference: http://web.archive.org/web/20090226001358/http://bookmark-file-format.netscape.com/
type BookmarkFile struct {
	Title string
	Links []model.Link
}

// Export writes the bookmarks to the given writer in Netscape Bookmark File Format.
func Export(w io.Writer, links []model.Link) error {
	header := `<!DOCTYPE NETSCAPE-Bookmark-file-1>
<!-- This is an automatically generated file.
     It will be read and overwritten.
     DO NOT EDIT! -->
<META HTTP-EQUIV="Content-Type" CONTENT="text/html; charset=UTF-8">
<TITLE>Bookmarks</TITLE>
<H1>Bookmarks</H1>
<DL><p>
`
	if _, err := fmt.Fprint(w, header); err != nil {
		return err
	}

	for _, link := range links {
		// ADD_DATE is expected to be a unix timestamp
		// Since model.Link doesn't have a timestamp, we use current time for now or 0
		addDate := time.Now().Unix()

		// Netscape format uses <DT><A HREF="..." ADD_DATE="...">Name</A>
		// Some browsers also include ICON for favicons.
		line := fmt.Sprintf(`    <DT><A HREF="%s" ADD_DATE="%d">%s</A>
`, link.Href, addDate, link.Name)

		if _, err := fmt.Fprint(w, line); err != nil {
			return err
		}
	}

	footer := "</DL><p>\n"
	if _, err := fmt.Fprint(w, footer); err != nil {
		return err
	}

	return nil
}
