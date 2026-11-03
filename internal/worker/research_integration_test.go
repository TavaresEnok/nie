package worker

import (
	"context"
	"encoding/json"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"nie/internal/config"
	"nie/internal/discovery"
	"nie/internal/domain"
	"nie/internal/store"
	"os"
	"strings"
	"testing"
	"time"
)

// Este ensaio usa banco real e fonte de descoberta HTTP controlada. A coleta
// pública, o browser e a qualidade de fontes externas exigem ensaio separado.
func TestResearchCycleIntegration(t *testing.T) {
	dsn := os.Getenv("NIE_TEST_DATABASE_URL")
	if dsn == "" {
		t.Skip("NIE_TEST_DATABASE_URL não definido")
	}
	ctx, cancel := context.WithTimeout(context.Background(), 45*time.Second)
	defer cancel()
	st, err := store.New(ctx, dsn)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(st.Close)
	if err := st.Migrate(ctx); err != nil {
		t.Fatal(err)
	}
	camp, err := st.CreateCampaign(ctx, "Huawei NE8000 VRP v8 BGP", nil, domain.Budget{MaxURLs: 20, MaxDomains: 5, MaxBytes: 1 << 20})
	if err != nil {
		t.Fatal(err)
	}
	defer st.Pool().Exec(context.Background(), `DELETE FROM campaigns WHERE id=$1::uuid`, camp.ID)
	var received string
	source := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		received = r.URL.Query().Get("q")
		json.NewEncoder(w).Encode(map[string]any{"results": []map[string]string{{"url": "https://support.huawei.com/research-guide", "title": "Guide"}}})
	}))
	defer source.Close()
	searx := discovery.NewSearXNG(source.URL)
	worker := New(st, nil, nil, nil, searx, nil, config.Config{AutomaticResearch: true, MaxResearchRounds: 3}, slog.New(slog.NewTextHandler(io.Discard, nil)))
	addDocument := func(hash, text, url string) {
		t.Helper()
		doc, _, err := st.RecordDocument(ctx, store.DocumentRecord{SHA256: hash, MIME: "text/plain", Text: text, Size: int64(len(text)), CampaignID: camp.ID, URL: url, NormalizedURL: url, Domain: "support.huawei.com", StatusCode: 200, Passages: []domain.Passage{{Ordinal: 0, Text: text}}})
		if err != nil {
			t.Fatal(err)
		}
		t.Cleanup(func() { st.Pool().Exec(context.Background(), `DELETE FROM documents WHERE id=$1::uuid`, doc) })
	}
	addDocument(strings.Repeat("a", 64), "Huawei supports BGP.", "https://support.huawei.com/overview")
	if _, err := st.EnqueueDiscovery(ctx, camp.ID, "discover_web", camp.Query); err != nil {
		t.Fatal(err)
	}
	initial, err := st.ClaimJob(ctx, time.Minute)
	if err != nil || initial == nil {
		t.Fatalf("initial=%+v err=%v", initial, err)
	}
	if err := st.CompleteJob(ctx, initial.ID); err != nil {
		t.Fatal(err)
	}
	next := func() *domain.Job {
		t.Helper()
		if err := st.FinishIfIdle(ctx, camp.ID); err != nil {
			t.Fatal(err)
		}
		job, err := st.ClaimJob(ctx, time.Minute)
		if err != nil || job == nil {
			t.Fatalf("next=%+v err=%v", job, err)
		}
		return job
	}
	evaluation := next()
	if evaluation.Kind != "investigate" {
		t.Fatalf("kind=%s", evaluation.Kind)
	}
	if err := worker.process(ctx, evaluation); err != nil {
		t.Fatal(err)
	}
	if err := st.CompleteJob(ctx, evaluation.ID); err != nil {
		t.Fatal(err)
	}
	search := next()
	if search.Kind != "discover_web" {
		t.Fatalf("kind=%s", search.Kind)
	}
	if err := worker.process(ctx, search); err != nil {
		t.Fatal(err)
	}
	if received == camp.Query || !strings.Contains(received, "release") {
		t.Fatalf("busca não derivou da lacuna de versão: %q", received)
	}
	if err := st.CompleteJob(ctx, search.ID); err != nil {
		t.Fatal(err)
	}
	fetched := next()
	if fetched.Kind != "fetch" {
		t.Fatalf("resultado HTTP não enfileirou coleta: %+v", fetched)
	}
	addDocument(strings.Repeat("b", 64), "Huawei NE8000 VRP v8 supports BGP.", fetched.URL)
	if err := st.CompleteJob(ctx, fetched.ID); err != nil {
		t.Fatal(err)
	}
	second := next()
	if second.Kind != "investigate" {
		t.Fatalf("segunda avaliação: %+v", second)
	}
	if err := worker.process(ctx, second); err != nil {
		t.Fatal(err)
	}
	if err := st.CompleteJob(ctx, second.ID); err != nil {
		t.Fatal(err)
	}
	if err := st.FinishIfIdle(ctx, camp.ID); err != nil {
		t.Fatal(err)
	}
	rounds, err := st.ListResearchRounds(ctx, camp.ID)
	if err != nil || len(rounds) != 2 {
		t.Fatalf("rounds=%+v err=%v", rounds, err)
	}
	if rounds[0].Decision != "continue" || rounds[0].Reason != "evidence_gaps" {
		t.Fatalf("a primeira rodada deveria seguir a lacuna de versão: %+v", rounds[0])
	}
	if rounds[1].Decision != "continue" || rounds[1].Reason != "open_subquestions" || rounds[1].EnqueuedQueries < 1 {
		t.Fatalf("evidência com escopo não abriu as subperguntas: %+v", rounds[1])
	}
	questions, err := st.ListResearchQuestions(ctx, camp.ID)
	if err != nil || len(questions) < 4 {
		t.Fatalf("questions=%+v err=%v", questions, err)
	}
	queued := 0
	for _, question := range questions {
		if question.Status == "queued" {
			queued++
		}
	}
	if queued == 0 {
		t.Fatalf("nenhuma subpergunta foi enfileirada: %+v", questions)
	}
}
