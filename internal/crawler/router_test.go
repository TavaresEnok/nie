package crawler

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"nie/internal/domain"
)

type mockExecutor struct {
	name       string
	calledWith []domain.FetchTask
}

func (m *mockExecutor) Name() string { return m.name }
func (m *mockExecutor) Execute(ctx context.Context, task domain.FetchTask) (*domain.FetchResult, error) {
	m.calledWith = append(m.calledWith, task)
	return &domain.FetchResult{
		TaskID:            task.ID,
		EffectiveURL:      task.URL,
		StatusCode:        200,
		ContentType:       "text/html",
		Text:              "RFC 1997 e NE8000 teste",
		ExtractedLinks:    []string{"https://example.com/sub"},
		ExtractedMentions: []string{"RFC 1997", "NE8000"},
		Duration:          10 * time.Millisecond,
		Metadata:          map[string]any{"executor": m.name},
	}, nil
}

func TestRouterSelection(t *testing.T) {
	router := NewRouter()
	native := &mockExecutor{name: "native"}
	scrapy := &mockExecutor{name: "scrapy"}
	crawl4ai := &mockExecutor{name: "crawl4ai"}

	router.Register(native)
	router.Register(scrapy)
	router.Register(crawl4ai)

	// 1. Explicit executor requested
	task1 := domain.FetchTask{ID: "1", URL: "https://example.com/page1", Executor: "scrapy"}
	res1, err := router.Execute(context.Background(), task1)
	if err != nil {
		t.Fatalf("erro ao executar task1: %v", err)
	}
	if res1.Metadata["executor"] != "scrapy" {
		t.Errorf("esperava scrapy, obteve %v", res1.Metadata["executor"])
	}
	if len(scrapy.calledWith) != 1 {
		t.Errorf("scrapy deveria ter sido chamado 1 vez, foi %d", len(scrapy.calledWith))
	}
	if len(native.calledWith) != 0 || len(crawl4ai.calledWith) != 0 {
		t.Errorf("apenas um executor deve ser acionado por URL")
	}

	// 2. Mode browser -> Crawl4AI
	task2 := domain.FetchTask{ID: "2", URL: "https://example.com/spa", Mode: "browser"}
	res2, err := router.Execute(context.Background(), task2)
	if err != nil {
		t.Fatalf("erro ao executar task2: %v", err)
	}
	if res2.Metadata["executor"] != "crawl4ai" {
		t.Errorf("esperava crawl4ai, obteve %v", res2.Metadata["executor"])
	}

	// 3. Default -> Native
	task3 := domain.FetchTask{ID: "3", URL: "https://example.com/normal"}
	res3, err := router.Execute(context.Background(), task3)
	if err != nil {
		t.Fatalf("erro ao executar task3: %v", err)
	}
	if res3.Metadata["executor"] != "native" {
		t.Errorf("esperava native, obteve %v", res3.Metadata["executor"])
	}
}

func TestScrapyBridgeLive(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(`{
			"effective_url": "https://example.com/scraped",
			"status_code": 200,
			"content_type": "text/html",
			"text": "Manual do DM4270 com suporte a BGP EVPN",
			"extracted_links": ["https://example.com/manual2"]
		}`))
	}))
	defer ts.Close()

	bridge := NewScrapyBridge(ts.URL, nil)
	task := domain.FetchTask{ID: "scrapy-test", URL: "https://example.com/doc"}
	res, err := bridge.Execute(context.Background(), task)
	if err != nil {
		t.Fatalf("erro ao executar ScrapyBridge: %v", err)
	}

	if res.StatusCode != 200 || res.EffectiveURL != "https://example.com/scraped" {
		t.Errorf("resposta inesperada do ScrapyBridge: %+v", res)
	}
	if len(res.ExtractedLinks) != 1 || res.ExtractedLinks[0] != "https://example.com/manual2" {
		t.Errorf("links extraídos incorretos: %v", res.ExtractedLinks)
	}
	if len(res.ExtractedMentions) == 0 {
		t.Errorf("esperava menções extraídas (DM4270 / BGP EVPN), obteve vazias")
	}
}

func TestCrawl4AIBridgeLive(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(`{
			"effective_url": "https://example.com/rendered",
			"status_code": 200,
			"content_type": "text/markdown",
			"markdown": "# Roteador Huawei NE8000\nConsulte a RFC 4360.",
			"extracted_links": ["https://example.com/bgp"]
		}`))
	}))
	defer ts.Close()

	bridge := NewCrawl4AIBridge(ts.URL, nil)
	task := domain.FetchTask{ID: "c4ai-test", URL: "https://example.com/page", Mode: "browser"}
	res, err := bridge.Execute(context.Background(), task)
	if err != nil {
		t.Fatalf("erro no Crawl4AIBridge: %v", err)
	}

	if res.StatusCode != 200 || res.Metadata["executor"] != "crawl4ai" {
		t.Errorf("metadados inesperados: %+v", res)
	}
	foundRFC := false
	for _, m := range res.ExtractedMentions {
		if m == "RFC 4360" {
			foundRFC = true
		}
	}
	if !foundRFC {
		t.Errorf("esperava RFC 4360 nas menções: %v", res.ExtractedMentions)
	}
}

// TestRouterDynamicURLHeuristics (P1-01) valida a seleção automática e determinística
// entre Native, Scrapy e Crawl4AI/Browser sem requisição manual de executor.
func TestRouterDynamicURLHeuristics(t *testing.T) {
	router := NewRouter()
	native := &mockExecutor{name: "native"}
	scrapy := &mockExecutor{name: "scrapy"}
	crawl4ai := &mockExecutor{name: "crawl4ai"}

	router.Register(native)
	router.Register(scrapy)
	router.Register(crawl4ai)

	// 1. URL com SPA / dynamic hash -> Crawl4AI
	t1 := domain.FetchTask{ID: "t1", URL: "https://support.huawei.com/#/dashboard/device"}
	e1 := router.SelectExecutor(t1)
	if e1.Name() != "crawl4ai" {
		t.Errorf("Esperava crawl4ai para URL com hash SPA, obteve %s", e1.Name())
	}

	// 2. URL com catálogo de documentação técnica /docs/ -> Scrapy
	t2 := domain.FetchTask{ID: "t2", URL: "https://wiki.mikrotik.com/docs/manual/routing-bgp"}
	e2 := router.SelectExecutor(t2)
	if e2.Name() != "scrapy" {
		t.Errorf("Esperava scrapy para URL de documentação técnica /docs/, obteve %s", e2.Name())
	}

	// 3. URL padrão estática -> Native
	t3 := domain.FetchTask{ID: "t3", URL: "https://example.org/about.html"}
	e3 := router.SelectExecutor(t3)
	if e3.Name() != "native" {
		t.Errorf("Esperava native para URL estática simples, obteve %s", e3.Name())
	}
}

// TestRawContentPreservationSeparatedFromDerived (P1-04) valida que os bytes originais (RawHTML)
// são preservados em RawContent para cálculo de SHA-256 no CAS, e não o Markdown/texto derivado.
func TestRawContentPreservationSeparatedFromDerived(t *testing.T) {
	rawHTML := "<html><head><title>Original HTML</title></head><body><h1>Comandos OLT</h1></body></html>"
	markdownDerived := "# Comandos OLT\n"

	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(`{
			"effective_url": "https://example.com/olt-doc",
			"status_code": 200,
			"content_type": "text/markdown",
			"markdown": "# Comandos OLT\n",
			"raw_html": "<html><head><title>Original HTML</title></head><body><h1>Comandos OLT</h1></body></html>",
			"extracted_links": ["https://example.com/next"]
		}`))
	}))
	defer ts.Close()

	bridge := NewCrawl4AIBridge(ts.URL, nil)
	task := domain.FetchTask{ID: "c4ai-cas-test", URL: "https://example.com/olt-doc", Mode: "browser"}
	res, err := bridge.Execute(context.Background(), task)
	if err != nil {
		t.Fatalf("erro no Crawl4AIBridge: %v", err)
	}

	// RawContent DEVE conter os bytes exatos do rawHTML recebido, preservando integridade CAS
	if string(res.RawContent) != rawHTML {
		t.Errorf("P1-04: RawContent não preservou o HTML original: obteve %q", string(res.RawContent))
	}
	// Text deve conter a representação derivada para busca/markdown
	if res.Text != markdownDerived {
		t.Errorf("P1-04: Text derivado incorreto: obteve %q", res.Text)
	}
}
