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

func GetDDGResults(q string) ([]model.SearchResult, error) {
	searchURL := fmt.Sprintf("https://html.duckduckgo.com/html/?q=%s", url.QueryEscape(q))
	resp, err := utils.Fetch(searchURL)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		return nil, fmt.Errorf("ddg search returned status %d", resp.StatusCode)
	}

	doc, err := goquery.NewDocumentFromReader(resp.Body)
	if err != nil {
		return nil, err
	}

	var results []model.SearchResult

	doc.Find(".result.results_links_deep").Each(func(i int, s *goquery.Selection) {
		title := strings.TrimSpace(s.Find(".result__title a").First().Text())
		rawLink, _ := s.Find(".result__title a[href]").First().Attr("href")

		desc := strings.TrimSpace(s.Find(".result__snippet").First().Text())
		if desc == "" {
			desc = strings.TrimSpace(s.Find(".js-result-snippet").First().Text())
		}

		if rawLink == "" {
			return
		}

		// Decode DDG redirect URL
		actualURL := rawLink
		if strings.Contains(rawLink, "uddg=") {
			u, err := url.Parse(rawLink)
			if err == nil {
				actualURL = u.Query().Get("uddg")
			}
		}

		// Extract domain
		domain := ""
		if u, err := url.Parse(actualURL); err == nil {
			domain = u.Host
		}

		results = append(results, model.SearchResult{
			URL:    actualURL,
			Title:  title,
			Desc:   desc,
			Domain: domain,
			Icon:   favicon.GetFaviconURL(actualURL),
			Type:   model.TypeWeb,
		})
	})

	return results, nil
}
