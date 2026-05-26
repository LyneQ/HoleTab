package search

import "holetab/internal/model"

// MergeResults fusionne plusieurs slices de résultats, déduplique par URL.
// Les résultats de type TypeVideo sont regroupés et insérés à la position 2.
func MergeResults(sources ...[]model.SearchResult) []model.SearchResult {
	seen := make(map[string]bool)
	var webResults []model.SearchResult
	var videoResults []model.SearchResult

	for _, source := range sources {
		for _, res := range source {
			if res.URL == "" || seen[res.URL] {
				continue
			}
			seen[res.URL] = true

			if res.Type == model.TypeVideo {
				if len(videoResults) < 5 {
					videoResults = append(videoResults, res)
				}
			} else {
				webResults = append(webResults, res)
			}
		}
	}

	finalResults := webResults
	if len(videoResults) > 0 {
		pos := 2
		if len(finalResults) < pos {
			pos = len(finalResults)
		}

		// Insert videos at pos
		finalResults = append(finalResults[:pos], append(videoResults, finalResults[pos:]...)...)
	}

	return finalResults
}
