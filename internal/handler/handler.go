package handler

import (
	"net/http"
	"net/url"
	"strconv"

	"github.com/go-chi/chi/v5"
	"go.etcd.io/bbolt"

	"holetab/internal/db"
	"holetab/internal/favicon"
	"holetab/internal/model"
	"holetab/web/templates"
)

// Handler holds shared dependencies for all HTTP handlers.
type Handler struct {
	DB *bbolt.DB
}

// New returns a configured chi router wired to all application routes.
func New(database *bbolt.DB) http.Handler {
	h := &Handler{DB: database}

	r := chi.NewRouter()

	r.Get("/", h.Index)
	r.Post("/search", h.Search)
	r.Post("/links", h.AddLink)
	r.Put("/links/{id}", h.UpdateLink)
	r.Delete("/links/{id}", h.DeleteLink)
	r.Get("/links/{id}/move", h.MoveLink)

	return r
}

// Index handles GET / — renders the full page with the current link list.
func (h *Handler) Index(w http.ResponseWriter, r *http.Request) {
	links, err := db.GetAllLinks(h.DB)
	if err != nil {
		http.Error(w, "failed to load links", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	if err := templates.Index(links).Render(r.Context(), w); err != nil {
		http.Error(w, "render error", http.StatusInternalServerError)
	}
}

// AddLink handles POST /links — inserts a new link and returns the updated grid fragment.
func (h *Handler) AddLink(w http.ResponseWriter, r *http.Request) {
	if err := r.ParseForm(); err != nil {
		http.Error(w, "bad request", http.StatusBadRequest)
		return
	}

	link := model.Link{
		Name: r.FormValue("name"),
		Href: r.FormValue("href"),
		Img:  favicon.GetFaviconURL(r.FormValue("href")),
	}

	if err := db.AddLink(h.DB, link); err != nil {
		http.Error(w, "failed to add link", http.StatusInternalServerError)
		return
	}

	h.renderGrid(w, r)
}

// UpdateLink handles PUT /links/{id} — updates an existing link and returns the updated grid fragment.
func (h *Handler) UpdateLink(w http.ResponseWriter, r *http.Request) {
	id, err := parseID(r)
	if err != nil {
		http.Error(w, "invalid id", http.StatusBadRequest)
		return
	}

	if err := r.ParseForm(); err != nil {
		http.Error(w, "bad request", http.StatusBadRequest)
		return
	}

	link := model.Link{
		ID:   id,
		Name: r.FormValue("name"),
		Href: r.FormValue("href"),
		Img:  favicon.GetFaviconURL(r.FormValue("href")),
	}

	// Preserve the existing position by loading the old record first.
	existing, err := db.GetAllLinks(h.DB)
	if err != nil {
		http.Error(w, "internal error", http.StatusInternalServerError)
		return
	}
	for _, l := range existing {
		if l.ID == id {
			link.Position = l.Position
			break
		}
	}

	if err := db.UpdateLink(h.DB, link); err != nil {
		http.Error(w, "failed to update link", http.StatusInternalServerError)
		return
	}

	h.renderGrid(w, r)
}

// DeleteLink handles DELETE /links/{id} — removes a link and returns the updated grid fragment.
func (h *Handler) DeleteLink(w http.ResponseWriter, r *http.Request) {
	id, err := parseID(r)
	if err != nil {
		http.Error(w, "invalid id", http.StatusBadRequest)
		return
	}

	if err := db.DeleteLink(h.DB, id); err != nil {
		http.Error(w, "failed to delete link", http.StatusInternalServerError)
		return
	}

	h.renderGrid(w, r)
}

// MoveLink handles GET /links/{id}/move?dir=up|down — reorders a link and returns the updated grid.
func (h *Handler) MoveLink(w http.ResponseWriter, r *http.Request) {
	id, err := parseID(r)
	if err != nil {
		http.Error(w, "invalid id", http.StatusBadRequest)
		return
	}

	dir := r.URL.Query().Get("dir")
	if dir != "up" && dir != "down" {
		http.Error(w, "dir must be up or down", http.StatusBadRequest)
		return
	}

	if err := db.MoveLink(h.DB, id, dir); err != nil {
		http.Error(w, "failed to move link", http.StatusInternalServerError)
		return
	}

	h.renderGrid(w, r)
}

// Search handles POST /search — builds the engine search URL and redirects via HX-Redirect.
func (h *Handler) Search(w http.ResponseWriter, r *http.Request) {
	if err := r.ParseForm(); err != nil {
		http.Error(w, "bad request", http.StatusBadRequest)
		return
	}

	q := url.QueryEscape(r.FormValue("q"))

	var searchURL string
	switch r.FormValue("engine") {
	case "duckduckgo":
		searchURL = "https://duckduckgo.com/?q=" + q
	case "bing":
		searchURL = "https://www.bing.com/search?q=" + q
	case "brave":
		searchURL = "https://search.brave.com/search?q=" + q
	default:
		searchURL = "https://www.google.com/search?q=" + q
	}

	w.Header().Set("HX-Redirect", searchURL)
	w.WriteHeader(http.StatusNoContent)
}

// renderGrid is a helper that fetches the current link list and renders the
// grid fragment — used as the HTMX swap target for all mutating operations.
func (h *Handler) renderGrid(w http.ResponseWriter, r *http.Request) {
	links, err := db.GetAllLinks(h.DB)
	if err != nil {
		http.Error(w, "internal error", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	if err := templates.LinkGrid(links).Render(r.Context(), w); err != nil {
		http.Error(w, "render error", http.StatusInternalServerError)
	}
}

// parseID extracts and validates the {id} URL parameter.
func parseID(r *http.Request) (uint64, error) {
	return strconv.ParseUint(chi.URLParam(r, "id"), 10, 64)
}
