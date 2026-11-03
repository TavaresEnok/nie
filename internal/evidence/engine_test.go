package evidence

import (
	"strings"
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
	// Deve receber status "documented" e aplicabilidade "indeterminate", NUNCA "confirmed" ou "applicable" (P0-01)
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

	// Case 2: Manual legítimo com modelo, firmware e feature comprovados via campos estruturados
	c2 := &domain.Claim{Title: "BGP Test 2"}
	eng.ResolveClaimStatus(c2, []domain.Evidence{
		{
			ID:            "e1",
			Stance:        "supports",
			IsOfficialDoc: true,
			VendorMatch:   true,
			FirmwareMatch: true,
			FeatureMatch:  true,
			ModelMatch:    true,
			Notes:         "Fonte: huawei.com",
		},
	})
	if c2.Status != "sustained" || c2.Applicability != "indeterminate" {
		t.Errorf("manual oficial escopado sem laboratório deve ser sustained/indeterminate, obteve %s/%s", c2.Status, c2.Applicability)
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
	if c3.Status != "unconfirmed" || c3.Applicability != "indeterminate" {
		t.Errorf("3 cópias do mesmo artigo não devem ser corroboradas, obteve status/applicability: %s/%s", c3.Status, c3.Applicability)
	}

	// Case 4: Duas origens genuinamente independentes com verificação estruturada
	c4 := &domain.Claim{Title: "BGP Test 4"}
	clusterA := "cluster-a"
	clusterB := "cluster-b"
	eng.ResolveClaimStatus(c4, []domain.Evidence{
		{
			ID:            "e1",
			Stance:        "supports",
			ClusterID:     &clusterA,
			VendorMatch:   true,
			ModelMatch:    true,
			FirmwareMatch: true,
			FeatureMatch:  true,
			Notes:         "Fonte: netops.org",
		},
		{
			ID:            "e2",
			Stance:        "supports",
			ClusterID:     &clusterB,
			VendorMatch:   true,
			ModelMatch:    true,
			FirmwareMatch: true,
			FeatureMatch:  true,
			Notes:         "Fonte: bbr.org",
		},
	})
	if c4.Status != "corroborated" || c4.Applicability != "indeterminate" {
		t.Errorf("duas origens escopadas sem laboratório devem ser corroborated/indeterminate, obteve %s/%s", c4.Status, c4.Applicability)
	}

	// Case 5: refutação que trata do mesmo fabricante, modelo, firmware e recurso
	// torna a claim contestada e o cenário incompatível.
	c5 := &domain.Claim{Title: "BGP Test 5"}
	eng.ResolveClaimStatus(c5, []domain.Evidence{
		{ID: "e1", Stance: "supports", IsOfficialDoc: true, Notes: "Fonte: huawei.com"},
		{ID: "e2", Stance: "refutes", VendorMatch: true, ModelMatch: true, FirmwareMatch: true, FeatureMatch: true,
			Notes: "Fonte: forum.huawei.com (sintaxe obsoleta no V8)"},
	})
	if c5.Status != "contested" || c5.Applicability != "incompatible" {
		t.Errorf("Refutação com escopo completo deve resultar em contested/incompatible, obteve %s/%s", c5.Status, c5.Applicability)
	}

	// Case 6: refutação do fabricante e recurso, sem comprovar modelo/firmware:
	// contesta a claim, mas não decide a incompatibilidade do equipamento.
	c6 := &domain.Claim{Title: "BGP Test 6"}
	eng.ResolveClaimStatus(c6, []domain.Evidence{
		{ID: "e1", Stance: "refutes", VendorMatch: true, FeatureMatch: true, Notes: "Fonte: forum.huawei.com"},
	})
	if c6.Status != "contested" || c6.Applicability != "indeterminate" {
		t.Errorf("Refutação sem modelo/firmware deve ser contested/indeterminate, obteve %s/%s", c6.Status, c6.Applicability)
	}

	// Case 7: refutação fora do escopo (outro fabricante/recurso) é ignorada.
	c7 := &domain.Claim{Title: "BGP Test 7"}
	eng.ResolveClaimStatus(c7, []domain.Evidence{
		{ID: "e1", Stance: "supports", IsOfficialDoc: true, Notes: "Fonte: huawei.com"},
		{ID: "e2", Stance: "refutes", Notes: "Fonte: forum.cisco.com"},
	})
	if c7.Status != "documented" || c7.Applicability != "indeterminate" {
		t.Errorf("Refutação fora de escopo não pode contestar a claim, obteve %s/%s", c7.Status, c7.Applicability)
	}
}

// TestP001_DivergentVendorNeverApplicable valida que divergência de fabricante resulta em refutação e incompatibilidade
func TestP001_DivergentVendorNeverApplicable(t *testing.T) {
	eng := NewEngine()
	query := "Cisco ASR9000 BGP peering"

	passages := []domain.SearchResult{
		{
			PassageID:  1,
			DocumentID: "doc-huawei",
			Domain:     "support.huawei.com",
			Snippet:    "bgp 65000\n peer 10.1.1.1 as-number 65001",
			SHA256:     "sha-hw",
		},
	}

	claims := eng.BuildClaimsFromPassages("camp-divergent", query, passages)
	if len(claims) == 0 {
		t.Fatalf("esperava ao menos 1 claim sintetizada")
	}

	for _, c := range claims {
		if c.Applicability == "applicable" || c.Status == "sustained" {
			t.Errorf("Passagem de fabricante divergente (Huawei para consulta Cisco) gerou applicable/sustained indevidamente: %+v", c)
		}
		if c.Applicability != "incompatible" && c.Applicability != "indeterminate" {
			t.Errorf("Esperava indeterminate ou incompatible para fabricante divergente, obteve %s", c.Applicability)
		}
	}
}

// TestP001_MissingFirmwareVersionNeverPromotedToApplicable valida que ausência de versão solicitada impede promoção
func TestP001_MissingFirmwareVersionNeverPromotedToApplicable(t *testing.T) {
	eng := NewEngine()
	// Consulta sem especificação de versão ou firmware
	query := "Huawei NE8000 BGP configuration"

	passages := []domain.SearchResult{
		{
			PassageID:  1,
			DocumentID: "doc-official",
			Domain:     "support.huawei.com",
			Snippet:    "NE8000 bgp 65000 peer 1.1.1.1 as-number 65001",
			SHA256:     "sha-official",
		},
	}

	claims := eng.BuildClaimsFromPassages("camp-no-firmware", query, passages)
	for _, c := range claims {
		if c.Applicability == "applicable" {
			t.Errorf("Documento sem firmware especificado/verificado não pode ser applicable: %+v", c)
		}
		if c.Status != "documented" {
			t.Errorf("Status deveria ser documented para manual oficial sem versão verificada, obteve %s", c.Status)
		}
	}
}

// TestP001_CompoundRequirementsNotCollapsed valida que múltiplas features (BGP + RADIUS + PPPoE) não são colapsadas
func TestP001_CompoundRequirementsNotCollapsed(t *testing.T) {
	eng := NewEngine()
	query := "MikroTik RouterOS v7 BGP RADIUS PPPoE failover"

	req := ParseRequirements(query)
	if len(req.Features) < 3 {
		t.Fatalf("Esperava que features BGP, RADIUS e PPPoE fossem detectadas individualmente, obteve: %v", req.Features)
	}

	// Passagem apenas fala de BGP, não comprova RADIUS nem continuidade de PPPoE
	passages := []domain.SearchResult{
		{
			PassageID:  1,
			DocumentID: "doc-bgp-only",
			Domain:     "help.mikrotik.com",
			Snippet:    "RouterOS v7 /routing bgp connection add name=peer1 as=65000",
			SHA256:     "sha-bgp",
		},
	}

	claims := eng.BuildClaimsFromPassages("camp-compound", query, passages)
	for _, c := range claims {
		if c.Applicability == "applicable" || c.Status == "sustained" {
			t.Errorf("passagem só de BGP não sustenta a consulta composta: %+v", c)
		}
		if strings.Contains(c.Title, "RADIUS") || strings.Contains(c.Title, "PPPOE") {
			for _, ev := range c.Evidences {
				if ev.FeatureMatch && ev.Stance == "supports" {
					t.Errorf("RADIUS ou PPPoE não foram afirmados: %+v", ev)
				}
			}
		}
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
		if c.Applicability == "confirmed" || c.Applicability == "applicable" {
			t.Errorf("Claim não pode ser confirmada com evidências de feature errada ou firmware incompatível: %+v", c)
		}
	}
}

// TestAudit_MissingModelNeverApplicable assegura que ausência de ModelMatch impede aplicabilidade
func TestAudit_MissingModelNeverApplicable(t *testing.T) {
	eng := NewEngine()
	claim := &domain.Claim{
		ID:    "claim-audit-1",
		Title: "Huawei NE8000 firmware BGP",
	}
	evidences := []domain.Evidence{
		{
			ID:            "ev-audit-1",
			Stance:        "supports",
			IsOfficialDoc: true,
			VendorMatch:   true,
			ModelMatch:    false, // modelo não comprovado!
			FirmwareMatch: true,
			FeatureMatch:  true,
			Notes:         "Fonte: support.huawei.com | firmware verificado | modelo correspondente",
		},
	}

	eng.ResolveClaimStatus(claim, evidences)

	if claim.Applicability == "applicable" {
		t.Fatalf("VIOLAÇÃO P0-01: Claim com ModelMatch=false foi marcada como applicable! Status: %s, Applicability: %s",
			claim.Status, claim.Applicability)
	}
	if claim.Status != "documented" || claim.Applicability != "indeterminate" {
		t.Fatalf("Esperava status 'documented' e applicability 'indeterminate', obteve status '%s' e applicability '%s'",
			claim.Status, claim.Applicability)
	}
}

// TestAudit_NotesCannotOverrideMissingStructuredEvidence assegura que notas textuais jamais sobrepõem campos estruturados falsos
func TestAudit_NotesCannotOverrideMissingStructuredEvidence(t *testing.T) {
	eng := NewEngine()
	claim := &domain.Claim{
		ID:    "claim-audit-2",
		Title: "Huawei NE8000 BGP",
	}
	evidences := []domain.Evidence{
		{
			ID:            "ev-audit-2",
			Stance:        "supports",
			IsOfficialDoc: true,
			VendorMatch:   false,
			ModelMatch:    false,
			FirmwareMatch: false,
			FeatureMatch:  false,
			Notes:         "Fonte: support.huawei.com | firmware verificado | modelo correspondente",
		},
	}

	eng.ResolveClaimStatus(claim, evidences)

	if claim.Applicability == "applicable" || claim.Status == "sustained" {
		t.Fatalf("VIOLAÇÃO P0-01: Notas textuais sobrepuseram campos estruturados falsos! Status: %s, Applicability: %s",
			claim.Status, claim.Applicability)
	}
	if claim.Status != "documented" || claim.Applicability != "indeterminate" {
		t.Fatalf("Esperava status 'documented' e applicability 'indeterminate', obteve status '%s' e applicability '%s'",
			claim.Status, claim.Applicability)
	}
}

// TestAudit_DocumentedPassageWithoutModelEvidence assegura que passagem oficial sem menção ao modelo não é promovida a applicable
func TestAudit_DocumentedPassageWithoutModelEvidence(t *testing.T) {
	eng := NewEngine()
	query := "Huawei NE8000 VRPv8 BGP peering"

	passages := []domain.SearchResult{
		{
			PassageID:  1,
			DocumentID: "doc-official-no-model",
			Domain:     "support.huawei.com",
			Snippet:    "Huawei VRPv8 bgp 65000 peer 10.1.1.1 as-number 65001", // contém Huawei, VRPv8, BGP, mas NÃO contém NE8000
			SHA256:     "sha256-no-model",
		},
	}

	claims := eng.BuildClaimsFromPassages("camp-audit-3", query, passages)
	if len(claims) == 0 {
		t.Fatalf("Esperava ao menos 1 claim sintetizada")
	}

	for _, c := range claims {
		if c.Applicability == "applicable" {
			t.Fatalf("VIOLAÇÃO P0-01: Passagem oficial sem modelo NE8000 gerou applicability 'applicable': %+v", c)
		}
		if c.Status != "documented" || c.Applicability != "indeterminate" {
			t.Fatalf("Esperava status 'documented' e applicability 'indeterminate', obteve status '%s' e applicability '%s'",
				c.Status, c.Applicability)
		}
		for _, ev := range c.Evidences {
			if ev.ModelMatch {
				t.Fatalf("Evidência não deveria ter ModelMatch=true sem menção ao NE8000: %+v", ev)
			}
		}
	}
}
