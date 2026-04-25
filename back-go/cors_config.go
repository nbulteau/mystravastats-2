package main

import (
	"mystravastats/internal/platform/runtimeconfig"
	"net/http"

	"github.com/rs/cors"
)

func newCORSHandler(next http.Handler) http.Handler {
	return cors.New(corsOptions()).Handler(next)
}

func corsOptions() cors.Options {
	return cors.Options{
		AllowedOrigins:   runtimeconfig.CORSAllowedOrigins(),
		AllowedMethods:   runtimeconfig.CORSAllowedMethods(),
		AllowedHeaders:   runtimeconfig.CORSAllowedHeaders(),
		AllowCredentials: true,
	}
}
