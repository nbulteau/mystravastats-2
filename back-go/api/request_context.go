package api

import (
	"crypto/rand"
	"encoding/hex"
	"net/http"
	"regexp"
	"strings"
)

var validRequestID = regexp.MustCompile(`^[A-Za-z0-9._:-]{1,128}$`)

// Use one identifier throughout routing, error responses and generation diagnostics.
func apiRequestContext(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/api" && !strings.HasPrefix(r.URL.Path, "/api/") {
			next.ServeHTTP(w, r)
			return
		}
		id := strings.TrimSpace(r.Header.Get("X-Request-Id"))
		if !validRequestID.MatchString(id) {
			var value [16]byte
			_, _ = rand.Read(value[:])
			id = hex.EncodeToString(value[:])
		}
		r = r.Clone(r.Context())
		r.Header.Set("X-Request-Id", id)
		w.Header().Set("X-Request-Id", id)
		if r.Method == http.MethodHead {
			w = headResponseWriter{w}
		}
		next.ServeHTTP(w, r)
	})
}

type headResponseWriter struct{ http.ResponseWriter }

func (w headResponseWriter) Write(p []byte) (int, error) { return len(p), nil }
