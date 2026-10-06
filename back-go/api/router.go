package api

import (
	"mystravastats/domain"
	"net/http"
	"sort"
	"strings"

	"github.com/gorilla/mux"
	"github.com/swaggo/http-swagger"
)

func NewRouter() *mux.Router {

	router := mux.NewRouter().StrictSlash(true)
	// Router middleware also covers redirects produced by StrictSlash.
	router.Use(apiRequestContext)
	pathRoutes := make([]*mux.Route, 0, len(routes))
	for _, route := range routes {
		var handler http.Handler

		handler = route.HandlerFunc
		handler = mutationOriginGuard(handler)
		handler = domain.Logger(handler, route.Name)
		methods := []string{route.Method}
		if route.Method == http.MethodGet {
			methods = append(methods, http.MethodHead)
		}
		pathRoutes = append(pathRoutes, mux.NewRouter().NewRoute().Path(route.Pattern))

		router.
			Path(route.Pattern).
			Methods(methods...).
			Name(route.Name).
			Handler(handler)

	}

	// Reserve API paths before the production SPA fallback is installed.
	apiFallback := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		allowed := map[string]bool{}
		for index, pathRoute := range pathRoutes {
			if pathRoute.Match(r, &mux.RouteMatch{}) {
				allowed[routes[index].Method] = true
			}
		}
		if len(allowed) == 0 {
			writeNotFound(w, "Resource not found", "Unknown API endpoint")
			return
		}
		if allowed[http.MethodGet] {
			allowed[http.MethodHead] = true
		}
		allowed[http.MethodOptions] = true
		methods := make([]string, 0, len(allowed))
		for method := range allowed {
			methods = append(methods, method)
		}
		sort.Strings(methods)
		w.Header().Set("Allow", strings.Join(methods, ", "))
		if r.Method == http.MethodOptions {
			w.WriteHeader(http.StatusOK)
			return
		}
		writeAPIError(w, http.StatusMethodNotAllowed, "Method Not Allowed", "HTTP method is not supported for this endpoint")
	})
	router.PathPrefix("/api/").Handler(apiFallback)
	router.Path("/api").Handler(apiFallback)

	// Add Swagger UI route
	router.PathPrefix("/swagger/").Handler(httpSwagger.WrapHandler)

	return router
}
