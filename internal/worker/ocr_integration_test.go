package worker

import (
	"context"
	"io"
	"log/slog"
	"net/url"
	"nie/internal/config"
	"nie/internal/crawler"
	"nie/internal/domain"
	"nie/internal/storage"
	"nie/internal/store"
	"nie/internal/testfixture"
	"os"
	"strings"
	"testing"
	"time"
)

type fixtureRobots struct{}

func (fixtureRobots) Allowed(context.Context, *url.URL) bool { return true }

type scannedExecutor struct{}

func (scannedExecutor) Name() string { return "native" }
func (scannedExecutor) Execute(_ context.Context, task domain.FetchTask) (*domain.FetchResult, error) {
	return &domain.FetchResult{EffectiveURL: task.URL, StatusCode: 200, ContentType: "application/pdf", RawContent: testfixture.ScannedPDF, Metadata: map[string]any{"executor": "controlled-fixture"}}, nil
}

// Real OCR and PostgreSQL, with collection supplied by a controlled executor.
func TestWorkerScannedPDFPersistenceIntegration(t *testing.T) {
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
	camp, err := st.CreateCampaign(ctx, "scanned document", []string{"https://example.org/scanned.pdf"}, domain.Budget{MaxDepth: 1, MaxURLs: 5, MaxDomains: 2, MaxBytes: 10 << 20, MinDomainIntervalSeconds: 1})
	if err != nil {
		t.Fatal(err)
	}
	defer st.Pool().Exec(context.Background(), `DELETE FROM campaigns WHERE id=$1::uuid`, camp.ID)
	cas, err := storage.New(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	router := crawler.NewRouter()
	router.Register(scannedExecutor{})
	worker := &Worker{store: st, storage: cas, router: router, robots: fixtureRobots{}, cfg: config.Config{OCREnabled: true, OCRMaxPages: 2, OCRLanguage: "eng", MaxPDFBytes: 5 << 20, MaxHTMLBytes: 1 << 20, RequestTimeout: 30 * time.Second}, logger: slog.New(slog.NewTextHandler(io.Discard, nil))}
	job, err := st.ClaimJob(ctx, 2*time.Minute)
	if err != nil || job == nil {
		t.Fatalf("job=%+v %v", job, err)
	}
	if err := worker.processFetch(ctx, job); err != nil {
		t.Fatal(err)
	}
	docs, err := st.ListDocuments(ctx, 10, camp.ID)
	if err != nil || len(docs) != 1 {
		t.Fatalf("docs=%+v %v", docs, err)
	}
	defer st.Pool().Exec(context.Background(), `DELETE FROM documents WHERE id=$1::uuid`, docs[0].ID)
	doc, err := st.GetDocument(ctx, docs[0].ID)
	if err != nil || !strings.Contains(doc.Text, "BGP NETWORK ENGINE") {
		t.Fatalf("OCR não entrou no acervo: %+v %v", doc, err)
	}
	run, err := st.GetExtraction(ctx, doc.ID, *doc.ActiveExtractionID)
	if err != nil || run.Parser != "tesseract-poppler" || run.Passages[0].Page == nil {
		t.Fatalf("proveniência perdida: %+v %v", run, err)
	}
	original, err := cas.OpenVerified(doc.SHA256)
	if err != nil {
		t.Fatal(err)
	}
	original.Close()
}
