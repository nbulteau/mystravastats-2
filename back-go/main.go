package main

import (
	"embed"
	"flag"
	"io/fs"
	"log"
	"mystravastats/api"
	"net/http"
	"os"
	"strings"

	"github.com/joho/godotenv"
	"github.com/rs/cors"
)

//go:embed public
var public embed.FS

func main() {
	// Load environment variables from .env file if not already set
	loadEnvironmentVariables()

	// Define a debug flag
	debug := flag.Bool("debug", false, "run in debug mode")
	flag.Parse()

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
	}

	// Apply the CORS middleware to the router
	handler := c.Handler(router)

	log.Fatal(http.ListenAndServe("localhost:8080", handler))
}

func loadEnvironmentVariables() {
	err := godotenv.Overload()
	if err != nil {
		log.Println("No .env file found")
	} else {
		log.Println("Loaded .env file")
	}
	cachePath := os.Getenv("STRAVA_CACHE_PATH")
	if cachePath == "" {
		cachePath = "strava-cache" // default value if environment variable is not set
		log.Printf("STRAVA_CACHE_PATH not set. Using default cache path: %s\n", cachePath)
	} else {
		log.Printf("Using cache path: %s\n", cachePath)
	}
}
