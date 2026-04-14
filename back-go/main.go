// @title MyStravaStats API
// @version 1.0
// @description API for Strava statistics
// @termsOfService http://swagger.io/terms/
// @contact.name API Support
// @contact.url http://www.swagger.io/support
// @contact.email support@swagger.io
// @license.name Apache 2.0
// @license.url http://www.apache.org/licenses/LICENSE-2.0.html
// @host localhost:8080
// @BasePath /

package main

import (
	"embed"
	"flag"
	"fmt"
	"io/fs"
	"log"
	"mystravastats/api"
	"mystravastats/internal/services"
	"net/http"
	"os"
	"strings"

	_ "mystravastats/docs" // Import for generated Swagger documentation

	"github.com/rs/cors"
)

//go:embed public
var public embed.FS

func main() {
	// Define a debug flag
	debug := flag.Bool("debug", false, "run in debug mode")
	port := flag.String("port", "8080", "server port")
	flag.Parse()

	// Get port from environment variable if not provided as flag
	if envPort := os.Getenv("PORT"); envPort != "" {
		*port = envPort
	}

	// Eager initialization keeps cache loading and background refresh
	// behavior unchanged from a user perspective at startup.
	services.InitActivityProvider(*port)

	// Create a new CORS handler
	c := cors.New(cors.Options{
		AllowedOrigins:   []string{"http://localhost", "http://localhost:5173"}, // Allow any port on localhost and localhost:5173
		AllowedMethods:   []string{"GET", "POST", "PUT", "DELETE"},
		AllowedHeaders:   []string{"Content-Type", "Authorization"},
		AllowCredentials: true,
	})

	// Create a new router
	router := api.NewRouter()

	if !*debug {
		publicFS, err := fs.Sub(public, "public")
		if err != nil {
			log.Fatal(err)
		}

		// Serve static files from the "public" directory with cache headers.
		// index.html must never be long-cached (no-cache) so that new deployments
		// are picked up immediately by the browser.
		// Hashed assets (JS/CSS/images) are cached for 1 year.
		staticFileHandler := http.FileServer(http.FS(publicFS))
		cacheControlHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			path := r.URL.Path

			// Never serve SPA fallback for API paths.
			if strings.HasPrefix(path, "/api") {
				http.NotFound(w, r)
				return
			}

			// Set MIME types explicitly for JS and CSS
			if strings.HasSuffix(path, ".css") {
				w.Header().Set("Content-Type", "text/css")
			} else if strings.HasSuffix(path, ".js") {
				w.Header().Set("Content-Type", "application/javascript")
			}

			// index.html (and SPA fallback routes) must not be long-cached
			isHTML := path == "/" || path == "/index.html" || !strings.Contains(path, ".")
			if isHTML {
				w.Header().Set("Cache-Control", "no-cache, no-store, must-revalidate")
				w.Header().Set("Pragma", "no-cache")
				w.Header().Set("Expires", "0")

				// SPA fallback: always serve the app root to avoid FileServer redirect
				// loops caused by "/index.html" canonicalization.
				r2 := r.Clone(r.Context())
				r2.URL.Path = "/"
				staticFileHandler.ServeHTTP(w, r2)
				return
			} else {
				// Hashed assets can be cached for a long time
				w.Header().Set("Cache-Control", "public, max-age=31536000, immutable")
			}

			staticFileHandler.ServeHTTP(w, r)
		})
		router.PathPrefix("/").Handler(cacheControlHandler)
	}

	// Apply the CORS middleware to the router
	handler := c.Handler(router)

	addr := fmt.Sprintf("localhost:%s", *port)
	log.Printf("Starting server on http://%s", addr)
	log.Fatal(http.ListenAndServe(addr, handler))
}
