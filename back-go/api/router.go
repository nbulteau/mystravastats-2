package api

import (
	"github.com/gorilla/mux"
	"github.com/swaggo/http-swagger"
	"mystravastats/domain"
	"net/http"
)

func NewRouter() *mux.Router {

	router := mux.NewRouter().StrictSlash(true)
	for _, route := range routes {
		var handler http.Handler

		handler = route.HandlerFunc
		handler = domain.Logger(handler, route.Name)

		router.
			Methods(route.Method).
			Path(route.Pattern).
			Name(route.Name).
			Handler(handler)

	}

	// Ajout de la route Swagger UI
	router.PathPrefix("/swagger/").Handler(httpSwagger.WrapHandler)

	return router
}
