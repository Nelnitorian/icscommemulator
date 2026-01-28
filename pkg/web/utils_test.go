package web

import (
	"bytes"
	"encoding/json"
	"net"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestDecodeJSONBodyRejectsUnknownFields(t *testing.T) {
	type payload struct {
		Name string `json:"name"`
	}

	body := []byte(`{"name":"demo","extra":1}`)
	req := httptest.NewRequest(http.MethodPost, "/", bytes.NewReader(body))
	rec := httptest.NewRecorder()

	var dst payload
	err := decodeJSONBody(rec, req, &dst)
	if err == nil {
		t.Fatalf("expected error for unknown fields")
	}
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("expected status 400, got %d", rec.Code)
	}
}

func TestExtractPathSegment(t *testing.T) {
	got := extractPathSegment("/api/networks/demo/", "/api/networks/")
	if got != "demo" {
		t.Fatalf("expected demo, got %s", got)
	}
}

func TestIncrementIP(t *testing.T) {
	ip := net.ParseIP("192.168.1.1")
	next := incrementIP(ip)
	if next.String() != "192.168.1.2" {
		t.Fatalf("expected 192.168.1.2, got %s", next.String())
	}
}

func TestToJSONFunc(t *testing.T) {
	data := map[string]string{"hello": "world"}
	js := toJSONFunc(data)
	var decoded map[string]string
	if err := json.Unmarshal([]byte(js), &decoded); err != nil {
		t.Fatalf("failed to unmarshal: %v", err)
	}
	if decoded["hello"] != "world" {
		t.Fatalf("unexpected decoded data: %v", decoded)
	}
}
