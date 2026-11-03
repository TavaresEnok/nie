package evidence

import (
	"testing"

	"nie/internal/domain"
)

// Testes adversariais da Auditoria V6 (Seção 19.2 do Plano Mestre V3):
//   1. Suporte de um fabricante atribuído a outro na mesma passagem.
//   2. Negação de BGP atribuída indevidamente a OSPF.
//   3. Negação de PPPoE atribuída indevidamente a BGP.
//   4. Conflito explícito entre fontes ou orações transformado em incompatibilidade.

func TestAuditV6_VendorSupportNeverAttributedToAnotherVendor(t *testing.T) {
	eng := NewEngine()
	q := "Huawei NE8000 VRP v8 BGP"

	cases := []string{
		"Huawei NE8000 VRP v8 migration guide. Cisco ASR9000 supports BGP.",
		"Compared to Huawei NE8000 VRP v8, Cisco supports BGP.",
		"Unlike Huawei NE8000 VRP v8, Cisco ASR9000 supports BGP.",
		"On Huawei NE8000 VRP v8 interop lab, Juniper supports BGP.",
		"Ao contrário do Huawei NE8000 VRP v8, a Cisco suporta BGP.",
		"A diferencia de Huawei NE8000 VRP v8, Cisco soporta BGP.",
		"Huawei NE8000 VRP v8 supports OSPF, whereas Cisco supports BGP.",
	}

	for _, snippet := range cases {
		ps := []domain.SearchResult{{
			PassageID:  1,
			DocumentID: "doc-v6-vendor",
			Domain:     "support.huawei.com",
			Snippet:    snippet,
			SHA256:     "sha-v6-1",
		}}
		claims := eng.BuildClaimsFromPassages("camp-v6-1", q, ps)
		for _, c := range claims {
			if c.Status == "sustained" || c.Status == "corroborated" || c.Applicability == "applicable" {
				t.Errorf("suporte de outro fabricante atribuído a Huawei (%q): status=%s applicability=%s ev=%+v",
					snippet, c.Status, c.Applicability, c.Evidences)
			}
			for _, ev := range c.Evidences {
				if ev.Stance == "supports" {
					t.Errorf("evidência de outro fabricante recebeu stance='supports' (%q): %+v", snippet, ev)
				}
			}
		}
	}
}

func TestAuditV6_NegationOfBGPNotAttributedToOSPF(t *testing.T) {
	eng := NewEngine()
	q := "Huawei NE8000 VRP v8 OSPF"

	// Casos onde OSPF é suportado e BGP é negado na mesma frase/passagem.
	positiveOSPFCases := []string{
		"Huawei NE8000 VRP v8 supports OSPF and does not support BGP.",
		"Huawei NE8000 VRP v8 does not support BGP and supports OSPF.",
		"Huawei NE8000 VRP v8 supports OSPF, while BGP is not supported.",
		"While BGP is not supported, Huawei NE8000 VRP v8 supports OSPF.",
		"O Huawei NE8000 VRP v8 suporta OSPF e não suporta BGP.",
		"El Huawei NE8000 VRP v8 soporta OSPF y no soporta BGP.",
		"Huawei NE8000 VRP v8 supports OSPF without BGP.",
		"Huawei NE8000 VRP v8 supports OSPF, not BGP.",
	}

	for _, snippet := range positiveOSPFCases {
		ps := []domain.SearchResult{{
			PassageID:  1,
			DocumentID: "doc-v6-ospf",
			Domain:     "support.huawei.com",
			Snippet:    snippet,
			SHA256:     "sha-v6-2a",
		}}
		claims := eng.BuildClaimsFromPassages("camp-v6-2a", q, ps)
		for _, c := range claims {
			if c.Status == "contested" || c.Applicability == "incompatible" {
				t.Errorf("negação de BGP foi atribuída a OSPF (%q): status=%s applicability=%s ev=%+v",
					snippet, c.Status, c.Applicability, c.Evidences)
			}
			if c.Status != "sustained" || c.Applicability != "indeterminate" {
				t.Errorf("suporte explícito de OSPF deveria ser sustained/indeterminate (%q): obteve %s/%s",
					snippet, c.Status, c.Applicability)
			}
		}
	}

	// Casos onde BGP é negado em contexto que apenas menciona OSPF sem afirmar nem negar OSPF.
	mentionOSPFCases := []string{
		"Huawei NE8000 VRP v8 does not support BGP in OSPF topologies.",
		"BGP is not supported alongside OSPF on Huawei NE8000 VRP v8.",
		"With OSPF enabled, BGP is not supported on Huawei NE8000 VRP v8.",
	}
	for _, snippet := range mentionOSPFCases {
		ps := []domain.SearchResult{{
			PassageID:  1,
			DocumentID: "doc-v6-ospf-m",
			Domain:     "support.huawei.com",
			Snippet:    snippet,
			SHA256:     "sha-v6-2b",
		}}
		claims := eng.BuildClaimsFromPassages("camp-v6-2b", q, ps)
		for _, c := range claims {
			if c.Status == "contested" || c.Applicability == "incompatible" {
				t.Errorf("negação de BGP em contexto OSPF refutou OSPF (%q): status=%s applicability=%s",
					snippet, c.Status, c.Applicability)
			}
		}
	}
}

func TestAuditV6_NegationOfPPPoENotAttributedToBGP(t *testing.T) {
	eng := NewEngine()
	q := "Huawei NE8000 VRP v8 BGP"

	positiveBGPCases := []string{
		"Huawei NE8000 VRP v8 supports BGP and does not support PPPoE.",
		"Huawei NE8000 VRP v8 does not support PPPoE and supports BGP.",
		"Huawei NE8000 VRP v8 supports BGP, while PPPoE is not supported.",
		"PPPoE is not supported on Huawei NE8000 VRP v8, but BGP is supported.",
		"Huawei NE8000 VRP v8 supports BGP without PPPoE.",
		"Huawei NE8000 VRP v8 supports BGP, not PPPoE.",
		"O Huawei NE8000 VRP v8 suporta BGP e não suporta PPPoE.",
		"El Huawei NE8000 VRP v8 soporta BGP y no soporta PPPoE.",
	}

	for _, snippet := range positiveBGPCases {
		ps := []domain.SearchResult{{
			PassageID:  1,
			DocumentID: "doc-v6-bgp",
			Domain:     "support.huawei.com",
			Snippet:    snippet,
			SHA256:     "sha-v6-3a",
		}}
		claims := eng.BuildClaimsFromPassages("camp-v6-3a", q, ps)
		for _, c := range claims {
			if c.Status == "contested" || c.Applicability == "incompatible" {
				t.Errorf("negação de PPPoE foi atribuída a BGP (%q): status=%s applicability=%s ev=%+v",
					snippet, c.Status, c.Applicability, c.Evidences)
			}
			if c.Status != "sustained" || c.Applicability != "indeterminate" {
				t.Errorf("suporte explícito de BGP deveria ser sustained/indeterminate (%q): obteve %s/%s",
					snippet, c.Status, c.Applicability)
			}
		}
	}

	mentionBGPCases := []string{
		"Huawei NE8000 VRP v8 does not support PPPoE over BGP.",
		"PPPoE is not supported with BGP on Huawei NE8000 VRP v8.",
		"With BGP peering, PPPoE is not supported on Huawei NE8000 VRP v8.",
	}
	for _, snippet := range mentionBGPCases {
		ps := []domain.SearchResult{{
			PassageID:  1,
			DocumentID: "doc-v6-bgp-m",
			Domain:     "support.huawei.com",
			Snippet:    snippet,
			SHA256:     "sha-v6-3b",
		}}
		claims := eng.BuildClaimsFromPassages("camp-v6-3b", q, ps)
		for _, c := range claims {
			if c.Status == "contested" || c.Applicability == "incompatible" {
				t.Errorf("negação de PPPoE com menção a BGP refutou BGP (%q): status=%s applicability=%s",
					snippet, c.Status, c.Applicability)
			}
		}
	}
}

func TestAuditV6_ExplicitConflictPreservedAndNotTurnedIntoIncompatible(t *testing.T) {
	eng := NewEngine()
	q := "Huawei NE8000 VRP v8 BGP"

	// 1. Duas passagens completas contraditórias (uma suporta com escopo completo, outra refuta com escopo completo)
	passages := []domain.SearchResult{
		{
			PassageID:  1,
			DocumentID: "doc-support",
			Domain:     "support.huawei.com",
			Snippet:    "Huawei NE8000 VRP v8 supports BGP. Configure with bgp 65000 and peer 10.0.0.1 as-number 65001.",
			SHA256:     "sha-sup",
		},
		{
			PassageID:  2,
			DocumentID: "doc-refute",
			Domain:     "support.huawei.com",
			Snippet:    "Huawei NE8000 VRP v8 does not support BGP on this platform.",
			SHA256:     "sha-ref",
		},
	}

	claims := eng.BuildClaimsFromPassages("camp-v6-conflict-1", q, passages)
	if len(claims) != 1 {
		t.Fatalf("esperava 1 claim unificada para a consulta, obteve %d", len(claims))
	}
	if claims[0].Status != "contested" {
		t.Errorf("conflito explícito deve ter status 'contested', obteve %s", claims[0].Status)
	}
	if claims[0].Applicability == "incompatible" || claims[0].Applicability == "applicable" {
		t.Errorf("conflito explícito entre evidências completas NÃO pode virar %s (esperava indeterminate)", claims[0].Applicability)
	}
	if len(claims[0].Contradictions) == 0 {
		t.Errorf("esperava registro explícito de contradição em claim.Contradictions")
	}

	// 2. Conflito explícito dentro da mesma passagem
	intraConflictSnippets := []string{
		"One guide states Huawei NE8000 VRP v8 supports BGP, but another release note states Huawei NE8000 VRP v8 does not support BGP.",
		"There is conflicting documentation on whether Huawei NE8000 VRP v8 supports BGP.",
		"Há informações contraditórias se o Huawei NE8000 VRP v8 suporta BGP.",
	}
	for _, snippet := range intraConflictSnippets {
		cs := eng.BuildClaimsFromPassages("camp-v6-conflict-2", q, []domain.SearchResult{{
			PassageID:  10,
			DocumentID: "doc-intra",
			Domain:     "support.huawei.com",
			Snippet:    snippet,
			SHA256:     "sha-intra",
		}})
		for _, c := range cs {
			if c.Status != "contested" {
				t.Errorf("conflito explícito intra-passagem (%q) deve ser contested, obteve %s", snippet, c.Status)
			}
			if c.Applicability == "incompatible" || c.Applicability == "applicable" {
				t.Errorf("conflito explícito intra-passagem (%q) virou %s, esperava indeterminate", snippet, c.Applicability)
			}
		}
	}
}
