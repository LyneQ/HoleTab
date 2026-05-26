package search

import (
	"holetab/internal/model"
	"holetab/web/templates"
	"net/http"

	"golang.org/x/sync/errgroup"
)

func Handler(w http.ResponseWriter, r *http.Request) {
	q := r.URL.Query().Get("q")
	if q == "" {
		http.Error(w, "Query parameter 'q' is required", http.StatusBadRequest)
		return
	}

	g, _ := errgroup.WithContext(r.Context())
	var braveRes, ddgRes []model.SearchResult

	g.Go(func() error {
		var err error
		braveRes, err = GetBraveResults(q)
		// On ignore l'erreur pour ne pas faire échouer toute la recherche
		if err != nil {
			// On pourrait logger ici
		}
		return nil
	})

	g.Go(func() error {
		var err error
		ddgRes, err = GetDDGResults(q)
		if err != nil {
			// On pourrait logger ici
		}
		return nil
	})

	_ = g.Wait()

	results := MergeResults(braveRes, ddgRes)

	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	if err := templates.SearchResultsPage(q, results).Render(r.Context(), w); err != nil {
		http.Error(w, "render error", http.StatusInternalServerError)
	}
}
