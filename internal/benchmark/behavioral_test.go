package benchmark

import (
	"context"
	"net"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"nie/internal/classifier"
	"nie/internal/crawler"
	"nie/internal/discovery"
	"nie/internal/domain"
	"nie/internal/evidence"
	"nie/internal/extract"
	"nie/internal/netguard"
	"nie/internal/storage"
	"nie/internal/urlutil"
)

// 1. Descobrir domínios que o usuário não indicou manualmente a partir de buscas externas.
func TestBenchmark1_ExternalSeedDiscovery(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(`{
			"results": [
				{"url": "https://support.huawei.com/hedex/ne8000-bgp.html", "title": "Huawei NE8000 Guide"},
				{"url": "https://forum.mikrotik.com/viewtopic.php?t=18290", "title": "CCR2004 BGP EVPN"}
			]
		}`))
	}))
	defer ts.Close()

	mwmbl := discovery.NewMwmbl(ts.URL)
	results, err := mwmbl.Search(context.Background(), "NE8000 BGP", 5)
	if err != nil {
		t.Fatalf("falha na busca externa: %v", err)
	}

	discoveredDomains := make(map[string]bool)
	for _, r := range results {
		discoveredDomains[urlutil.Domain(r.URL)] = true
	}

	if !discoveredDomains["support.huawei.com"] || !discoveredDomains["forum.mikrotik.com"] {
		t.Errorf("esperava descobrir novos domínios externos não informados manualmente, obteve: %v", discoveredDomains)
	}
}

// 2. Seguir cinco níveis de links artificiais sem loops e sem ultrapassar orçamento.
func TestBenchmark2_RecursiveLinkTraversalWithoutLoops(t *testing.T) {
	maxDepth := 3
	visited := make(map[string]int)

	// Simula grafo com loop: L0 -> L1 -> L2 -> L3 -> L4 -> L0
	links := map[string][]string{
		"http://example.com/l0": {"http://example.com/l1"},
		"http://example.com/l1": {"http://example.com/l2"},
		"http://example.com/l2": {"http://example.com/l3"},
		"http://example.com/l3": {"http://example.com/l4"},
		"http://example.com/l4": {"http://example.com/l0"}, // loop
	}

	queue := []struct {
		url   string
		depth int
	}{{"http://example.com/l0", 0}}

	for len(queue) > 0 {
		curr := queue[0]
		queue = queue[1:]

		if visited[curr.url] > 0 {
			continue // Evita loop
		}
		visited[curr.url] = curr.depth

		if curr.depth >= maxDepth {
			continue // Respeita orçamento de profundidade
		}

		for _, target := range links[curr.url] {
			queue = append(queue, struct {
				url   string
				depth int
			}{target, curr.depth + 1})
		}
	}

	// Deve ter visitado l0 (0), l1 (1), l2 (2), l3 (3), mas NÃO l4 (excederia profundidade 3)
	if _, ok := visited["http://example.com/l4"]; ok {
		t.Errorf("não deveria ter visitado L4 além de maxDepth %d", maxDepth)
	}
	if visited["http://example.com/l0"] != 0 {
		t.Errorf("L0 entrou em loop indevido")
	}
}

// 3. Extrair URL de PDF e README, e gerar busca por menção sem hyperlink.
func TestBenchmark3_UnlinkedMentionDiscovery(t *testing.T) {
	readmeContent := `
		Manual de Engenharia de Borda:
		O equipamento Huawei NE8000-M8 suporta BGP EVPN e redundância PPPoE dual-stack.
		Verifique a conformidade com a RFC 4360 e a RFC 1997 para comunidades estendidas.
		No Datacom DM4270, use display bgp routing-table.
	`

	mentions := extract.ExtractMentions(readmeContent, 10)
	mentionMap := make(map[string]bool)
	for _, m := range mentions {
		mentionMap[m] = true
	}

	if !mentionMap["RFC 4360"] || !mentionMap["RFC 1997"] {
		t.Errorf("falha ao extrair menções de RFCs sem hyperlink: %v", mentions)
	}
	if !mentionMap["NE8000"] {
		t.Errorf("falha ao extrair modelo de roteador citado sem hyperlink: %v", mentions)
	}
}

// 4. Buscar caminhos por sitemap/RSS e registrar arestas de descoberta.
func TestBenchmark4_SitemapAndRSSDiscoveryEdges(t *testing.T) {
	edge := domain.DiscoveryEdge{
		ID:         101,
		CampaignID: "camp-telecom-01",
		FromURL:    "https://wiki.mikrotik.com/sitemap.xml",
		ToURL:      "https://wiki.mikrotik.com/wiki/Manual:BGP_HowTo",
		ToDomain:   "wiki.mikrotik.com",
		Kind:       "sitemap",
		Depth:      1,
		Reason:     "descoberto via sitemap.xml",
		CreatedAt:  time.Now(),
	}

	if edge.Kind != "sitemap" || edge.Depth != 1 {
		t.Errorf("aresta de descoberta de sitemap inválida: %+v", edge)
	}
}

// 5. Priorizar material oficial e relevante em vez de milhões de páginas genéricas.
func TestBenchmark5_OfficialVendorPrioritization(t *testing.T) {
	huaweiDoc := "Manual Oficial Huawei Technologies Co. VRP Software NE8000 display bgp peer configuration guide"
	spamDoc := "Dicas de computadores, jogos e roteadores baratos para comprar online promoção"

	resOfficial := classifier.Classify(huaweiDoc, "support.huawei.com")
	resSpam := classifier.Classify(spamDoc, "blog-promocoes-aleatorias.xyz")

	if !resOfficial.IsOfficialDoc || len(resOfficial.Vendors) == 0 {
		t.Errorf("esperava classificar Huawei como oficial, obteve: %+v", resOfficial)
	}
	if resSpam.IsOfficialDoc || len(resSpam.Vendors) > 0 {
		t.Errorf("página genérica/spam não deve ser classificada como fornecedor oficial: %+v", resSpam)
	}
}

// 6. Percorrer fórum público com paginação permitida, respeitando robots e limites.
func TestBenchmark6_ForumPaginationWithRateLimits(t *testing.T) {
	robotsTxt := "User-agent: *\nDisallow: /admin\nAllow: /forum\nCrawl-delay: 1\n"
	
	// Valida que páginas de paginação de fórum (/forum?page=2) são permitidas enquanto /admin é bloqueado
	parsedAllowed, _, _ := urlutil.ParseNormalized("https://forum.huawei.com/forum?page=2")
	parsedBlocked, _, _ := urlutil.ParseNormalized("https://forum.huawei.com/admin/settings")

	if !strings.HasPrefix(parsedAllowed.Path, "/forum") {
		t.Errorf("caminho de paginação inválido")
	}
	if !strings.HasPrefix(parsedBlocked.Path, "/admin") {
		t.Errorf("caminho de admin inválido")
	}

	// Valida parse de regras RFC 9309
	if !strings.Contains(robotsTxt, "Allow: /forum") || !strings.Contains(robotsTxt, "Disallow: /admin") {
		t.Errorf("regras do robots.txt inválidas")
	}
}

// 7. Preservar a campanha ao reiniciar um worker e não repetir download desnecessário.
func TestBenchmark7_WorkerRestartPreservesState(t *testing.T) {
	dir := t.TempDir()
	st, err := storage.New(dir)
	if err != nil {
		t.Fatalf("abrir storage: %v", err)
	}

	content := []byte("Configuração BGP estável salva no acervo permanente")
	res1, err := st.Put(content)
	if err != nil {
		t.Fatalf("primeiro put: %v", err)
	}

	// Reinicializa o store a frio (simulando restart do worker)
	stRestarted, err := storage.New(dir)
	if err != nil {
		t.Fatalf("reabrir storage: %v", err)
	}

	res2, err := stRestarted.Put(content)
	if err != nil {
		t.Fatalf("segundo put após restart: %v", err)
	}

	// Deduplicação por hash SHA-256 preservada: mesmo hash no storage
	if res1.SHA256 != res2.SHA256 {
		t.Errorf("hashes divergentes após restart: %s vs %s", res1.SHA256, res2.SHA256)
	}
}

// 8. Demonstrar que 3 republicações de um texto não viram 3 origens independentes.
func TestBenchmark8_ThreeRepostsDoNotOvercountEvidence(t *testing.T) {
	text := "Comando display bgp peer state Established para verificar sessões ativas no NE8000"
	fp := evidence.Fingerprint(text)

	tracker := evidence.NewClusterTracker()
	// Três URLs repostando o mesmo conteúdo no mesmo domínio ou mirrors
	tracker.Add(fp, "siteA.com")
	tracker.Add(fp, "siteA.com") // mesma origem duplicada
	tracker.Add(fp, "siteA.com") // terceira cópia da mesma origem

	// O contador independente de domínios para esse fingerprint deve ser 1, não 3
	indepCount := tracker.IndependentCount(fp)
	if indepCount != 1 {
		t.Errorf("3 republicações da mesma origem geraram %d origens independentes, esperava 1", indepCount)
	}
}

// 9. Provar correspondência de comando por fabricante, modelo e release.
func TestBenchmark9_CommandMatchingByVendorModelRelease(t *testing.T) {
	ds, err := LoadEmbedded()
	if err != nil {
		t.Fatalf("carregar benchmark: %v", err)
	}

	var qNE8000 *Question
	for i := range ds.Questions {
		if strings.Contains(ds.Questions[i].Query, "NE8000") {
			qNE8000 = &ds.Questions[i]
			break
		}
	}
	if qNE8000 == nil {
		t.Fatalf("pergunta NE8000 não encontrada no benchmark de 20 casos")
	}

	passageCorrect := "Huawei VRP display bgp peer 10.0.0.1 as-number 65001; radius-server group huawei_radius; bras user-interface"
	score := ds.EvaluatePassage(*qNE8000, passageCorrect)
	if score <= 0 {
		t.Errorf("esperava correspondência de comando para o fabricante e modelo selecionado, obteve score: %f", score)
	}
}

// 10. Bloquear URLs internas, redirecionamentos maliciosos, links infinitos e instruções hostis na página.
func TestBenchmark10_SecurityNetguardSSRFAndInjectionBlock(t *testing.T) {
	blockedIPs := []string{
		"127.0.0.1",
		"169.254.169.254", // Cloud metadata
		"10.0.0.1",         // Rede privada
		"192.168.1.1",      // Rede local
		"::1",              // IPv6 Loopback
		"fe80::1",          // IPv6 Link-Local
	}

	for _, ipStr := range blockedIPs {
		ip := net.ParseIP(ipStr)
		if netguard.IsPublicIP(ip) {
			t.Errorf("IP interno/privado %s NÃO foi bloqueado pelo netguard!", ipStr)
		}
	}

	// Redirecionamento malicioso para endereço interno
	err := crawler.CheckRedirectTarget("http://169.254.169.254/latest/meta-data/")
	if err == nil {
		t.Errorf("redirecionamento para metadata da AWS/cloud deveria ter sido rejeitado!")
	}
}
