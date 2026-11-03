package backup

import (
	"context"
	"os"
	"os/exec"
	"path/filepath"
	"testing"
	"time"

	"nie/internal/domain"
	"nie/internal/storage"
	"nie/internal/store"
)

func TestResticBackupRestoreIntegration(t *testing.T) {
	sourceDSN, targetDSN := os.Getenv("NIE_BACKUP_TEST_DATABASE_URL"), os.Getenv("NIE_RESTORE_TEST_DATABASE_URL")
	if sourceDSN == "" || targetDSN == "" {
		t.Skip("defina NIE_BACKUP_TEST_DATABASE_URL e NIE_RESTORE_TEST_DATABASE_URL (bancos isolados)")
	}
	for _, tool := range []string{"restic", "pg_dump", "pg_restore"} {
		if _, err := exec.LookPath(tool); err != nil {
			t.Fatalf("ferramenta de validação ausente: %s", tool)
		}
	}
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Minute)
	defer cancel()
	source, err := store.New(ctx, sourceDSN)
	if err != nil {
		t.Fatal(err)
	}
	defer source.Close()
	if err := source.Migrate(ctx); err != nil {
		t.Fatal(err)
	}
	root := t.TempDir()
	repo := filepath.Join(t.TempDir(), "repository")
	password := filepath.Join(t.TempDir(), "password")
	if err := os.WriteFile(password, []byte("isolated-backup-integration-password"), 0600); err != nil {
		t.Fatal(err)
	}
	t.Setenv("RESTIC_REPOSITORY", repo)
	t.Setenv("RESTIC_PASSWORD_FILE", password)
	if _, err := run(exec.CommandContext(ctx, "restic", "init")); err != nil {
		t.Fatal(err)
	}
	objects, err := storage.New(root)
	if err != nil {
		t.Fatal(err)
	}
	raw, err := objects.Put([]byte("historical original support"))
	if err != nil {
		t.Fatal(err)
	}
	rendered, err := objects.Put([]byte("rendered supported text"))
	if err != nil {
		t.Fatal(err)
	}
	camp, err := source.CreateCampaign(ctx, "restore evidence", nil, domain.Budget{MaxURLs: 10, MaxDomains: 3, MaxBytes: 1 << 20})
	if err != nil {
		t.Fatal(err)
	}
	doc, _, err := source.RecordDocument(ctx, store.DocumentRecord{SHA256: raw.SHA256, MIME: "text/plain", Size: raw.Size, Text: "historical original support", Parser: "fixture", ParserVersion: "1", CampaignID: camp.ID, URL: "https://example.org/manual", NormalizedURL: "https://example.org/manual", Domain: "example.org", Passages: []domain.Passage{{Ordinal: 0, Text: "historical original support"}}})
	if err != nil {
		t.Fatal(err)
	}
	original, err := source.GetDocument(ctx, doc)
	if err != nil {
		t.Fatal(err)
	}
	oldPassage := original.Passages[0].ID
	claim := domain.Claim{CampaignID: camp.ID, Title: "restored claim", Statement: "original conclusion", Status: "contested", Applicability: "indeterminate", OperationalValidation: "untested", Evidences: []domain.Evidence{{ID: "support", DocumentID: doc, PassageID: &oldPassage, Stance: "supports", Excerpt: "historical original support"}, {ID: "refute", DocumentID: doc, Stance: "refutes", Excerpt: "conflicting original"}}, Contradictions: []domain.Contradiction{{EvidenceAID: "support", EvidenceBID: "refute", Explanation: "preserved conflict"}}}
	if err := source.SaveClaims(ctx, []domain.Claim{claim}); err != nil {
		t.Fatal(err)
	}
	if _, err := source.RecordExtraction(ctx, doc, store.ExtractionRecord{Parser: "fixture-rendered", ParserVersion: "2", InputSHA256: rendered.SHA256, Text: "rendered supported text", Passages: []domain.Passage{{Ordinal: 0, Text: "rendered supported text"}}}); err != nil {
		t.Fatal(err)
	}
	if err := source.RecordDiscoveryEdge(ctx, camp.ID, "https://example.org/", "https://example.org/manual", "link", "preserved path", 1); err != nil {
		t.Fatal(err)
	}
	configFile := filepath.Join(t.TempDir(), "settings.yml")
	if err := os.WriteFile(configFile, []byte("configured: true\n"), 0600); err != nil {
		t.Fatal(err)
	}
	id, err := Create(ctx, Options{DatabaseURL: sourceDSN, DataDir: root, ConfigFiles: []string{configFile}})
	if err != nil {
		t.Fatal(err)
	}
	// Changes after the exported snapshot must not affect the restored corpus.
	if _, err := source.Pool().Exec(ctx, `UPDATE claims SET statement='later change' WHERE campaign_id=$1::uuid`, camp.ID); err != nil {
		t.Fatal(err)
	}
	targetRoot := filepath.Join(t.TempDir(), "restored")
	manifest, err := Restore(ctx, Options{DatabaseURL: targetDSN, DataDir: targetRoot}, id)
	if err != nil {
		t.Fatal(err)
	}
	if len(manifest.Tables) < 10 {
		t.Fatalf("manifesto incompleto: %+v", manifest)
	}
	restored, err := store.New(ctx, targetDSN)
	if err != nil {
		t.Fatal(err)
	}
	defer restored.Close()
	restoredDoc, err := restored.GetDocument(ctx, doc)
	if err != nil || restoredDoc.Text != "rendered supported text" {
		t.Fatalf("documento restaurado: %+v %v", restoredDoc, err)
	}
	runs, err := restored.ListExtractions(ctx, doc)
	if err != nil || len(runs) != 2 {
		t.Fatalf("extrações: %+v %v", runs, err)
	}
	claims, err := restored.ListClaimsByCampaign(ctx, camp.ID)
	if err != nil || len(claims) != 1 || claims[0].Statement != "original conclusion" || len(claims[0].Contradictions) != 1 {
		t.Fatalf("claims: %+v %v", claims, err)
	}
	revisions, err := restored.ListClaimRevisions(ctx, camp.ID, 100)
	if err != nil || len(revisions) == 0 {
		t.Fatalf("revisões: %+v %v", revisions, err)
	}
	cas, err := storage.New(targetRoot)
	if err != nil {
		t.Fatal(err)
	}
	for _, hash := range []string{raw.SHA256, rendered.SHA256} {
		f, err := cas.OpenVerified(hash)
		if err != nil {
			t.Fatal(err)
		}
		f.Close()
	}
	if _, err := os.Stat(filepath.Join(targetRoot, "recovered-config", "000-settings.yml")); err != nil {
		t.Fatal(err)
	}
	if _, err := Restore(ctx, Options{DatabaseURL: targetDSN, DataDir: filepath.Join(t.TempDir(), "another")}, id); err == nil {
		t.Fatal("restauração sobrescreveu banco preenchido")
	}
	if _, err := run(exec.CommandContext(ctx, "restic", "check", "--read-data")); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(raw.Path, []byte("corrupted"), 0600); err != nil {
		t.Fatal(err)
	}
	if _, err := Create(ctx, Options{DatabaseURL: sourceDSN, DataDir: root}); err == nil {
		t.Fatal("backup aceitou original corrompido")
	}
}
