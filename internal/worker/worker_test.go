package worker

import (
	"context"
	"crypto/sha256"
	"fmt"
	"io"
	"log/slog"
	"os"
	"strings"
	"testing"
	"time"

	"nie/internal/domain"
	"nie/internal/extract"
	"nie/internal/storage"
	"nie/internal/store"
)

// TestMentionDiscoveryOutsideTelecom garante que a descoberta por menções sem
// link funciona em assuntos fora de redes (critério de arquitetura genérica).
func TestMentionDiscoveryOutsideTelecom(t *testing.T) {
	text := "O estudo (doi 10.1056/NEJMoa2034577) cita a norma ISO 27001:2022, " +
		"a vulnerabilidade CVE-2024-3094 e o preprint arXiv:2303.08774."
	mentions := extract.ExtractMentions(text, 25)

	want := map[string]bool{"10.1056/NEJMoa2034577": false, "ISO 27001:2022": false, "CVE-2024-3094": false, "arXiv:2303.08774": false}
	for _, m := range mentions {
		if _, ok := want[m]; ok {
			want[m] = true
			if !isSearchableMention(m) {
				t.Errorf("menção %q deveria gerar nova consulta", m)
			}
		}
	}
	for m, found := range want {
		if !found {
			t.Errorf("menção %q não extraída; obtidas: %v", m, mentions)
		}
	}
}

func TestComputeFrontierPriorityPrefersQueryTermsAndPenalizesNoise(t *testing.T) {
	q := "metformina efeitos colaterais"
	relevant := computeFrontierPriority("https://example.org/metformina/guia", 1, q)
	neutral := computeFrontierPriority("https://example.org/outra-pagina", 1, q)
	noise := computeFrontierPriority("https://example.org/login", 1, q)
	if !(relevant < neutral && neutral < noise) {
		t.Errorf("prioridade (menor = antes): relevante=%d neutra=%d ruído=%d", relevant, neutral, noise)
	}
}

// Um PDF sem texto extraível (ex.: escaneado) não pode ser perdido: o original
// fica no CAS e o documento é registrado como extraction_pending.
func TestUnextractableDocumentIsPreservedForReprocessing(t *testing.T) {
	dsn := strings.TrimSpace(os.Getenv("NIE_TEST_DATABASE_URL"))
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
	camp, err := st.CreateCampaign(ctx, "pdf escaneado "+fmt.Sprint(time.Now().UnixNano()), []string{"https://seed.example/scan.pdf"},
		domain.Budget{MaxDepth: 1, MaxURLs: 5, MaxDomains: 2, MaxBytes: 10 << 20, MinDomainIntervalSeconds: 1})
	if err != nil {
		t.Fatal(err)
	}
	defer func() {
		_, _ = st.Pool().Exec(context.Background(), `DELETE FROM campaigns WHERE id=$1::uuid`, camp.ID)
	}()

	cas, err := storage.New(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	w := &Worker{store: st, storage: cas, logger: slog.New(slog.NewTextHandler(io.Discard, nil))}

	raw := []byte("%PDF-1.4\n% pdf sem camada de texto " + fmt.Sprint(time.Now().UnixNano()) + "\n%%EOF")
	fr := &domain.FetchResult{ContentType: "application/pdf", StatusCode: 200, Metadata: map[string]any{"executor": "native"}}
	job := &domain.Job{CampaignID: camp.ID}
	if err := w.preserveUnextracted(ctx, job, "https://seed.example/scan.pdf", "https://seed.example/scan.pdf", raw, fr, "sem texto"); err != nil {
		t.Fatalf("preserveUnextracted: %v", err)
	}

	sum := fmt.Sprintf("%x", sha256.Sum256(raw))
	if !cas.Has(sum) {
		t.Fatalf("o original deve estar no CAS com SHA-256 %s", sum)
	}
	var status, mime string
	var size int64
	if err := st.Pool().QueryRow(ctx, `SELECT metadata->>'extraction_status', mime, size FROM documents WHERE sha256=$1`, sum).
		Scan(&status, &mime, &size); err != nil {
		t.Fatalf("documento não registrado: %v", err)
	}
	if status != "pending" || mime != "application/pdf" || size != int64(len(raw)) {
		t.Errorf("documento: status=%q mime=%q size=%d", status, mime, size)
	}
	_, _ = st.Pool().Exec(ctx, `DELETE FROM documents WHERE sha256=$1`, sum)
}
