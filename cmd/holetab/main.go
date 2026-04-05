package main

import (
	"flag"
	"io/fs"
	"log"
	"net/http"

	"github.com/go-chi/chi/v5"

	"holetab/internal/config"
	"holetab/internal/db"
	"holetab/internal/handler"
	"holetab/web"
)

func main() {
	// 1. Load (or create) configuration.
	configPath := flag.String("config", "", "path to config file")
	flag.Parse()

	cfg, err := config.LoadConfig(*configPath)
	if err != nil {
		log.Fatalf("config: %v", err)
	}

	// 2. Open bbolt database.
	database, err := db.Open(cfg.Database.Path)
	if err != nil {
		log.Fatalf("db: %v", err)
	}
	defer database.Close()

	// 3. Build the application router.
	appRouter := handler.New(database)

	// 4. Mount static files from the embedded FS.
	staticFS, err := fs.Sub(web.StaticFiles, "static")
	if err != nil {
		log.Fatalf("static fs: %v", err)
	}

	r := chi.NewRouter()
	r.Mount("/", appRouter)
	r.Handle("/static/*", http.StripPrefix("/static/", http.FileServer(http.FS(staticFS))))

	// 5. Start HTTP server.
	addr := ":" + cfg.Server.Port
	log.Printf("HoleTab listening on http://localhost%s", addr)
	if err := http.ListenAndServe(addr, r); err != nil {
		log.Fatalf("server: %v", err)
	}
}
