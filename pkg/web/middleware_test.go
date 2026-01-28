package web

import (
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestCorsMiddlewareAllowsOrigin(t *testing.T) {
	server := &Server{
		corsAllowedOrigins: map[string]struct{}{
			"http://example.com": {},
		},
	}

	called := false
	next := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		called = true
		w.WriteHeader(http.StatusOK)
	})

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.Header.Set("Origin", "http://example.com")
	rec := httptest.NewRecorder()

	server.corsMiddleware(next).ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d", rec.Code)
	}
	if !called {
		t.Fatalf("expected next handler to be called")
	}
	if got := rec.Header().Get("Access-Control-Allow-Origin"); got != "http://example.com" {
		t.Fatalf("unexpected allow origin header: %s", got)
	}
}

func TestCorsMiddlewareBlocksOrigin(t *testing.T) {
	server := &Server{
		corsAllowedOrigins: map[string]struct{}{
			"http://good.com": {},
		},
	}

	called := false
	next := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		called = true
		w.WriteHeader(http.StatusOK)
	})

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.Header.Set("Origin", "http://bad.com")
	rec := httptest.NewRecorder()

	server.corsMiddleware(next).ServeHTTP(rec, req)

	if rec.Code != http.StatusForbidden {
		t.Fatalf("expected status 403, got %d", rec.Code)
	}
	if called {
		t.Fatalf("expected next handler to not be called")
	}
}

func TestCorsMiddlewareHandlesOptions(t *testing.T) {
	server := &Server{
		corsAllowedOrigins: map[string]struct{}{
			"http://example.com": {},
		},
	}

	called := false
	next := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		called = true
		w.WriteHeader(http.StatusOK)
	})

	req := httptest.NewRequest(http.MethodOptions, "/", nil)
	req.Header.Set("Origin", "http://example.com")
	rec := httptest.NewRecorder()

	server.corsMiddleware(next).ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d", rec.Code)
	}
	if called {
		t.Fatalf("expected next handler to not be called for OPTIONS")
	}
}
