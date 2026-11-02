package api

import (
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"nie/internal/config"
)

func discardLogger() *slog.Logger {
	return slog.New(slog.NewTextHandler(io.Discard, nil))
}

func TestAuthorized(t *testing.T) {
	s := &Server{cfg: config.Config{APIToken: "segredo"}}
	req := httptest.NewRequest(http.MethodGet, "/api/search?q=x", nil)
	if s.authorized(req) {
		t.Fatal("sem token deveria negar")
	}
	req.Header.Set("X-API-Token", "segredo")
	if !s.authorized(req) {
		t.Fatal("X-API-Token correto deveria permitir")
	}
	req.Header.Del("X-API-Token")
	req.Header.Set("Authorization", "Bearer segredo")
	if !s.authorized(req) {
		t.Fatal("Bearer correto deveria permitir")
	}
	req.Header.Set("Authorization", "Bearer errado")
	if s.authorized(req) {
		t.Fatal("token errado deveria negar")
	}

	public := httptest.NewRequest(http.MethodGet, "/api/health", nil)
	if !s.authorized(public) {
		t.Fatal("healthcheck deveria ser público")
	}
	schema := httptest.NewRequest(http.MethodGet, "/api/schemas/Campaign.json", nil)
	if !s.authorized(schema) {
		t.Fatal("schema OpenAPI deveria ser público")
	}
	openapiJSON := httptest.NewRequest(http.MethodGet, "/api/openapi.json", nil)
	if !s.authorized(openapiJSON) {
		t.Fatal("openapi.json deveria ser público")
	}
	openapiYAML := httptest.NewRequest(http.MethodGet, "/api/openapi.yaml", nil)
	if !s.authorized(openapiYAML) {
		t.Fatal("openapi.yaml deveria ser público")
	}
	root := httptest.NewRequest(http.MethodGet, "/", nil)
	if !s.authorized(root) {
		t.Fatal("SPA deveria ser público")
	}
}

func TestMetaAndSecurityHeaders(t *testing.T) {
	h := New(nil, config.Config{}, discardLogger())
	req := httptest.NewRequest(http.MethodGet, "/api/meta", nil)
	w := httptest.NewRecorder()
	h.ServeHTTP(w, req)
	if w.Code != http.StatusOK {
		t.Fatalf("status = %d, corpo = %s", w.Code, w.Body.String())
	}
	if got := w.Header().Get("X-Content-Type-Options"); got != "nosniff" {
		t.Errorf("X-Content-Type-Options = %q", got)
	}
	if got := w.Header().Get("X-Frame-Options"); got != "DENY" {
		t.Errorf("X-Frame-Options = %q", got)
	}
	if !strings.Contains(w.Body.String(), `"auth_required":false`) {
		t.Errorf("corpo inesperado: %s", w.Body.String())
	}
}

func TestUnauthorizedCampaign(t *testing.T) {
	h := New(nil, config.Config{APIToken: "segredo"}, discardLogger())
	req := httptest.NewRequest(http.MethodGet, "/api/campaigns", nil)
	w := httptest.NewRecorder()
	h.ServeHTTP(w, req)
	if w.Code != http.StatusUnauthorized {
		t.Fatalf("status = %d, quer 401", w.Code)
	}
	if !strings.Contains(w.Body.String(), "unauthorized") {
		t.Errorf("corpo inesperado: %s", w.Body.String())
	}
}
