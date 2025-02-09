package main

import (
	"embed"
	"io/fs"
	"log"
	"mystravastats/api"
	"net/http"
	"strings"

	"github.com/rs/cors"
)

//go:embed public
var public embed.FS

func main() {
	// Create a new CORS handler
	c := cors.New(cors.Options{
		AllowedOrigins:   []string{"http://localhost"}, // Allow any port on localhost
		AllowedMethods:   []string{"GET", "POST", "PUT", "DELETE"},
		AllowedHeaders:   []string{"Content-Type", "Authorization"},
		AllowCredentials: true,
	})

	// Create a new router
	router := api.NewRouter()

	publicFS, err := fs.Sub(public, "public")
	if err != nil {
		log.Fatal(err)
	}

	// Serve static files from the "public" directory with cache headers
	staticFileHandler := http.FileServer(http.FS(publicFS))
	cacheControlHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if strings.HasSuffix(r.URL.Path, ".css") {
			w.Header().Set("Content-Type", "text/css")
		} else if strings.HasSuffix(r.URL.Path, ".js") {
			w.Header().Set("Content-Type", "application/javascript")
		}
		w.Header().Set("Cache-Control", "public, max-age=31536000")

		staticFileHandler.ServeHTTP(w, r)
	})
	router.PathPrefix("/").Handler(cacheControlHandler)

	// Apply the CORS middleware to the router
	handler := c.Handler(router)

	log.Fatal(http.ListenAndServe("localhost:8080", handler))
}
