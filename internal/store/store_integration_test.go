package store

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"os"
	"strings"
	"testing"
	"time"

	"nie/internal/domain"
)

// TestStoreIntegration valida migrações, campanha, documento, busca e fila
// contra um PostgreSQL real. Execute com NIE_TEST_DATABASE_URL definido.
func TestStoreIntegration(t *testing.T) {
	dsn := strings.TrimSpace(os.Getenv("NIE_TEST_DATABASE_URL"))
	if dsn == "" {
		t.Skip("NIE_TEST_DATABASE_URL não definido")
	}
	ctx, cancel := context.WithTimeout(context.Background(), 45*time.Second)
	defer cancel()

	st, err := New(ctx, dsn)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(st.Close)
	if err := st.Migrate(ctx); err != nil {
		t.Fatal(err)
	}

	suffix := fmt.Sprintf("%d", time.Now().UnixNano())
	budget := domain.Budget{
		MaxDepth:                 3,
		MaxURLs:                  10,
		MaxDomains:               5,
		MaxBytes:                 10 << 20,
		MinDomainIntervalSeconds: 1,
	}
	campaign, err := st.CreateCampaign(ctx, "teste de integração "+suffix, []string{"https://example.org/doc?utm_source=test"}, budget)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		cleanupCtx, cleanupCancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cleanupCancel()
		_, _ = st.Pool().Exec(cleanupCtx, `DELETE FROM campaigns WHERE id=$1::uuid`, campaign.ID)
		_, _ = st.Pool().Exec(cleanupCtx, `DELETE FROM documents WHERE sha256=$1 OR sha256=$2`, hashFor("one-"+suffix), hashFor("two-"+suffix))
		_, _ = st.Pool().Exec(cleanupCtx, `DELETE FROM domain_state WHERE domain=$1`, "throttle-"+suffix+".example")
	})
	if len(campaign.SeedURLs) != 1 || campaign.SeedURLs[0] != "https://example.org/doc" {
		t.Fatalf("seeds normalizadas = %v", campaign.SeedURLs)
	}
	if campaign.Stats.JobsPending != 1 {
		t.Errorf("jobs pendentes = %d, quer 1", campaign.Stats.JobsPending)
	}
	if campaign.Stats.Passages != 0 {
		t.Errorf("trechos iniciais = %d, quer 0", campaign.Stats.Passages)
	}

	doc1, created, err := st.RecordDocument(ctx, DocumentRecord{
		SHA256:        hashFor("one-" + suffix),
		MIME:          "text/plain",
		Title:         "BGP Guide",
		Size:          128,
		Text:          "Use display bgp peer para verificar as sessões. Rota: ip route 10.0.0.0/8.",
		Metadata:      map[string]any{"source": "test"},
		CampaignID:    campaign.ID,
		URL:           "https://example.org/doc",
		NormalizedURL: "https://example.org/doc",
		Domain:        "example.org",
		StatusCode:    200,
		Passages: []domain.Passage{
			{Ordinal: 0, Text: "Use display bgp peer para verificar as sessões. Rota: ip route 10.0.0.0/8."},
		},
	})
	if err != nil {
		t.Fatal(err)
	}
	if !created {
		t.Error("primeiro documento deveria ser criado")
	}

	doc2, created, err := st.RecordDocument(ctx, DocumentRecord{
		SHA256:        hashFor("two-" + suffix),
		MIME:          "text/plain",
		Title:         "Sessões",
		Size:          32,
		Text:          "verificar sessões bgp",
		CampaignID:    campaign.ID,
		URL:           "https://example.org/outro",
		NormalizedURL: "https://example.org/outro",
		Domain:        "example.org",
		StatusCode:    200,
		Passages: []domain.Passage{
			{Ordinal: 0, Text: "verificar sessões bgp"},
		},
	})
	if err != nil {
		t.Fatal(err)
	}
	if !created {
		t.Error("segundo documento deveria ser criado")
	}

	exact, err := st.Search(ctx, "display bgp peer", "", 20)
	if err != nil {
		t.Fatal(err)
	}
	if !hasResult(exact, doc1, "exact") {
		t.Fatalf("busca exata não encontrou o documento: %+v", exact)
	}
	if !strings.Contains(exact[0].Snippet, "<mark>") {
		t.Errorf("snippet sem destaque: %q", exact[0].Snippet)
	}
	route, err := st.Search(ctx, "ip route 10.0.0.0/8", "", 20)
	if err != nil {
		t.Fatal(err)
	}
	if !hasResult(route, doc1, "exact") {
		t.Fatalf("busca exata com símbolos não encontrou o documento: %+v", route)
	}

	fts, err := st.Search(ctx, "display sessões", "", 20)
	if err != nil {
		t.Fatal(err)
	}
	if !hasResult(fts, doc1, "fts") {
		t.Fatalf("busca FTS não encontrou documento com os dois tokens: %+v", fts)
	}

	fuzzy, err := st.Search(ctx, "verifcar sessoe", "", 20)
	if err != nil {
		t.Fatal(err)
	}
	if !hasResult(fuzzy, doc2, "fuzzy") {
		t.Fatalf("busca aproximada não encontrou o documento: %+v", fuzzy)
	}

	detail, err := st.GetDocument(ctx, doc1)
	if err != nil {
		t.Fatal(err)
	}
	if len(detail.Passages) != 1 || detail.Passages[0].Text == "" {
		t.Fatalf("trechos do documento = %+v", detail.Passages)
	}
	if len(detail.Versions) != 1 || detail.Versions[0].CampaignID != campaign.ID {
		t.Fatalf("versões do documento = %+v", detail.Versions)
	}
	docs, err := st.ListDocuments(ctx, 20, campaign.ID)
	if err != nil {
		t.Fatal(err)
	}
	if len(docs) != 2 {
		t.Errorf("documentos da campanha = %d, quer 2", len(docs))
	}

	fresh, err := st.GetCampaign(ctx, campaign.ID)
	if err != nil {
		t.Fatal(err)
	}
	if fresh.Stats.Documents != 2 || fresh.Stats.Passages != 2 {
		t.Errorf("estatísticas da campanha = %+v", fresh.Stats)
	}

	ok, err := st.EnqueueFetch(ctx, campaign.ID, "https://example.org/outra?utm_campaign=x", 1, "https://example.org/doc", "link", 100)
	if err != nil {
		t.Fatal(err)
	}
	if !ok {
		t.Fatal("nova URL deveria ser enfileirada")
	}
	ok, err = st.EnqueueFetch(ctx, campaign.ID, "https://example.org/outra", 2, "https://example.org/doc", "link", 100)
	if err != nil {
		t.Fatal(err)
	}
	if ok {
		t.Error("URL duplicada não deveria ser enfileirada")
	}
	jobs, err := st.ListJobs(ctx, campaign.ID, "", 50)
	if err != nil {
		t.Fatal(err)
	}
	if len(jobs) < 2 {
		t.Errorf("jobs da campanha = %d, quer >= 2", len(jobs))
	}
	edges, err := st.ListEdges(ctx, campaign.ID, 50)
	if err != nil {
		t.Fatal(err)
	}
	if len(edges) == 0 {
		t.Error("aresta de descoberta não registrada")
	}

	if err := st.SetCampaignStatus(ctx, campaign.ID, "paused"); err != nil {
		t.Fatal(err)
	}
	paused, err := st.GetCampaign(ctx, campaign.ID)
	if err != nil {
		t.Fatal(err)
	}
	if paused.Status != "paused" {
		t.Errorf("status = %q, quer paused", paused.Status)
	}
	if err := st.SetCampaignStatus(ctx, campaign.ID, "running"); err != nil {
		t.Fatal(err)
	}

	// Jobs running com lease expirado devem voltar a ser elegíveis.
	if err := st.SetCampaignStatus(ctx, campaign.ID, "paused"); err != nil {
		t.Fatal(err)
	}
	leaseCampaign, err := st.CreateCampaign(ctx, "teste de lease "+suffix, []string{"https://lease.example/doc"}, domain.Budget{
		MaxDepth: 1, MaxURLs: 1, MaxDomains: 1, MaxBytes: 1 << 20, MinDomainIntervalSeconds: 1,
	})
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		cleanupCtx, cleanupCancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cleanupCancel()
		_, _ = st.Pool().Exec(cleanupCtx, `DELETE FROM campaigns WHERE id=$1::uuid`, leaseCampaign.ID)
	})
	firstClaim, err := st.ClaimJob(ctx, time.Minute)
	if err != nil {
		t.Fatal(err)
	}
	if firstClaim == nil || firstClaim.CampaignID != leaseCampaign.ID {
		t.Fatalf("claim inicial inesperado: %+v", firstClaim)
	}
	if _, err := st.Pool().Exec(ctx, `UPDATE jobs SET lease_until=now()-interval '1 minute' WHERE id=$1`, firstClaim.ID); err != nil {
		t.Fatal(err)
	}
	secondClaim, err := st.ClaimJob(ctx, time.Minute)
	if err != nil {
		t.Fatal(err)
	}
	if secondClaim == nil || secondClaim.ID != firstClaim.ID {
		t.Fatalf("job com lease expirado não foi reivindicado: %+v", secondClaim)
	}
	if secondClaim.Attempts != firstClaim.Attempts+1 {
		t.Errorf("attempts = %d, quer %d", secondClaim.Attempts, firstClaim.Attempts+1)
	}
	if err := st.CompleteJob(ctx, secondClaim.ID); err != nil {
		t.Fatal(err)
	}
	var deadID int64
	if err := st.Pool().QueryRow(ctx, `
		INSERT INTO jobs(campaign_id, kind, status, attempts, max_attempts, lease_until)
		VALUES($1::uuid, 'fetch', 'running', 3, 3, now()-interval '1 minute')
		RETURNING id`, leaseCampaign.ID).Scan(&deadID); err != nil {
		t.Fatal(err)
	}
	if n, err := st.ReapExpiredJobs(ctx); err != nil {
		t.Fatal(err)
	} else if n < 1 {
		t.Error("reaper deveria encerrar ao menos um job")
	}
	var deadStatus string
	if err := st.Pool().QueryRow(ctx, `SELECT status FROM jobs WHERE id=$1`, deadID).Scan(&deadStatus); err != nil {
		t.Fatal(err)
	}
	if deadStatus != "dead" {
		t.Errorf("status do job expirado = %q, quer dead", deadStatus)
	}
	if _, err := st.FinishIdleCampaigns(ctx); err != nil {
		t.Fatal(err)
	}
	leaseFresh, err := st.GetCampaign(ctx, leaseCampaign.ID)
	if err != nil {
		t.Fatal(err)
	}
	if leaseFresh.Status != "completed" {
		t.Errorf("campanha ociosa = %q, quer completed", leaseFresh.Status)
	}

	next, allowed, err := st.Throttle(ctx, "throttle-"+suffix+".example", time.Minute)
	if err != nil {
		t.Fatal(err)
	}
	if !allowed || next.IsZero() {
		t.Fatal("primeiro acesso ao domínio deveria ser permitido")
	}
	_, allowed, err = st.Throttle(ctx, "throttle-"+suffix+".example", time.Minute)
	if err != nil {
		t.Fatal(err)
	}
	if allowed {
		t.Fatal("segundo acesso imediato deveria ser adiado")
	}

	stats, err := st.Stats(ctx)
	if err != nil {
		t.Fatal(err)
	}
	if stats.Campaigns < 1 || stats.Documents < 2 || stats.Passages < 2 {
		t.Errorf("estatísticas globais inesperadas: %+v", stats)
	}
}

func hasResult(items []domain.SearchResult, documentID, match string) bool {
	for _, item := range items {
		if item.DocumentID == documentID && item.Match == match {
			return true
		}
	}
	return false
}

func hashFor(s string) string {
	sum := sha256.Sum256([]byte(s))
	return hex.EncodeToString(sum[:])
}

// TestP103_MultipleOriginsPreserveEdges verifica que se A e B apontam para C:
// Ocorre apenas 1 job de coleta para C, mas 2 arestas distintas no grafo de descoberta.
func TestP103_MultipleOriginsPreserveEdges(t *testing.T) {
	dsn := strings.TrimSpace(os.Getenv("NIE_TEST_DATABASE_URL"))
	if dsn == "" {
		t.Skip("NIE_TEST_DATABASE_URL não definido")
	}
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	st, err := New(ctx, dsn)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(st.Close)

	suffix := fmt.Sprintf("%d", time.Now().UnixNano())
	camp, err := st.CreateCampaign(ctx, "teste P1-03 multi-origem "+suffix, []string{"https://seed.example"}, domain.Budget{
		MaxDepth:   3,
		MaxURLs:    20,
		MaxDomains: 10,
		MaxBytes:   10 << 20,
	})
	if err != nil {
		t.Fatal(err)
	}
	cleanupCampaign(t, st, camp.ID)

	urlA := "https://page-a.example/guide"
	urlB := "https://page-b.example/forum"
	targetC := "https://target-c.example/telecom-firmware.pdf"

	// 1. A descobre C (primeira vez -> job inserido e aresta gravada)
	enq1, err := st.EnqueueFetch(ctx, camp.ID, targetC, 1, urlA, "link", 40)
	if err != nil {
		t.Fatalf("EnqueueFetch A->C falhou: %v", err)
	}
	if !enq1 {
		t.Fatalf("Esperava que primeiro enfileiramento de C retornasse true")
	}

	// 2. B também descobre C (segunda vez -> dedup no download, mas aresta B->C deve ser preservada!)
	enq2, err := st.EnqueueFetch(ctx, camp.ID, targetC, 1, urlB, "link", 40)
	if err != nil {
		t.Fatalf("EnqueueFetch B->C falhou: %v", err)
	}
	if enq2 {
		t.Fatalf("Esperava que segundo enfileiramento de C retornasse false (job deduplicado)")
	}

	// 3. Verifica o grafo: deve conter as duas arestas rastreáveis (A->C e B->C)
	graph, err := st.GetDiscoveryGraph(ctx, camp.ID)
	if err != nil {
		t.Fatalf("GetDiscoveryGraph falhou: %v", err)
	}

	foundAC := false
	foundBC := false
	for _, edge := range graph.Edges {
		if edge.Target == targetC {
			if edge.Source == urlA {
				foundAC = true
			}
			if edge.Source == urlB {
				foundBC = true
			}
		}
	}

	if !foundAC || !foundBC {
		t.Fatalf("Critério de aceite P1-03 não satisfeito: A->C (%v), B->C (%v) em arestas: %+v", foundAC, foundBC, graph.Edges)
	}
}

// TestP102_ConcurrentMigrations verifica inicialização concorrente de duas instâncias
// usando advisory lock com conexão dedicada sem travamento ou migração duplicada.
func TestP102_ConcurrentMigrations(t *testing.T) {
	dsn := strings.TrimSpace(os.Getenv("NIE_TEST_DATABASE_URL"))
	if dsn == "" {
		t.Skip("NIE_TEST_DATABASE_URL não definido")
	}
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	st1, err := New(ctx, dsn)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(st1.Close)

	st2, err := New(ctx, dsn)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(st2.Close)

	errCh := make(chan error, 2)
	go func() {
		errCh <- st1.Migrate(ctx)
	}()
	go func() {
		errCh <- st2.Migrate(ctx)
	}()

	for i := 0; i < 2; i++ {
		if err := <-errCh; err != nil {
			t.Fatalf("Migração concorrente falhou: %v", err)
		}
	}
}

// TestP002_SaveClaimsWithDocumentedAndApplicableSchema valida que o schema Postgres
// suporta status 'documented', aplicabilidade 'applicable' e colunas estruturadas de evidência.
func TestP002_SaveClaimsWithDocumentedAndApplicableSchema(t *testing.T) {
	dsn := strings.TrimSpace(os.Getenv("NIE_TEST_DATABASE_URL"))
	if dsn == "" {
		t.Skip("NIE_TEST_DATABASE_URL não definido")
	}
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	st, err := New(ctx, dsn)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(st.Close)

	if err := st.Migrate(ctx); err != nil {
		t.Fatalf("Migrate falhou ao aplicar migrações incluindo 0005: %v", err)
	}

	budget := domain.Budget{
		MaxDepth:                 1,
		MaxURLs:                  10,
		MaxDomains:               2,
		MaxBytes:                 10 << 20,
		MinDomainIntervalSeconds: 1,
	}
	camp, err := st.CreateCampaign(ctx, "Audit Schema Test", []string{"https://example.com/test"}, budget)
	if err != nil {
		t.Fatalf("CreateCampaign falhou: %v", err)
	}
	cleanupCampaign(t, st, camp.ID)

	// Criar um documento para vincular a evidência
	docID, _, err := st.RecordDocument(ctx, DocumentRecord{
		SHA256:        hashFor("schema-test-" + fmt.Sprintf("%d", time.Now().UnixNano())),
		MIME:          "text/html",
		Title:         "Test Doc",
		Size:          100,
		Text:          "bgp 65000",
		Metadata:      map[string]any{"source": "test"},
		CampaignID:    camp.ID,
		URL:           "https://example.com/test",
		NormalizedURL: "https://example.com/test",
		Domain:        "example.com",
		StatusCode:    200,
		Passages: []domain.Passage{
			{Ordinal: 0, Text: "bgp 65000"},
		},
	})
	if err != nil {
		t.Fatalf("RecordDocument falhou: %v", err)
	}

	claim := domain.Claim{
		CampaignID:            camp.ID,
		Title:                 "Huawei NE8000 CLI Specification",
		Statement:             "Procedimento CLI para BGP e Interfaces",
		Status:                "documented",
		Applicability:         "applicable",
		OperationalValidation: "untested",
		Evidences: []domain.Evidence{
			{
				DocumentID:         docID,
				Stance:             "supports",
				Excerpt:            "bgp 65000",
				Confidence:         0.95,
				IsOfficialDoc:      true,
				Notes:              "Fonte: support.huawei.com",
				VendorMatch:        true,
				ModelMatch:         true,
				FirmwareMatch:      true,
				FeatureMatch:       true,
				ModeMatch:          true,
				PrerequisitesMatch: true,
				SourceSpan:         "doc:1:p:1",
			},
		},
	}

	if err := st.SaveClaims(ctx, []domain.Claim{claim}); err != nil {
		t.Fatalf("SaveClaims falhou com violação de schema P0-02: %v", err)
	}

	retrieved, err := st.ListClaimsByCampaign(ctx, camp.ID)
	if err != nil {
		t.Fatalf("ListClaimsByCampaign falhou: %v", err)
	}
	if len(retrieved) == 0 {
		t.Fatalf("Nenhuma claim recuperada")
	}

	c := retrieved[0]
	if c.Status != "documented" {
		t.Errorf("Esperava status 'documented', obteve %s", c.Status)
	}
	if c.Applicability != "applicable" {
		t.Errorf("Esperava applicability 'applicable', obteve %s", c.Applicability)
	}
	if len(c.Evidences) == 0 {
		t.Fatalf("Nenhuma evidência recuperada")
	}
	ev := c.Evidences[0]
	if !ev.VendorMatch || !ev.ModelMatch || !ev.FirmwareMatch || !ev.FeatureMatch || !ev.ModeMatch || !ev.PrerequisitesMatch {
		t.Errorf("Campos booleanos estruturados de evidência não foram preservados: %+v", ev)
	}
	if ev.SourceSpan != "doc:1:p:1" {
		t.Errorf("SourceSpan não preservado, obteve %s", ev.SourceSpan)
	}
}

// cleanupCampaign remove a campanha (e, em cascata, jobs e arestas) ao fim do
// teste, para que ClaimJob em outros testes não pegue jobs remanescentes.
func cleanupCampaign(t *testing.T, st *Store, id string) {
	t.Helper()
	t.Cleanup(func() {
		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()
		_, _ = st.Pool().Exec(ctx, `DELETE FROM campaigns WHERE id=$1::uuid`, id)
	})
}

// newTestStore abre o banco de testes e aplica as migrações.
func newTestStore(t *testing.T) (*Store, context.Context) {
	t.Helper()
	dsn := strings.TrimSpace(os.Getenv("NIE_TEST_DATABASE_URL"))
	if dsn == "" {
		t.Skip("NIE_TEST_DATABASE_URL não definido")
	}
	ctx, cancel := context.WithTimeout(context.Background(), 45*time.Second)
	t.Cleanup(cancel)
	st, err := New(ctx, dsn)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(st.Close)
	if err := st.Migrate(ctx); err != nil {
		t.Fatal(err)
	}
	return st, ctx
}

// Pistas geradas depois que a campanha terminou precisam ser executadas:
// EnqueueDiscovery reabre a campanha, e a mesma consulta nunca entra duas vezes.
func TestP0D_DiscoveryReopensCompletedCampaignAndDedupes(t *testing.T) {
	st, ctx := newTestStore(t)
	camp, err := st.CreateCampaign(ctx, "reabrir "+fmt.Sprint(time.Now().UnixNano()), []string{"https://seed.example/a"},
		domain.Budget{MaxDepth: 1, MaxURLs: 2, MaxDomains: 2, MaxBytes: 10 << 20, MinDomainIntervalSeconds: 1})
	if err != nil {
		t.Fatal(err)
	}
	cleanupCampaign(t, st, camp.ID)

	// Campanha termina normalmente.
	if _, err := st.Pool().Exec(ctx, `UPDATE jobs SET status='done' WHERE campaign_id=$1::uuid`, camp.ID); err != nil {
		t.Fatal(err)
	}
	if n, err := st.FinishIdleCampaigns(ctx); err != nil || n < 1 {
		t.Fatalf("FinishIdleCampaigns = %d, %v", n, err)
	}
	if got, _ := st.GetCampaign(ctx, camp.ID); got.Status != "completed" {
		t.Fatalf("campanha deveria estar concluída, status=%s", got.Status)
	}

	ok, err := st.EnqueueDiscovery(ctx, camp.ID, "discover_web", "Huawei  NE8000 PPPoE")
	if err != nil || !ok {
		t.Fatalf("primeira pista deveria entrar: ok=%v err=%v", ok, err)
	}
	if got, _ := st.GetCampaign(ctx, camp.ID); got.Status != "running" {
		t.Fatalf("a campanha deveria ser reaberta, status=%s", got.Status)
	}
	job, err := st.ClaimJob(ctx, time.Minute)
	if err != nil || job == nil || job.CampaignID != camp.ID || job.Kind != "discover_web" {
		t.Fatalf("o worker deveria obter a pista da campanha reaberta: %+v err=%v", job, err)
	}

	// Mesma consulta (outra caixa e espaçamento) não gera novo job.
	for _, q := range []string{"huawei ne8000 pppoe", "  HUAWEI NE8000   PPPoE "} {
		if dup, err := st.EnqueueDiscovery(ctx, camp.ID, "discover_web", q); err != nil || dup {
			t.Errorf("consulta repetida %q não deveria entrar: dup=%v err=%v", q, dup, err)
		}
	}
	// Outro provedor com a mesma consulta é uma pesquisa distinta.
	if ok, err := st.EnqueueDiscovery(ctx, camp.ID, "discover_mwmbl", "huawei ne8000 pppoe"); err != nil || !ok {
		t.Errorf("mesmo texto em outro provedor deveria entrar: ok=%v err=%v", ok, err)
	}

	var n int
	if err := st.Pool().QueryRow(ctx, `SELECT count(*) FROM jobs WHERE campaign_id=$1::uuid AND kind='discover_web'`, camp.ID).Scan(&n); err != nil || n != 1 {
		t.Errorf("jobs discover_web = %d (err=%v), quer 1", n, err)
	}
	// Pesquisas externas não consomem o orçamento de URLs (max_urls=2 já tem 1 fetch).
	if ok, err := st.EnqueueFetch(ctx, camp.ID, "https://other.example/b", 1, "https://seed.example/a", "link", 50); err != nil || !ok {
		t.Errorf("fetch dentro do orçamento deveria entrar mesmo após pesquisas: ok=%v err=%v", ok, err)
	}
}

// Reexecutar a investigação não duplica claims, a independência de origens
// sobrevive ao recarregamento e o tipo da aresta de descoberta é preservado.
func TestP1_SaveClaimsIdempotentAndClustersPersist(t *testing.T) {
	st, ctx := newTestStore(t)
	camp, err := st.CreateCampaign(ctx, "claims idempotentes "+fmt.Sprint(time.Now().UnixNano()), []string{"https://seed.example/a"},
		domain.Budget{MaxDepth: 1, MaxURLs: 10, MaxDomains: 5, MaxBytes: 10 << 20, MinDomainIntervalSeconds: 1})
	if err != nil {
		t.Fatal(err)
	}
	cleanupCampaign(t, st, camp.ID)

	mkDoc := func(text, host string) string {
		id, _, err := st.RecordDocument(ctx, DocumentRecord{
			SHA256: hashFor(text + fmt.Sprint(time.Now().UnixNano())), MIME: "text/html", Title: text, Size: 10, Text: text,
			CampaignID: camp.ID, URL: "https://" + host + "/p", NormalizedURL: "https://" + host + "/p", Domain: host, StatusCode: 200,
			Passages: []domain.Passage{{Ordinal: 0, Text: text}},
		})
		if err != nil {
			t.Fatal(err)
		}
		return id
	}
	docA, docB := mkDoc("bgp 65000 peer", "blog-a.example"), mkDoc("bgp 65000 peer", "blog-b.example")
	fp := "fingerprint-copia"
	claims := []domain.Claim{{
		CampaignID: camp.ID, Title: "BGP", Statement: "x", Status: "unconfirmed", Applicability: "indeterminate", OperationalValidation: "untested",
		Evidences: []domain.Evidence{
			{DocumentID: docA, ClusterID: &fp, Stance: "supports", Excerpt: "a", Notes: "Fonte: blog-a.example"},
			{DocumentID: docB, ClusterID: &fp, Stance: "supports", Excerpt: "b", Notes: "Fonte: blog-b.example"},
		},
	}}

	for i := 0; i < 3; i++ { // três execuções de /investigate
		if err := st.SaveClaims(ctx, claims); err != nil {
			t.Fatalf("SaveClaims #%d: %v", i+1, err)
		}
	}
	got, err := st.ListClaimsByCampaign(ctx, camp.ID)
	if err != nil {
		t.Fatal(err)
	}
	if len(got) != 1 || len(got[0].Evidences) != 2 {
		t.Fatalf("reexecuções não podem duplicar: %d claims, evidências=%d", len(got), len(got[0].Evidences))
	}
	a, b := got[0].Evidences[0].ClusterID, got[0].Evidences[1].ClusterID
	if a == nil || b == nil || *a != *b {
		t.Fatalf("cópias do mesmo texto devem compartilhar um cluster persistido: %v %v", a, b)
	}
	var count int
	var rootDomain string
	if err := st.Pool().QueryRow(ctx, `SELECT duplicate_count, root_domain FROM origin_clusters WHERE id=$1::uuid`, *a).Scan(&count, &rootDomain); err != nil {
		t.Fatal(err)
	}
	if count != 2 || rootDomain == "" {
		t.Errorf("origin_cluster: réplicas=%d (quer 2), domínio=%q", count, rootDomain)
	}

	// Claim sem validação some quando deixa de ser produzido; o validado em bancada permanece.
	if _, err := st.Pool().Exec(ctx, `UPDATE claims SET operational_validation='lab_validated' WHERE campaign_id=$1::uuid`, camp.ID); err != nil {
		t.Fatal(err)
	}
	other := []domain.Claim{{CampaignID: camp.ID, Title: "Outro", Statement: "y", Status: "unconfirmed", Applicability: "indeterminate", OperationalValidation: "untested"}}
	if err := st.SaveClaims(ctx, other); err != nil {
		t.Fatal(err)
	}
	got, _ = st.ListClaimsByCampaign(ctx, camp.ID)
	if len(got) != 2 {
		t.Errorf("claim validado em bancada deve ser preservado: %d claims", len(got))
	}
}

func TestP1_EdgeKindPreservedForSitemapAndFeed(t *testing.T) {
	st, ctx := newTestStore(t)
	camp, err := st.CreateCampaign(ctx, "arestas "+fmt.Sprint(time.Now().UnixNano()), []string{"https://seed.example/sitemap.xml"},
		domain.Budget{MaxDepth: 2, MaxURLs: 10, MaxDomains: 5, MaxBytes: 10 << 20, MinDomainIntervalSeconds: 1})
	if err != nil {
		t.Fatal(err)
	}
	cleanupCampaign(t, st, camp.ID)
	for _, kind := range []string{"sitemap", "feed", "link"} {
		if ok, err := st.EnqueueFetch(ctx, camp.ID, "https://seed.example/"+kind, 1, "https://seed.example/sitemap.xml", kind, 50); err != nil || !ok {
			t.Fatalf("EnqueueFetch %s: ok=%v err=%v", kind, ok, err)
		}
	}
	rows, err := st.Pool().Query(ctx, `SELECT kind, to_url FROM discovery_edges WHERE campaign_id=$1::uuid`, camp.ID)
	if err != nil {
		t.Fatal(err)
	}
	defer rows.Close()
	got := map[string]string{}
	for rows.Next() {
		var k, u string
		if err := rows.Scan(&k, &u); err != nil {
			t.Fatal(err)
		}
		got[u] = k
	}
	for _, kind := range []string{"sitemap", "feed", "link"} {
		if got["https://seed.example/"+kind] != kind {
			t.Errorf("aresta para /%s gravada como %q", kind, got["https://seed.example/"+kind])
		}
	}
}
