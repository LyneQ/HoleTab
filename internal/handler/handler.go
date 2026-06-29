package handler

import (
	"context"
	"database/sql"
	"encoding/json"
	"log"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"

	"github.com/go-chi/chi/v5"

	"holetab/internal/bookmarks"
	"holetab/internal/config"
	"holetab/internal/db"
	"holetab/internal/favicon"
	"holetab/internal/model"
	"holetab/internal/search"
	"holetab/internal/weather"
	"holetab/web/templates"
	"holetab/web/templates/widget"
)

// Handler holds shared dependencies for all HTTP handlers.
type Handler struct {
	DB      *sql.DB
	Config  *config.Config
	DevMode bool
}

// New returns a configured chi router wired to all application routes.
func New(database *sql.DB, cfg *config.Config, devMode bool) http.Handler {
	h := &Handler{
		DB:      database,
		Config:  cfg,
		DevMode: devMode,
	}

	r := chi.NewRouter()

	r.Use(h.AuthMiddleware)

	r.Get("/", h.Index)
	r.Get("/search", search.Handler)
	r.Post("/search", h.Search)
	r.Post("/register", h.Register)
	r.Post("/login", h.Login)
	r.Post("/logout", h.Logout)
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

func (h *Handler) AuthMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		cookie, err := r.Cookie("session")
		if err == nil {
			user, _ := db.GetUserByToken(h.DB, cookie.Value)
			if user != nil {
				ctx := context.WithValue(r.Context(), "user", user)
				next.ServeHTTP(w, r.WithContext(ctx))
				return
			}
		}
		next.ServeHTTP(w, r)
	})
}

func (h *Handler) currentUser(r *http.Request) *model.User {
	u, _ := r.Context().Value("user").(*model.User)
	return u
}

func (h *Handler) currentUserID(r *http.Request) uint64 {
	if u := h.currentUser(r); u != nil {
		return u.ID
	}
	return 0
}

// Index handles GET / — renders the full page with the current link list.
func (h *Handler) Index(w http.ResponseWriter, r *http.Request) {
	user := h.currentUser(r)
	userID := uint64(0)
	if user != nil {
		userID = user.ID
	}

	links, err := db.GetAllLinks(h.DB, userID)
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
	if err := templates.Index(user, links, h.DevMode, weatherEnabled, lat, lon).Render(r.Context(), w); err != nil {
		http.Error(w, "render error", http.StatusInternalServerError)
	}
}

func (h *Handler) Register(w http.ResponseWriter, r *http.Request) {
	if err := r.ParseForm(); err != nil {
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		templates.RegisterForm("Bad request").Render(r.Context(), w)
		return
	}

	username := r.FormValue("username")
	password := r.FormValue("password")

	if username == "" || password == "" {
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		templates.RegisterForm("Username and password are required").Render(r.Context(), w)
		return
	}

	userID, err := db.CreateUser(h.DB, username, password)
	if err != nil {
		errorMessage := "Failed to create user"
		if strings.Contains(err.Error(), "UNIQUE constraint failed: users.username") {
			errorMessage = "Username already taken"
		}
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		templates.RegisterForm(errorMessage).Render(r.Context(), w)
		return
	}

	token, err := db.CreateSession(h.DB, userID)
	if err != nil {
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		templates.RegisterForm("Failed to create session").Render(r.Context(), w)
		return
	}

	http.SetCookie(w, &http.Cookie{
		Name:     "session",
		Value:    token,
		Expires:  time.Now().Add(24 * time.Hour),
		HttpOnly: true,
		Path:     "/",
	})

	w.Header().Set("HX-Redirect", "/")
}

func (h *Handler) Login(w http.ResponseWriter, r *http.Request) {
	if err := r.ParseForm(); err != nil {
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		templates.LoginForm("Bad request").Render(r.Context(), w)
		return
	}

	username := r.FormValue("username")
	password := r.FormValue("password")

	user, err := db.AuthenticateUser(h.DB, username, password)
	if err != nil {
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		templates.LoginForm("Invalid username or password").Render(r.Context(), w)
		return
	}

	token, err := db.CreateSession(h.DB, user.ID)
	if err != nil {
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		templates.LoginForm("Failed to create session").Render(r.Context(), w)
		return
	}

	http.SetCookie(w, &http.Cookie{
		Name:     "session",
		Value:    token,
		Expires:  time.Now().Add(24 * time.Hour),
		HttpOnly: true,
		Path:     "/",
	})

	w.Header().Set("HX-Redirect", "/")
}

func (h *Handler) Logout(w http.ResponseWriter, r *http.Request) {
	cookie, err := r.Cookie("session")
	if err == nil {
		_ = db.DeleteSession(h.DB, cookie.Value)
	}

	http.SetCookie(w, &http.Cookie{
		Name:     "session",
		Value:    "",
		Expires:  time.Unix(0, 0),
		HttpOnly: true,
		Path:     "/",
	})

	w.Header().Set("HX-Redirect", "/")
}

// AddLink handles POST /links — inserts a new link and returns the updated grid fragment.
func (h *Handler) AddLink(w http.ResponseWriter, r *http.Request) {
	userID := h.currentUserID(r)
	if userID == 0 {
		http.Error(w, "unauthorized", http.StatusUnauthorized)
		return
	}

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

	if err := db.AddLink(h.DB, userID, link); err != nil {
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
	userID := h.currentUserID(r)
	existing, err := db.GetAllLinks(h.DB, userID)
	if err != nil {
		http.Error(w, "internal error", http.StatusInternalServerError)
		return
	}
	for _, e := range existing {
		if e.ID == id {
			link.Position = e.Position
			link.Type = e.Type
			break
		}
	}

	if err := db.UpdateLink(h.DB, userID, link); err != nil {
		http.Error(w, "failed to update link", http.StatusInternalServerError)
		return
	}

	h.renderGrid(w, r)
}

// DeleteLink removes the link with the given id and recompacts positions
// so there are no gaps.
func (h *Handler) DeleteLink(w http.ResponseWriter, r *http.Request) {
	id, err := parseID(r)
	if err != nil {
		http.Error(w, "invalid id", http.StatusBadRequest)
		return
	}

	userID := h.currentUserID(r)
	if err := db.DeleteLink(h.DB, userID, id); err != nil {
		http.Error(w, "failed to delete link", http.StatusInternalServerError)
		return
	}

	h.renderGrid(w, r)
}

// MoveLink handles GET /links/{id}/move?dir=up|down — swaps the link with its neighbor.
func (h *Handler) MoveLink(w http.ResponseWriter, r *http.Request) {
	id, err := parseID(r)
	if err != nil {
		http.Error(w, "invalid id", http.StatusBadRequest)
		return
	}

	userID := h.currentUserID(r)
	dir := r.URL.Query().Get("dir")
	if err := db.MoveLink(h.DB, userID, id, dir); err != nil {
		http.Error(w, "failed to move link", http.StatusInternalServerError)
		return
	}

	h.renderGrid(w, r)
}

// ReorderLinks handles PUT /links/reorder — sets the position of all links.
func (h *Handler) ReorderLinks(w http.ResponseWriter, r *http.Request) {
	if err := r.ParseForm(); err != nil {
		http.Error(w, "bad request", http.StatusBadRequest)
		return
	}

	idsStr := r.FormValue("ids")
	var ids []uint64
	for _, s := range strings.Split(idsStr, ",") {
		if s == "" {
			continue
		}
		id, _ := strconv.ParseUint(s, 10, 64)
		ids = append(ids, id)
	}

	userID := h.currentUserID(r)
	if err := db.ReorderLinks(h.DB, userID, ids); err != nil {
		http.Error(w, "failed to reorder links", http.StatusInternalServerError)
		return
	}

	h.renderGrid(w, r)
}

// Search handles POST /search — redirects to the chosen search engine.
func (h *Handler) Search(w http.ResponseWriter, r *http.Request) {
	if err := r.ParseForm(); err != nil {
		http.Error(w, "bad request", http.StatusBadRequest)
		return
	}

	q := r.FormValue("q")
	engine := r.FormValue("engine")

	var target string
	switch engine {
	case "google":
		target = "https://www.google.com/search?q=" + url.QueryEscape(q)
	case "duckduckgo":
		target = "https://duckduckgo.com/?q=" + url.QueryEscape(q)
	case "bing":
		target = "https://www.bing.com/search?q=" + url.QueryEscape(q)
	case "brave":
		target = "https://search.brave.com/search?q=" + url.QueryEscape(q)
	default:
		target = "https://www.google.com/search?q=" + url.QueryEscape(q)
	}

	w.Header().Set("HX-Redirect", target)
}

func (h *Handler) renderGrid(w http.ResponseWriter, r *http.Request) {
	userID := h.currentUserID(r)
	links, err := db.GetAllLinks(h.DB, userID)
	if err != nil {
		http.Error(w, "failed to load links", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	if err := templates.LinkGrid(links).Render(r.Context(), w); err != nil {
		http.Error(w, "render error", http.StatusInternalServerError)
	}
}

func parseID(r *http.Request) (uint64, error) {
	return strconv.ParseUint(chi.URLParam(r, "id"), 10, 64)
}

// Export handles GET /export — returns a JSON of all bookmarks.
func (h *Handler) Export(w http.ResponseWriter, r *http.Request) {
	userID := h.currentUserID(r)
	links, err := db.GetAllLinks(h.DB, userID)
	if err != nil {
		http.Error(w, "export failed", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("Content-Disposition", "attachment; filename=\"holetab-backup.json\"")
	json.NewEncoder(w).Encode(links)
}

// Import handles POST /import — replaces all bookmarks with those from a file.
func (h *Handler) Import(w http.ResponseWriter, r *http.Request) {
	userID := h.currentUserID(r)
	if userID == 0 {
		http.Error(w, "unauthorized", http.StatusUnauthorized)
		return
	}

	file, header, err := r.FormFile("bookmarks")
	if err != nil {
		http.Error(w, "bad request", http.StatusBadRequest)
		return
	}
	defer file.Close()

	var links []model.Link
	// Try to detect format based on content type or extension
	isJSON := strings.HasSuffix(strings.ToLower(header.Filename), ".json") ||
		header.Header.Get("Content-Type") == "application/json"

	if isJSON {
		if err := json.NewDecoder(file).Decode(&links); err != nil {
			http.Error(w, "invalid json", http.StatusBadRequest)
			return
		}
	} else {
		// Default to HTML bookmark format (Netscape)
		links, err = bookmarks.Import(file)
		if err != nil {
			http.Error(w, "invalid bookmark file", http.StatusBadRequest)
			return
		}
	}

	if len(links) == 0 {
		http.Error(w, "no bookmarks found in file", http.StatusBadRequest)
		return
	}

	if err := db.ResetLinks(h.DB, userID); err != nil {
		http.Error(w, "import failed (reset)", http.StatusInternalServerError)
		return
	}
	if err := db.AddLinks(h.DB, userID, links); err != nil {
		http.Error(w, "import failed (add)", http.StatusInternalServerError)
		return
	}

	w.Header().Set("HX-Redirect", "/")
}

// ResetLinks handles POST /reset — deletes all links.
func (h *Handler) ResetLinks(w http.ResponseWriter, r *http.Request) {
	userID := h.currentUserID(r)
	if err := db.ResetLinks(h.DB, userID); err != nil {
		http.Error(w, "reset failed", http.StatusInternalServerError)
		return
	}
	h.renderGrid(w, r)
}

func (h *Handler) GetWeather(w http.ResponseWriter, r *http.Request) {
	lat, _ := db.GetConfig(h.DB, "weather_lat")
	lon, _ := db.GetConfig(h.DB, "weather_lon")

	if lat == "" || lon == "" {
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		_ = widget.WeatherWidget(nil).Render(r.Context(), w)
		return
	}

	wData, err := weather.GetWeather(lat, lon)
	if err != nil {
		log.Printf("weather error: %v", err)
	}

	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	_ = widget.WeatherWidget(wData).Render(r.Context(), w)
}

func (h *Handler) UpdateWeatherConfig(w http.ResponseWriter, r *http.Request) {
	if err := r.ParseForm(); err != nil {
		http.Error(w, "bad request", http.StatusBadRequest)
		return
	}

	lat := r.FormValue("lat")
	lon := r.FormValue("lon")

	_ = db.SetConfig(h.DB, "weather_location", lat+","+lon)
	_ = db.SetConfig(h.DB, "weather_enabled", "true")

	w.Header().Set("HX-Trigger", "load-weather")
	w.WriteHeader(http.StatusOK)
}

func (h *Handler) ToggleWeather(w http.ResponseWriter, r *http.Request) {
	if err := r.ParseForm(); err != nil {
		http.Error(w, "bad request", http.StatusBadRequest)
		return
	}

	enabled := r.FormValue("enabled") == "on"
	_ = db.SetConfig(h.DB, "weather_enabled", strconv.FormatBool(enabled))

	w.Header().Set("HX-Refresh", "true")
	w.WriteHeader(http.StatusOK)
}
