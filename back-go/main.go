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
	"context"
	"embed"
	"errors"
	"flag"
	"io/fs"
	"log"
	"mystravastats/api"
	"mystravastats/internal/platform/activityprovider"
	"mystravastats/internal/platform/runtimeconfig"
	"net"
	"net/http"
	"os"
	"os/signal"
	"strconv"
	"strings"
	"syscall"
	"time"

	_ "mystravastats/docs" // Import for generated Swagger documentation
)

//go:embed all:public
var public embed.FS

func main() {
	// Define a debug flag
	debug := flag.Bool("debug", false, "run in debug mode")
	host := flag.String("host", "localhost", "server host")
	port := flag.String("port", "8080", "server port")
	flag.Parse()

	// Get host and port from environment variables when provided.
	if envHost := runtimeconfig.FirstStringValue("", "SERVER_HOST", "HOST"); envHost != "" {
		*host = envHost
	}
	if envPort := runtimeconfig.StringValue("PORT", ""); envPort != "" {
		*port = envPort
	}

	// Validate that the port is a valid number in range [1, 65535].
	if portNum, err := strconv.Atoi(*port); err != nil || portNum < 1 || portNum > 65535 {
		log.Fatalf("invalid port %q: must be a number between 1 and 65535", *port)
	}

	// Eager initialization keeps cache loading and background refresh
	// behavior unchanged from a user perspective at startup.
	activityprovider.Init(*port)

	// Start the generated-route cache eviction loop; it stops when the
	// application exits via context cancellation.
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	api.StartCacheEviction(ctx)

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
			}

			// Hashed assets can be cached for a long time
			w.Header().Set("Cache-Control", "public, max-age=31536000, immutable")

			staticFileHandler.ServeHTTP(w, r)
		})
		router.PathPrefix("/").Handler(cacheControlHandler)
	}

	// Apply the CORS middleware to the router
	handler := newCORSHandler(router)

	addr := net.JoinHostPort(*host, *port)
	log.Printf("Starting server on http://%s", displayAddress(*host, *port))
	srv := &http.Server{
		Addr:         addr,
		Handler:      handler,
		ReadTimeout:  15 * time.Second,
		WriteTimeout: 60 * time.Second,
		IdleTimeout:  120 * time.Second,
	}

	// Graceful shutdown
	go func() {
		if err := srv.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			log.Fatalf("could not listen on %s: %v\n", addr, err)
		}
	}()

	stop := make(chan os.Signal, 1)
	signal.Notify(stop, syscall.SIGINT, syscall.SIGTERM)
	<-stop

	log.Println("Shutting down server...")

	shutdownCtx, cancelShutdown := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancelShutdown()

	if err := srv.Shutdown(shutdownCtx); err != nil {
		log.Fatalf("Server Shutdown Failed:%+v", err)
	}

	log.Println("Server gracefully stopped")
}

func displayAddress(host, port string) string {
	displayHost := host
	if displayHost == "" || displayHost == "0.0.0.0" || displayHost == "::" {
		displayHost = "localhost"
	}
	return net.JoinHostPort(displayHost, port)
}
