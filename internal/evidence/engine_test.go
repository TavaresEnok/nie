package evidence

import (
	"testing"

	"nie/internal/domain"
)

func TestOriginClusteringDoesNotOvercountCopies(t *testing.T) {
	tracker := NewClusterTracker()

	sampleText := "Para configurar BGP no MikroTik use /routing bgp connection add name=peer1 as=65000"
	fp := Fingerprint(sampleText)

	// Blog 1 copies the text
	c1 := tracker.Add(fp, "blog1.com")
	if c1 != 1 {
		t.Errorf("esperava 1 réplica, obteve %d", c1)
	}

	// Blog 2 copies the exact same text
	c2 := tracker.Add(fp, "blog2.com")
	if c2 != 2 {
		t.Errorf("esperava 2 réplicas no cluster, obteve %d", c2)
	}

	// Blog 1 repeats same article on another path
	c3 := tracker.Add(fp, "blog1.com")
	if c3 != 2 {
		t.Errorf("mesmo domínio não deve inflar contagem de réplicas, obteve %d", c3)
	}

	// Teste de contagem real de origens independentes (P1-06):
	// Três blogs copiando o EXATO MESMO TEXTO devem contar como 1 ÚNICA origem independente!
	fps := []string{fp, fp, fp}
	doms := []string{"blog1.com", "blog2.com", "blog3.com"}
	origins := tracker.CountIndependentOrigins(fps, doms)
	if origins != 1 {
		t.Errorf("3 cópias do mesmo artigo devem contar como 1 única origem independente, obteve %d", origins)
	}

	// Três trechos distintos em domínios distintos contam como 3 origens
	fpsDiff := []string{"fp1", "fp2", "fp3"}
	originsDiff := tracker.CountIndependentOrigins(fpsDiff, doms)
	if originsDiff != 3 {
		t.Errorf("3 textos distintos em 3 domínios distintos devem contar como 3 origens, obteve %d", originsDiff)
	}
}

func TestResolveClaimStatusAdversarial(t *testing.T) {
	eng := NewEngine()

	// Adversarial Case 1: Manual legítimo sem validação de modelo/firmware específico
	// Deve receber status "documented" e aplicabilidade "indeterminate", NUNCA "confirmed" (P0-01)
	c1 := &domain.Claim{Title: "BGP Test"}
	eng.ResolveClaimStatus(c1, []domain.Evidence{
		{ID: "e1", Stance: "supports", IsOfficialDoc: true, Notes: "Fonte: huawei.com"},
	})
	if c1.Status != "documented" {
		t.Errorf("Manual oficial sem comprovação de firmware deve ter status documented, obteve %s", c1.Status)
	}
	if c1.Applicability != "indeterminate" {
		t.Errorf("Manual oficial sem comprovação de firmware não deve ter applicability confirmed, obteve %s", c1.Applicability)
	}

	// Case 2: Manual legítimo com modelo e firmware explicitamente verificados
	c2 := &domain.Claim{Title: "BGP Test 2"}
	eng.ResolveClaimStatus(c2, []domain.Evidence{
		{ID: "e1", Stance: "supports", IsOfficialDoc: true, Notes: "Fonte: huawei.com | firmware verificado | modelo correspondente"},
	})
	if c2.Status != "sustained" || c2.Applicability != "applicable" {
		t.Errorf("Manual oficial com firmware verificado deve ser sustained/applicable, obteve %s/%s", c2.Status, c2.Applicability)
	}

	// Adversarial Case 3: Três documentos copiando o MESMO artigo (mesmo fingerprint)
	// Não devem corroborar como aplicável
	c3 := &domain.Claim{Title: "BGP Test 3"}
	clusterSame := "same-fingerprint"
	eng.ResolveClaimStatus(c3, []domain.Evidence{
		{ID: "e1", Stance: "supports", ClusterID: &clusterSame, Notes: "Fonte: blog1.com"},
		{ID: "e2", Stance: "supports", ClusterID: &clusterSame, Notes: "Fonte: blog2.com"},
		{ID: "e3", Stance: "supports", ClusterID: &clusterSame, Notes: "Fonte: blog3.com"},
	})
	if c3.Status != "unconfirmed" {
		t.Errorf("3 cópias do mesmo artigo não devem ser corroboradas, obteve status: %s", c3.Status)
	}

	// Case 4: Duas origens genuinamente independentes com firmware verificado
	c4 := &domain.Claim{Title: "BGP Test 4"}
	clusterA := "cluster-a"
	clusterB := "cluster-b"
	eng.ResolveClaimStatus(c4, []domain.Evidence{
		{ID: "e1", Stance: "supports", ClusterID: &clusterA, Notes: "Fonte: netops.org | firmware verificado"},
		{ID: "e2", Stance: "supports", ClusterID: &clusterB, Notes: "Fonte: bbr.org | firmware verificado"},
	})
	if c4.Status != "corroborated" || c4.Applicability != "applicable" {
		t.Errorf("Duas origens independentes com firmware verificado devem ser corroborated/applicable, obteve %s/%s", c4.Status, c4.Applicability)
	}

	// Adversarial Case 5: Contradição ou sintaxe obsoleta/depreciada
	c5 := &domain.Claim{Title: "BGP Test 5"}
	eng.ResolveClaimStatus(c5, []domain.Evidence{
		{ID: "e1", Stance: "supports", IsOfficialDoc: true, Notes: "Fonte: huawei.com"},
		{ID: "e2", Stance: "refutes", Notes: "Fonte: forum.huawei.com (sintaxe obsoleta no V8)"},
	})
	if c5.Status != "contested" || c5.Applicability != "incompatible" {
		t.Errorf("Contradição deve resultar em contested/incompatible, obteve %s/%s", c5.Status, c5.Applicability)
	}
}

func TestBuildClaimsFeatureAndFirmwareMismatch(t *testing.T) {
	eng := NewEngine()

	// Query pede BGP em RouterOS v7
	query := "MikroTik RouterOS v7 BGP peering"

	passages := []domain.SearchResult{
		// Passagem 1: PPPoE em vez de BGP (feature mismatch)
		{
			PassageID:  1,
			DocumentID: "doc-1",
			Domain:     "forum.mikrotik.com",
			Snippet:    "/interface pppoe-server add service-name=internet one-session-per-host=yes",
			SHA256:     "sha1",
		},
		// Passagem 2: RouterOS v6 BGP (firmware mismatch explícito)
		{
			PassageID:  2,
			DocumentID: "doc-2",
			Domain:     "wiki.mikrotik.com",
			Snippet:    "No RouterOS v6 a configuração de BGP é /routing bgp peer add name=peer1 as=65000",
			SHA256:     "sha2",
		},
	}

	claims := eng.BuildClaimsFromPassages("camp-1", query, passages)
	if len(claims) == 0 {
		t.Fatalf("esperava ao menos 1 claim estruturada")
	}

	for _, c := range claims {
		// Nenhuma claim deve ser marcada como confirmed ou applicable sem prova
		if c.Applicability == "confirmed" {
			t.Errorf("Claim não pode ser confirmada com evidências de feature errada ou firmware incompatível: %+v", c)
		}
	}
}
