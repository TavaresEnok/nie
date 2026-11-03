package evidence

import (
	"strings"
	"testing"

	"nie/internal/domain"
)

// As 20 perguntas do arquivo mestre são casos de avaliação, não resultados de
// uma campanha pública. Aqui cada uma recebe um texto que não afirma o fato
// (pergunta ou hipótese) e o motor não pode sustentá-lo. A passagem direta
// da primeira pergunta sustenta o átomo BGP e deixa a aplicabilidade em aberto.
// O caso fora de telecom não herda fabricante de rede.
func TestBenchmarkV7_TwentyQuestionsDoNotCertifyFromNonAssertions(t *testing.T) {
	questions := []string{
		"Huawei NE8000 VRP v8 BGP PPPoE",
		"Huawei NE8000 VRP v8 BNG",
		"Huawei NE8000 VRP v8 RADIUS",
		"Huawei NE8000 VRP v8 BGP",
		"MikroTik RouterOS v7 BGP",
		"MikroTik RouterOS v7 PPPoE",
		"MikroTik RouterOS v7 PBR",
		"Cisco IOS-XR PBR",
		"Datacom BGP",
		"Datacom SNMP",
		"Huawei MA5800 GPON",
		"ZTE C600 GPON",
		"FiberHome GPON",
		"VSOL VLAN",
		"Huawei VLAN",
		"SonicWall PBR",
		"Hillstone BGP",
		"Huawei MPLS BGP",
		"Huawei BGP",
		"Huawei NE8000 VRP v8",
	}
	if len(questions) != 20 {
		t.Fatalf("bateria = %d", len(questions))
	}
	eng := NewEngine()
	for _, query := range questions {
		passage := "Training question: does the document show that " + query + " is supported?"
		claims := eng.BuildClaimsFromPassages("bench", query, []domain.SearchResult{{
			PassageID: 1, DocumentID: "bench-doc", Domain: "support.huawei.com", Snippet: passage, SHA256: "bench",
		}})
		for _, claim := range claims {
			if claim.Status == "sustained" || claim.Status == "corroborated" || claim.Applicability == "applicable" {
				t.Errorf("pergunta da bateria certificada (%s): %s/%s", query, claim.Status, claim.Applicability)
			}
		}
	}
}

func TestBenchmarkV7_DirectAtomIsSustainedWithoutApplicability(t *testing.T) {
	eng := NewEngine()
	claims := eng.BuildClaimsFromPassages("bench-direct", "Huawei NE8000 VRP v8 BGP PPPoE", []domain.SearchResult{{
		PassageID: 1, DocumentID: "bench-direct", Domain: "support.huawei.com",
		Snippet: "Huawei NE8000 VRP v8 supports BGP. PPPoE session continuity is a separate mechanism.",
		SHA256:  "bench-direct",
	}})
	var bgp, compound bool
	for _, claim := range claims {
		if claim.Applicability == "applicable" {
			t.Fatalf("bateria marcou applicable sem laboratório: %+v", claim)
		}
		if strings.Contains(claim.Title, "BGP") && !strings.Contains(claim.Title, "PPPOE") && claim.Status == "sustained" {
			bgp = true
		}
		if strings.Contains(claim.Statement, "BGP") && strings.Contains(claim.Statement, "PPPOE") && claim.Status == "sustained" {
			compound = true
		}
	}
	if !bgp {
		t.Fatalf("átomo BGP não sustentado: %+v", claims)
	}
	if compound {
		t.Fatalf("a proposição composta foi sustentada só com BGP: %+v", claims)
	}
}

func TestBenchmarkV7_NonTelecomSubjectDoesNotInheritNetworkVendor(t *testing.T) {
	eng := NewEngine()
	query := "CPython asyncio library timeout history"
	claims := eng.BuildClaimsFromPassages("bench-py", query, []domain.SearchResult{{
		PassageID: 1, DocumentID: "py", Domain: "docs.python.org",
		Snippet: "CPython 3.11 documents asyncio.wait_for and the timeout parameter.",
		SHA256:  "py",
	}})
	for _, claim := range claims {
		if claim.VendorName != "" || claim.Status == "sustained" || claim.Applicability == "applicable" {
			t.Fatalf("tema fora de telecom herdou certificação de rede: %+v", claim)
		}
	}
}
