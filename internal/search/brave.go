package search

import (
	"fmt"
	"holetab/internal/favicon"
	"holetab/internal/model"
	"holetab/internal/utils"
	"net/url"
	"strings"

	"github.com/PuerkitoBio/goquery"
)

func GetBraveResults(q string) ([]model.SearchResult, error) {
	searchURL := fmt.Sprintf("https://search.brave.com/search?q=%s", url.QueryEscape(q))
	resp, err := utils.Fetch(searchURL)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		return nil, fmt.Errorf("brave search returned status %d", resp.StatusCode)
	}

	doc, err := goquery.NewDocumentFromReader(resp.Body)
	if err != nil {
		return nil, err
	}

	var results []model.SearchResult

	// Web results
	doc.Find("div.snippet:not(.standalone)").Each(func(i int, s *goquery.Selection) {
		title := strings.TrimSpace(s.Find(".title").First().Text())
		if title == "" {
			title = strings.TrimSpace(s.Find(".snippet-title").First().Text())
		}
		link, _ := s.Find("a[href]").First().Attr("href")

		// Description selectors
		desc := strings.TrimSpace(s.Find(".snippet-description").First().Text())
		if desc == "" {
			desc = strings.TrimSpace(s.Find(".snippet-content .snippet-description").First().Text())
		}
		if desc == "" {
			// New Brave structure (svelte)
			desc = strings.TrimSpace(s.Find(".generic-snippet .content").First().Text())
		}
		if desc == "" {
			desc = strings.TrimSpace(s.Find(".description").First().Text())
		}
		site := strings.TrimSpace(s.Find(".netloc").Text())

		if link == "" || strings.HasPrefix(link, "/") || strings.HasPrefix(link, "#") {
			return
		}

		results = append(results, model.SearchResult{
			URL:    link,
			Title:  title,
			Desc:   desc,
			Domain: site,
			Icon:   favicon.GetFaviconURL(link),
			Type:   model.TypeWeb,
		})
	})

	// Video results
	doc.Find(".carousel .card").Each(func(i int, s *goquery.Selection) {
		title := strings.TrimSpace(s.Find("h2").Text())
		link, _ := s.Find("a[href]").Attr("href")
		footer := strings.TrimSpace(s.Find(".card-footer").Text())

		if link == "" || strings.HasPrefix(link, "/") || strings.HasPrefix(link, "#") {
			return
		}

		results = append(results, model.SearchResult{
			URL:      link,
			Title:    title,
			SiteName: footer,
			Icon:     favicon.GetFaviconURL(link),
			Type:     model.TypeVideo,
		})
	})

	return results, nil
}
