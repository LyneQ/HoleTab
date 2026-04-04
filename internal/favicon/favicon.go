package favicon

import (
	"fmt"
	"net/url"
)

// GetFaviconURL returns a best-effort favicon URL for the given raw href.
// It uses the Google favicon service as a reliable, dependency-free fallback.
// Returns an empty string if the href cannot be parsed.
func GetFaviconURL(href string) string {
	u, err := url.Parse(href)
	if err != nil || u.Host == "" {
		return ""
	}

	origin := fmt.Sprintf("%s://%s", u.Scheme, u.Host)
	return fmt.Sprintf("https://www.google.com/s2/favicons?domain=%s&sz=64", url.QueryEscape(origin))
}
