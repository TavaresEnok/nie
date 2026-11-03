package api

import (
	"context"
	"encoding/json"
	"net/http/httptest"
	"nie/internal/config"
	"nie/internal/domain"
	"nie/internal/storage"
	"nie/internal/store"
	"nie/internal/testfixture"
	"os"
	"strings"
	"testing"
	"time"
)

func TestPDFOCRReprocessIntegration(t *testing.T) {
	dsn := os.Getenv("NIE_TEST_DATABASE_URL")
	if dsn == "" || os.Getenv("NIE_OCR_INTEGRATION") != "true" {
		t.Skip("defina NIE_TEST_DATABASE_URL e NIE_OCR_INTEGRATION=true")
	}
	ctx, cancel := context.WithTimeout(context.Background(), 45*time.Second)
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
	cas, err := storage.New(root)
	if err != nil {
		t.Fatal(err)
	}
	raw, err := cas.Put(testfixture.ScannedPDF)
	if err != nil {
		t.Fatal(err)
	}
	camp, err := st.CreateCampaign(ctx, "ocr reprocess", nil, domain.Budget{MaxURLs: 5, MaxDomains: 2, MaxBytes: 1 << 20})
	if err != nil {
		t.Fatal(err)
	}
	defer st.Pool().Exec(context.Background(), `DELETE FROM campaigns WHERE id=$1::uuid`, camp.ID)
	doc, _, err := st.RecordDocument(ctx, store.DocumentRecord{SHA256: raw.SHA256, MIME: "application/pdf", Size: raw.Size, CampaignID: camp.ID, URL: "https://example.org/scan.pdf", NormalizedURL: "https://example.org/scan.pdf", Domain: "example.org", Metadata: map[string]any{"extraction_status": "pending"}})
	if err != nil {
		t.Fatal(err)
	}
	defer st.Pool().Exec(context.Background(), `DELETE FROM documents WHERE id=$1::uuid`, doc)
	handler := New(st, config.Config{DataDir: root, OCREnabled: true, OCRLanguage: "eng", OCRMaxPages: 2}, discardLogger())
	request := func(query string) *httptest.ResponseRecorder {
		r := httptest.NewRequest("POST", "/api/documents/"+doc+"/reprocess"+query, nil)
		w := httptest.NewRecorder()
		handler.ServeHTTP(w, r)
		return w
	}
	if w := request(""); w.Code != 422 {
		t.Fatalf("extração nativa de PDF sem texto: %d %s", w.Code, w.Body.String())
	}
	first := request("?parser=ocr")
	if first.Code != 200 {
		t.Fatalf("OCR: %d %s", first.Code, first.Body.String())
	}
	var run domain.ExtractionRun
	if err := json.Unmarshal(first.Body.Bytes(), &run); err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(run.Text, "BGP NETWORK ENGINE") || run.InputSHA256 == nil || *run.InputSHA256 != raw.SHA256 || run.Passages[0].Page == nil {
		t.Fatalf("OCR sem proveniência: %+v", run)
	}
	if w := request("?parser=ocr"); w.Code != 200 {
		t.Fatalf("repetir OCR: %d %s", w.Code, w.Body.String())
	}
	runs, err := st.ListExtractions(ctx, doc)
	if err != nil || len(runs) != 1 {
		t.Fatalf("OCR duplicou extração: %+v %v", runs, err)
	}
}
