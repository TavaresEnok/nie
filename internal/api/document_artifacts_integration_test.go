package api

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"
	"time"

	"nie/internal/config"
	"nie/internal/domain"
	"nie/internal/storage"
	"nie/internal/store"
)

func TestDocumentArtifactsIntegration(t *testing.T) {
	dsn := os.Getenv("NIE_TEST_DATABASE_URL")
	if dsn == "" {
		t.Skip("NIE_TEST_DATABASE_URL não definido")
	}
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	st, err := store.New(ctx, dsn)
	if err != nil {
		t.Fatal(err)
	}
	defer st.Close()
	if err := st.Migrate(ctx); err != nil {
		t.Fatal(err)
	}
	root := t.TempDir()
	objects, err := storage.New(root)
	if err != nil {
		t.Fatal(err)
	}
	raw := []byte("<html><body>Original verificável</body></html>")
	object, err := objects.Put(raw)
	if err != nil {
		t.Fatal(err)
	}
	camp, err := st.CreateCampaign(ctx, "artifact integration", nil, domain.Budget{MaxURLs: 10, MaxBytes: 1 << 20, MaxDomains: 2})
	if err != nil {
		t.Fatal(err)
	}
	defer st.Pool().Exec(context.Background(), `DELETE FROM campaigns WHERE id=$1::uuid`, camp.ID)
	docID, _, err := st.RecordDocument(ctx, store.DocumentRecord{SHA256: object.SHA256, MIME: "text/html", Size: object.Size, Text: "old extraction", Parser: "fixture", ParserVersion: "1", CampaignID: camp.ID, URL: "https://example.org/", NormalizedURL: "https://example.org/", Domain: "example.org", Passages: []domain.Passage{{Ordinal: 0, Text: "old extraction"}}})
	if err != nil {
		t.Fatal(err)
	}
	defer st.Pool().Exec(context.Background(), `DELETE FROM documents WHERE id=$1::uuid`, docID)
	handler := New(st, config.Config{DataDir: root, APIToken: "test-token"}, discardLogger())
	request := func(method, path string, authorized bool) *httptest.ResponseRecorder {
		t.Helper()
		r := httptest.NewRequest(method, path, nil)
		if authorized {
			r.Header.Set("X-API-Token", "test-token")
		}
		w := httptest.NewRecorder()
		handler.ServeHTTP(w, r)
		return w
	}
	base := "/api/documents/" + docID
	if w := request("GET", base+"/original", false); w.Code != 401 {
		t.Fatalf("original sem autenticação: %d", w.Code)
	}
	w := request("GET", base+"/original", true)
	if w.Code != 200 || !bytes.Equal(w.Body.Bytes(), raw) || w.Header().Get("Content-Type") != "application/octet-stream" {
		t.Fatalf("original: %d %s", w.Code, w.Body.String())
	}
	w = request(http.MethodPost, base+"/reprocess", true)
	if w.Code != 200 {
		t.Fatalf("reprocessar: %d %s", w.Code, w.Body.String())
	}
	var run domain.ExtractionRun
	if err := json.Unmarshal(w.Body.Bytes(), &run); err != nil {
		t.Fatal(err)
	}
	if run.Parser != "nie-html" || run.Text != "Original verificável" {
		t.Fatalf("extração inválida: %+v", run)
	}
	history := request("GET", base+"/extractions", true)
	var runs []domain.ExtractionRun
	if err := json.Unmarshal(history.Body.Bytes(), &runs); err != nil || len(runs) != 2 {
		t.Fatalf("histórico=%s %v", history.Body.String(), err)
	}
	for _, r := range runs {
		w := request("GET", base+"/extractions/"+r.ID, true)
		if w.Code != 200 {
			t.Fatalf("extração histórica inacessível: %d", w.Code)
		}
	}
	if err := os.WriteFile(object.Path, bytes.Repeat([]byte("X"), len(raw)), 0600); err != nil {
		t.Fatal(err)
	}
	if w := request("GET", base+"/original", true); w.Code != 500 {
		t.Fatalf("corrupção não detectada: %d", w.Code)
	}
	if w := request("POST", base+"/reprocess", true); w.Code != 500 {
		t.Fatalf("objeto corrompido reprocessado: %d", w.Code)
	}
}
