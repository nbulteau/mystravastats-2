package api

import (
	"encoding/json"
	"net/http"
	"strings"

	"mystravastats/internal/platform/runtimeconfig"
)

var safeRequestMethods = map[string]struct{}{
	http.MethodGet:     {},
	http.MethodHead:    {},
	http.MethodOptions: {},
}

// mutationOriginGuard rejects browser-initiated cross-origin mutations unless
// their Origin is explicitly present in the CORS allow-list. Requests without
// an Origin remain available to the local CLI and integration tooling.
func mutationOriginGuard(next http.Handler) http.Handler {
	allowedOrigins := make(map[string]struct{}, len(runtimeconfig.CORSAllowedOrigins()))
	for _, origin := range runtimeconfig.CORSAllowedOrigins() {
		allowedOrigins[strings.TrimSpace(origin)] = struct{}{}
	}

	return http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		if _, safe := safeRequestMethods[request.Method]; safe {
			next.ServeHTTP(writer, request)
			return
		}

		origin := strings.TrimSpace(request.Header.Get("Origin"))
		if origin == "" {
			next.ServeHTTP(writer, request)
			return
		}
		if _, allowed := allowedOrigins[origin]; allowed {
			next.ServeHTTP(writer, request)
			return
		}

		writer.Header().Set("Content-Type", "application/json")
		writer.WriteHeader(http.StatusForbidden)
		_ = json.NewEncoder(writer).Encode(map[string]any{
			"code":    http.StatusForbidden,
			"message": "cross-origin mutation rejected",
		})
	})
}
