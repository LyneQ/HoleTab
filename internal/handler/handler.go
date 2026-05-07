package handler

import (
	"net/http"
	"net/url"
	"strconv"

	"github.com/go-chi/chi/v5"
	"go.etcd.io/bbolt"

	"holetab/internal/bookmarks"
	"holetab/internal/config"
	"holetab/internal/db"
	"holetab/internal/favicon"
	"holetab/internal/model"
	"holetab/web/templates"
)

// Handler holds shared dependencies for all HTTP handlers.
type Handler struct {
	DB      *bbolt.DB
	Config  *config.Config
	DevMode bool
}

// New returns a configured chi router wired to all application routes.
func New(database *bbolt.DB, cfg *config.Config, devMode bool) http.Handler {
	h := &Handler{
		DB:      database,
		Config:  cfg,
		DevMode: devMode,
	}

	r := chi.NewRouter()

	r.Get("/", h.Index)
	r.Post("/search", h.Search)
	r.Post("/links", h.AddLink)
	r.Put("/links/{id}", h.UpdateLink)
	r.Delete("/links/{id}", h.DeleteLink)
	r.Get("/links/{id}/move", h.MoveLink)
	r.Get("/export", h.Export)
	r.Post("/import", h.Import)

	if h.DevMode {
		r.Post("/reset", h.ResetLinks)
	}

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
	if err := templates.Index(links, h.DevMode).Render(r.Context(), w); err != nil {
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

	img := r.FormValue("img")
	if img == "" {
		img = favicon.GetFaviconURL(r.FormValue("href"))
	}
	link := model.Link{
		ID:   id,
		Name: r.FormValue("name"),
		Href: r.FormValue("href"),
		Img:  img,
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

// Export handles GET /export — triggers a download of the bookmarks in Netscape format.
func (h *Handler) Export(w http.ResponseWriter, r *http.Request) {
	links, err := db.GetAllLinks(h.DB)
	if err != nil {
		http.Error(w, "failed to load links", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.Header().Set("Content-Disposition", `attachment; filename="bookmarks.html"`)

	if err := bookmarks.Export(w, links); err != nil {
		http.Error(w, "export error", http.StatusInternalServerError)
	}
}

// Import handles POST /import — parses the uploaded file and adds new bookmarks.
func (h *Handler) Import(w http.ResponseWriter, r *http.Request) {
	// 10 MB max
	if err := r.ParseMultipartForm(10 << 20); err != nil {
		http.Error(w, "failed to parse multipart form", http.StatusBadRequest)
		return
	}

	file, _, err := r.FormFile("bookmarks")
	if err != nil {
		http.Error(w, "failed to get file", http.StatusBadRequest)
		return
	}
	defer file.Close()

	links, err := bookmarks.Import(file)
	if err != nil {
		http.Error(w, "import error", http.StatusInternalServerError)
		return
	}

	if err := db.AddLinks(h.DB, links); err != nil {
		http.Error(w, "failed to save links", http.StatusInternalServerError)
		return
	}

	// Redirect back to home to see the changes
	w.Header().Set("HX-Redirect", "/")
	w.WriteHeader(http.StatusNoContent)
}

// ResetLinks handles POST /reset — erases all links from the DB.
func (h *Handler) ResetLinks(w http.ResponseWriter, r *http.Request) {
	if !h.DevMode {
		http.Error(w, "forbidden", http.StatusForbidden)
		return
	}

	if err := db.ResetLinks(h.DB); err != nil {
		http.Error(w, "failed to reset links", http.StatusInternalServerError)
		return
	}

	h.renderGrid(w, r)
}
