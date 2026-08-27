package api

import (
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestMutationOriginGuard(t *testing.T) {
	tests := []struct {
		name       string
		method     string
		origin     string
		wantStatus int
		wantCalls  int
	}{
		{name: "safe cross-origin read", method: http.MethodGet, origin: "https://example.test", wantStatus: http.StatusNoContent, wantCalls: 1},
		{name: "local mutation", method: http.MethodPost, origin: "http://localhost:5173", wantStatus: http.StatusNoContent, wantCalls: 1},
		{name: "command line mutation", method: http.MethodPut, wantStatus: http.StatusNoContent, wantCalls: 1},
		{name: "rejected browser mutation", method: http.MethodDelete, origin: "https://example.test", wantStatus: http.StatusForbidden},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			calls := 0
			handler := mutationOriginGuard(http.HandlerFunc(func(writer http.ResponseWriter, _ *http.Request) {
				calls++
				writer.WriteHeader(http.StatusNoContent)
			}))
			request := httptest.NewRequest(test.method, "/api/test", nil)
			if test.origin != "" {
				request.Header.Set("Origin", test.origin)
			}
			response := httptest.NewRecorder()

			handler.ServeHTTP(response, request)

			if response.Code != test.wantStatus {
				t.Fatalf("expected status %d, got %d", test.wantStatus, response.Code)
			}
			if calls != test.wantCalls {
				t.Fatalf("expected downstream handler to be called %d times, got %d", test.wantCalls, calls)
			}
		})
	}
}
