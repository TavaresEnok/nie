package store

import (
	"context"
	"fmt"
	"os"
	"sync"
	"testing"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
	"nie/internal/domain"
)

func TestVersionedExtractionsIntegration(t *testing.T) {
	dsn := os.Getenv("NIE_TEST_DATABASE_URL")
	if dsn == "" {
		t.Skip("NIE_TEST_DATABASE_URL não definido")
	}
	ctx, cancel := context.WithTimeout(context.Background(), 45*time.Second)
	defer cancel()
	st, err := New(ctx, dsn)
	if err != nil {
		t.Fatal(err)
	}
	defer st.Close()
	if err := st.Migrate(ctx); err != nil {
		t.Fatal(err)
	}
	camp, err := st.CreateCampaign(ctx, "versioned original", nil, domain.Budget{MaxURLs: 10, MaxBytes: 1 << 20, MaxDomains: 5})
	if err != nil {
		t.Fatal(err)
	}
	defer st.pool.Exec(context.Background(), `DELETE FROM campaigns WHERE id=$1::uuid`, camp.ID)
	rec := DocumentRecord{SHA256: hashFor(fmt.Sprint(time.Now().UnixNano())), MIME: "text/plain", Size: 40, Text: "old command alpha", Parser: "fixture", ParserVersion: "1", CampaignID: camp.ID, URL: "https://example.org/manual", NormalizedURL: "https://example.org/manual", Domain: "example.org", Passages: []domain.Passage{{Ordinal: 0, Text: "old command alpha"}}}
	docID, _, err := st.RecordDocument(ctx, rec)
	if err != nil {
		t.Fatal(err)
	}
	defer st.pool.Exec(context.Background(), `DELETE FROM documents WHERE id=$1::uuid`, docID)
	original, err := st.GetDocument(ctx, docID)
	if err != nil || len(original.Passages) != 1 {
		t.Fatalf("original=%+v err=%v", original, err)
	}
	oldID := original.Passages[0].ID
	if err := st.SaveClaims(ctx, []domain.Claim{{CampaignID: camp.ID, Title: "historical citation", Statement: "alpha", Status: "unconfirmed", Applicability: "indeterminate", OperationalValidation: "untested", Evidences: []domain.Evidence{{DocumentID: docID, PassageID: &oldID, Stance: "mentions", Excerpt: rec.Text}}}}); err != nil {
		t.Fatal(err)
	}
	next := ExtractionRecord{Parser: "fixture", ParserVersion: "2", Text: "new command beta", Passages: []domain.Passage{{Ordinal: 0, Text: "new command beta"}}}
	runID, err := st.RecordExtraction(ctx, docID, next)
	if err != nil {
		t.Fatal(err)
	}
	again, err := st.RecordExtraction(ctx, docID, next)
	if err != nil || again != runID {
		t.Fatalf("extração não idempotente: %s %s %v", runID, again, err)
	}
	// A later fetch from the first parser must not downgrade the active run.
	if _, _, err := st.RecordDocument(ctx, rec); err != nil {
		t.Fatal(err)
	}
	active, err := st.GetDocument(ctx, docID)
	if err != nil || active.ActiveExtractionID == nil || *active.ActiveExtractionID != runID {
		t.Fatalf("projeção ativa errada: %+v %v", active, err)
	}
	oldRun, err := st.GetExtraction(ctx, docID, *original.ActiveExtractionID)
	if err != nil || oldRun.Passages[0].ID != oldID || oldRun.Text != rec.Text {
		t.Fatalf("extração histórica alterada: %+v %v", oldRun, err)
	}
	claims, err := st.ListClaimsByCampaign(ctx, camp.ID)
	if err != nil || len(claims) != 1 || claims[0].Evidences[0].PassageID == nil || *claims[0].Evidences[0].PassageID != oldID {
		t.Fatalf("citação perdida: %+v %v", claims, err)
	}
	results, err := st.Search(ctx, "command", camp.ID, 20)
	if err != nil || len(results) != 1 || results[0].ExtractionRunID != runID {
		t.Fatalf("busca misturou versões: %+v %v", results, err)
	}
	if active.Passages[0].StartOffset == nil || active.Passages[0].EndOffset == nil || active.Text[*active.Passages[0].StartOffset:*active.Passages[0].EndOffset] != active.Passages[0].Text {
		t.Fatal("offset não corresponde ao texto")
	}
	next.ParserVersion = "3"
	var wg sync.WaitGroup
	errs := make(chan error, 8)
	for i := 0; i < 8; i++ {
		wg.Add(1)
		go func() { defer wg.Done(); _, err := st.RecordExtraction(ctx, docID, next); errs <- err }()
	}
	wg.Wait()
	close(errs)
	for err := range errs {
		if err != nil {
			t.Fatal(err)
		}
	}
	runs, err := st.ListExtractions(ctx, docID)
	if err != nil || len(runs) != 3 {
		t.Fatalf("corrida duplicou extrações: %+v %v", runs, err)
	}
}

func TestLegacyExtractionsMigrationIntegration(t *testing.T) {
	dsn := os.Getenv("NIE_TEST_DATABASE_URL")
	if dsn == "" {
		t.Skip("NIE_TEST_DATABASE_URL não definido")
	}
	ctx, cancel := context.WithTimeout(context.Background(), 45*time.Second)
	defer cancel()
	base, err := New(ctx, dsn)
	if err != nil {
		t.Fatal(err)
	}
	defer base.Close()
	schema := fmt.Sprintf("legacy_%d", time.Now().UnixNano())
	if _, err := base.pool.Exec(ctx, `CREATE SCHEMA `+schema); err != nil {
		t.Fatal(err)
	}
	defer base.pool.Exec(context.Background(), `DROP SCHEMA `+schema+` CASCADE`)
	cfg, err := pgxpool.ParseConfig(dsn)
	if err != nil {
		t.Fatal(err)
	}
	cfg.ConnConfig.RuntimeParams["search_path"] = schema + ",public"
	pool, err := pgxpool.NewWithConfig(ctx, cfg)
	if err != nil {
		t.Fatal(err)
	}
	defer pool.Close()
	st := &Store{pool: pool}
	// Seed an existing installation before migration 0009, including a passage
	// ID already used as a citation. No legacy parser version is invented.
	if _, err := pool.Exec(ctx, `CREATE TABLE documents(id uuid PRIMARY KEY,sha256 text,mime text,title text,size bigint,text text,metadata jsonb,created_at timestamptz DEFAULT now()); CREATE TABLE passages(id bigserial PRIMARY KEY,document_id uuid REFERENCES documents(id) ON DELETE CASCADE,ordinal integer,page integer,text text)`); err != nil {
		t.Fatal(err)
	}
	var docID string
	if err := pool.QueryRow(ctx, `INSERT INTO documents VALUES(gen_random_uuid(),'legacy','text/plain','title',3,'old','{}',now()) RETURNING id::text`).Scan(&docID); err != nil {
		t.Fatal(err)
	}
	var passageID int64
	if err := pool.QueryRow(ctx, `INSERT INTO passages(document_id,ordinal,text) VALUES($1::uuid,0,'old') RETURNING id`, docID).Scan(&passageID); err != nil {
		t.Fatal(err)
	}
	migration, err := migrationsFS.ReadFile("migrations/0009_versioned_extractions.sql")
	if err != nil {
		t.Fatal(err)
	}
	tx, err := pool.Begin(ctx)
	if err != nil {
		t.Fatal(err)
	}
	defer tx.Rollback(ctx)
	for _, stmt := range splitStatements(string(migration)) {
		if _, err := tx.Exec(ctx, stmt); err != nil {
			t.Fatal(err)
		}
	}
	if err := tx.Commit(ctx); err != nil {
		t.Fatal(err)
	}
	runs, err := st.ListExtractions(ctx, docID)
	if err != nil || len(runs) != 1 || runs[0].Parser != "legacy-unknown" {
		t.Fatalf("legacy=%+v %v", runs, err)
	}
	run, err := st.GetExtraction(ctx, docID, runs[0].ID)
	if err != nil || len(run.Passages) != 1 || run.Passages[0].ID != passageID {
		t.Fatalf("ID histórico perdido: %+v %v", run, err)
	}
	if run.Passages[0].StartOffset != nil {
		t.Fatal("migração inventou offset ausente")
	}
}
