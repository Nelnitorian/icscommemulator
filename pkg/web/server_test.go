package web

import (
	"testing"
	"time"
)

func TestLoadCorsAllowedOrigins(t *testing.T) {
	t.Setenv("CORS_ALLOW_ORIGINS", "http://a.test, http://b.test,")
	origins := loadCorsAllowedOrigins()
	if len(origins) != 2 {
		t.Fatalf("expected 2 origins, got %d", len(origins))
	}
	if _, ok := origins["http://a.test"]; !ok {
		t.Fatalf("expected http://a.test to be allowed")
	}
	if _, ok := origins["http://b.test"]; !ok {
		t.Fatalf("expected http://b.test to be allowed")
	}
}

func TestNewServerTimeouts(t *testing.T) {
	server, err := NewServer("127.0.0.1", 0)
	if err != nil {
		t.Fatalf("NewServer() error: %v", err)
	}

	if server.httpServer.ReadTimeout != 15*time.Second {
		t.Fatalf("unexpected ReadTimeout: %v", server.httpServer.ReadTimeout)
	}
	if server.httpServer.ReadHeaderTimeout != 5*time.Second {
		t.Fatalf("unexpected ReadHeaderTimeout: %v", server.httpServer.ReadHeaderTimeout)
	}
	if server.httpServer.WriteTimeout != 15*time.Second {
		t.Fatalf("unexpected WriteTimeout: %v", server.httpServer.WriteTimeout)
	}
	if server.httpServer.IdleTimeout != 60*time.Second {
		t.Fatalf("unexpected IdleTimeout: %v", server.httpServer.IdleTimeout)
	}
	if server.httpServer.MaxHeaderBytes != 1<<20 {
		t.Fatalf("unexpected MaxHeaderBytes: %d", server.httpServer.MaxHeaderBytes)
	}
}
