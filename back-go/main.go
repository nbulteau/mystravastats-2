package main

import (
	"github.com/NYTimes/gziphandler"
	"github.com/rs/cors"
	"log"
	"mystravastats/api"
	"net/http"
	"path/filepath"
)

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

	// Serve static files from the "static" directory with Gzip compression and cache headers
	staticFileDirectory := http.Dir("./static")
	staticFileHandler := http.StripPrefix("/static/", http.FileServer(staticFileDirectory))
	gzipStaticFileHandler := gziphandler.GzipHandler(staticFileHandler)
	cacheControlHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if filepath.Ext(r.URL.Path) == ".css" {
			w.Header().Set("Content-Type", "text/css")
		} else if filepath.Ext(r.URL.Path) == ".js" {
			w.Header().Set("Content-Type", "application/javascript")
		}
		w.Header().Set("Cache-Control", "public, max-age=31536000")
		gzipStaticFileHandler.ServeHTTP(w, r)
	})
	router.Handle("/static/", cacheControlHandler)

	// Apply the CORS middleware to the router
	handler := c.Handler(router)

	log.Fatal(http.ListenAndServe(":8080", handler))
}
