package handler

import (
	"net/http"
	"net/url"
	"strconv"
	"strings"

	"github.com/go-chi/chi/v5"
	"go.etcd.io/bbolt"

	"holetab/internal/bookmarks"
	"holetab/internal/config"
	"holetab/internal/db"
	"holetab/internal/favicon"
	"holetab/internal/model"
	"holetab/internal/weather"
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
	r.Put("/links/reorder", h.ReorderLinks)
	r.Get("/export", h.Export)
	r.Post("/import", h.Import)

	r.Get("/widgets/weather", h.GetWeather)
	r.Put("/widgets/weather/config", h.UpdateWeatherConfig)
	r.Put("/widgets/weather/toggle", h.ToggleWeather)

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

	weatherEnabledStr, _ := db.GetConfig(h.DB, "weather_enabled")
	weatherEnabled := weatherEnabledStr == "true"
	weatherLocation, _ := db.GetConfig(h.DB, "weather_location")
	var lat, lon string
	if parts := strings.Split(weatherLocation, ","); len(parts) == 2 {
		lat, lon = parts[0], parts[1]
	}

	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	if err := templates.Index(links, h.DevMode, weatherEnabled, lat, lon).Render(r.Context(), w); err != nil {
		http.Error(w, "render error", http.StatusInternalServerError)
	}
}

// AddLink handles POST /links — inserts a new link and returns the updated grid fragment.
func (h *Handler) AddLink(w http.ResponseWriter, r *http.Request) {
	if err := r.ParseForm(); err != nil {
		http.Error(w, "bad request", http.StatusBadRequest)
		return
	}

	name := r.FormValue("name")
	href := r.FormValue("href")
	itemType := r.FormValue("type")

	if itemType == "" {
		itemType = "link"
	}

	img := r.FormValue("img")

	if itemType == "link" {
		if name == "" {
			name = generateName(href)
		}
		if img == "" {
			img = favicon.GetFaviconURL(href)
		}
	}

	link := model.Link{
		Type: itemType,
		Name: name,
		Href: href,
		Img:  img,
	}

	if err := db.AddLink(h.DB, link); err != nil {
		http.Error(w, "failed to add link", http.StatusInternalServerError)
		return
	}

	h.renderGrid(w, r)
}

func generateName(href string) string {
	u, err := url.Parse(href)
	if err != nil || u.Host == "" {
		return href
	}

	// Try to get first words of domain
	host := u.Host
	host = strings.TrimPrefix(host, "www.")
	parts := strings.Split(host, ".")
	if len(parts) > 0 {
		name := parts[0]
		if len(name) > 0 {
			// Capitalize first letter
			return strings.ToUpper(name[:1]) + name[1:]
		}
	}

	return host
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

// ReorderLinks handles PUT /links/reorder — reorders multiple links.
func (h *Handler) ReorderLinks(w http.ResponseWriter, r *http.Request) {
	if err := r.ParseForm(); err != nil {
		http.Error(w, "bad request", http.StatusBadRequest)
		return
	}

	idsStr := r.Form["ids"]
	if len(idsStr) == 1 && strings.Contains(idsStr[0], ",") {
		idsStr = strings.Split(idsStr[0], ",")
	}

	var ids []uint64
	for _, s := range idsStr {
		id, err := strconv.ParseUint(s, 10, 64)
		if err != nil {
			continue
		}
		ids = append(ids, id)
	}

	if err := db.ReorderLinks(h.DB, ids); err != nil {
		http.Error(w, "failed to reorder", http.StatusInternalServerError)
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

// GetWeather handles GET /widgets/weather — fetches and renders the weather widget.
func (h *Handler) GetWeather(w http.ResponseWriter, r *http.Request) {
	enabled, _ := db.GetConfig(h.DB, "weather_enabled")
	if enabled != "true" {
		return
	}

	location, _ := db.GetConfig(h.DB, "weather_location")
	if location == "" {
		return
	}
	parts := strings.Split(location, ",")
	if len(parts) != 2 {
		return
	}
	info, err := weather.GetWeather(parts[0], parts[1])
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	if err := templates.WeatherWidget(info).Render(r.Context(), w); err != nil {
		http.Error(w, "render error", http.StatusInternalServerError)
	}
}

// UpdateWeatherConfig handles PUT /widgets/weather/config — saves the lat/lon to the DB.
func (h *Handler) UpdateWeatherConfig(w http.ResponseWriter, r *http.Request) {
	lat := r.FormValue("lat")
	lon := r.FormValue("lon")
	if lat == "" || lon == "" {
		http.Error(w, "lat and lon are required", http.StatusBadRequest)
		return
	}
	err := db.SetConfig(h.DB, "weather_location", lat+","+lon)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	w.Header().Set("HX-Trigger", "load-weather")
	w.WriteHeader(http.StatusOK)
}

// ToggleWeather handles PUT /widgets/weather/toggle — enables or disables the weather widget.
func (h *Handler) ToggleWeather(w http.ResponseWriter, r *http.Request) {
	if err := r.ParseForm(); err != nil {
		http.Error(w, "bad request", http.StatusBadRequest)
		return
	}

	// For a checkbox, HTMX sends nothing if unchecked, and "on" if checked (by default).
	// But we can also check if the key exists.
	enabled := "false"
	if r.FormValue("enabled") == "on" || r.Form.Has("enabled") {
		enabled = "true"
	}

	if err := db.SetConfig(h.DB, "weather_enabled", enabled); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("HX-Refresh", "true")
	w.WriteHeader(http.StatusOK)
}
