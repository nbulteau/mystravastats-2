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
	"fmt"
	"io"
	"io/fs"
	"log"
	"mystravastats/api"
	"mystravastats/internal/helpers"
	"mystravastats/internal/platform/activityprovider"
	"mystravastats/internal/platform/runtimeconfig"
	"mystravastats/internal/sourcesync"
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
	go sourcesync.Synchronize("startup")

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
	displayAddr := displayAddress(*host, *port)
	listener, err := net.Listen("tcp", addr)
	if err != nil {
		log.Fatalf("could not listen on %s: %v\n", addr, err)
	}

	log.Printf("Starting server on http://%s", displayAddr)
	srv := &http.Server{
		Addr:         addr,
		Handler:      handler,
		ReadTimeout:  15 * time.Second,
		WriteTimeout: 60 * time.Second,
		IdleTimeout:  120 * time.Second,
	}

	// Graceful shutdown
	go func() {
		if err := srv.Serve(listener); err != nil && !errors.Is(err, http.ErrServerClosed) {
			log.Fatalf("could not listen on %s: %v\n", addr, err)
		}
	}()
	go openBrowserWhenServerIsReady(displayAddr)

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

func openBrowserWhenServerIsReady(displayAddr string) {
	appURL := fmt.Sprintf("http://%s", displayAddr)
	if !runtimeconfig.BoolValue("OPEN_BROWSER", true) {
		log.Printf("Browser auto-open disabled; open this URL manually: %s", appURL)
		return
	}

	readinessURL := appURL
	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()

	if !waitForServerReady(ctx, readinessURL, 200*time.Millisecond) {
		log.Printf("Server readiness check timed out; open this URL manually once ready: %s", appURL)
		return
	}

	log.Printf("Server ready; opening browser: %s", appURL)
	helpers.OpenBrowser(appURL)
	log.Printf("To view your Strava activities, open the following URL in your browser: %s", appURL)
}

func waitForServerReady(ctx context.Context, readinessURL string, interval time.Duration) bool {
	if interval <= 0 {
		interval = 200 * time.Millisecond
	}
	client := http.Client{Timeout: 500 * time.Millisecond}
	ticker := time.NewTicker(interval)
	defer ticker.Stop()

	for {
		if isServerReady(client, readinessURL) {
			return true
		}

		select {
		case <-ctx.Done():
			return false
		case <-ticker.C:
		}
	}
}

func isServerReady(client http.Client, readinessURL string) bool {
	response, err := client.Get(readinessURL)
	if err != nil {
		return false
	}
	defer response.Body.Close()
	_, _ = io.Copy(io.Discard, response.Body)
	return response.StatusCode >= http.StatusOK && response.StatusCode < http.StatusInternalServerError
}
