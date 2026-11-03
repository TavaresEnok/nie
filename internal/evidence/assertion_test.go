package evidence

import (
	"strings"
	"testing"

	"nie/internal/domain"
)

func TestSplitClausesKeepsVersionsAndAddresses(t *testing.T) {
	got := splitClauses("Huawei NE8000 VRP v8.2 supports BGP. Use peer 10.0.0.2 as-number 65001; Telnet is not supported, but SSH is.")
	want := []string{
		"Huawei NE8000 VRP v8.2 supports BGP",
		"Use peer 10.0.0.2 as-number 65001",
		"Telnet is not supported",
		"SSH is",
	}
	if len(got) != len(want) {
		t.Fatalf("orações = %q, quer %q", got, want)
	}
	for i := range want {
		if got[i] != want[i] {
			t.Errorf("oração %d = %q, quer %q", i, got[i], want[i])
		}
	}
}

func TestFirmwareMustBeExplicit(t *testing.T) {
	cases := []struct {
		text, fw string
		want     bool
	}{
		{"huawei ne8000 supports bgp", "VRP v8", false}, // modelo não implica release
		{"huawei ne8000 vrp v8 supports bgp", "VRP v8", true},
		{"ne8000 running vrpv8", "VRP v8", true},
		{"ne8000 v800r021c10 release notes", "VRP v8", true},
		{"routeros 7.12 bgp", "RouterOS v7", true},
		{"routeros v6 bgp peer", "RouterOS v7", false},
	}
	for _, tc := range cases {
		if got := mentionsFirmware(tc.text, tc.fw); got != tc.want {
			t.Errorf("mentionsFirmware(%q, %q) = %v, quer %v", tc.text, tc.fw, got, tc.want)
		}
	}
	if !mentionsConflictingFirmware("routeros v6 bgp peer", "RouterOS v7") {
		t.Error("RouterOS v6 explícito deve conflitar com pedido de v7")
	}
	if mentionsConflictingFirmware("migrating from routeros v6 to routeros v7", "RouterOS v7") {
		t.Error("trecho que cita a release pedida não é conflito")
	}
}

// Conjunto rotulado de frases (auditoria V5): stance esperado da evidência para
// cada trecho, todos de domínio oficial. Frases artificiais, não afirmações
// sobre equipamentos reais.
func TestLabeledStanceDataset(t *testing.T) {
	cases := []struct {
		query, snippet, wantStance string
	}{
		// afirmações positivas completas
		{"Huawei NE8000 VRP v8 BGP", "Huawei NE8000 VRP v8 supports BGP.", "supports"},
		{"Huawei NE8000 VRP v8 BGP PPPoE", "Huawei NE8000 VRP v8 supports BGP and PPPoE on the same chassis.", "supports"},
		{"Huawei NE8000 VRP v8 BGP", "Huawei NE8000 VRP v8: bgp 65000 peer 10.0.0.2 as-number 65001", "supports"},
		// negações do recurso pedido, em três idiomas
		{"Huawei NE8000 VRP v8 BGP", "BGP is not supported on Huawei NE8000 VRP v8.", "refutes"},
		{"Huawei NE8000 VRP v8 BGP", "O Huawei NE8000 VRP v8 não suporta BGP.", "refutes"},
		{"Huawei NE8000 VRP v8 BGP", "El Huawei NE8000 VRP v8 no soporta BGP.", "refutes"},
		{"Huawei NE8000 VRP v8 BGP", "Huawei NE8000 VRP v8 supports OSPF, but BGP is not supported.", "refutes"},
		// negação de outro recurso não afeta o pedido
		{"Huawei NE8000 VRP v8 BGP", "Huawei NE8000 VRP v8 supports BGP. The Telnet service is not supported in this release.", "supports"},
		{"Huawei NE8000 VRP v8 BGP", "Huawei NE8000 VRP v8 does not support Telnet.", "mentions"},
		// recursos em orações separadas não provam funcionamento conjunto
		{"Huawei NE8000 VRP v8 BGP PPPoE", "Huawei NE8000 VRP v8 supports BGP. PPPoE configuration guide listed under related topics.", "mentions"},
		{"Huawei NE8000 VRP v8 BGP PPPoE", "Huawei NE8000 VRP v8 supports BGP. Huawei NE8000 VRP v8 supports PPPoE.", "mentions"},
		// condicionais, parciais e listas
		{"Huawei NE8000 VRP v8 BGP", "Huawei NE8000 VRP v8 supports BGP only with the advanced license.", "mentions"},
		{"Huawei NE8000 VRP v8 BGP", "Index of documentation: Huawei NE8000 VRP v8 BGP.", "mentions"},
		// outra release ou outro fabricante: nem prova nem refuta
		{"Huawei NE8000 VRP v8 BGP", "Huawei NE8000 VRP v5 supports BGP.", "mentions"},
		{"Huawei NE8000 VRP v8 BGP", "Cisco ASR9000 supports BGP.", "mentions"},
	}
	for _, tc := range cases {
		ps := []domain.SearchResult{{PassageID: 1, DocumentID: "d", Domain: "support.huawei.com", Snippet: tc.snippet, SHA256: "h"}}
		cs := NewEngine().BuildClaimsFromPassages("camp", tc.query, ps)
		if len(cs) == 0 {
			t.Fatalf("sem claim para %q", tc.snippet)
		}
		if strings.Contains(tc.query, "PPPoE") {
			var joint *domain.Claim
			for i := range cs {
				if strings.Contains(cs[i].Statement, "BGP") && strings.Contains(cs[i].Statement, "PPPOE") {
					joint = &cs[i]
				}
			}
			if joint == nil {
				t.Fatalf("sem proposição conjunta para %q", tc.snippet)
			}
			got := "mentions"
			if joint.Status == "sustained" || joint.Status == "corroborated" {
				got = "supports"
			}
			if got != tc.wantStance {
				t.Errorf("%q / %q: conjunto = %s, quer %s", tc.query, tc.snippet, got, tc.wantStance)
			}
			continue
		}
		if len(cs[0].Evidences) == 0 {
			t.Fatalf("sem evidência para %q", tc.snippet)
		}
		if got := cs[0].Evidences[0].Stance; got != tc.wantStance {
			t.Errorf("%q / %q: stance = %s, quer %s", tc.query, tc.snippet, got, tc.wantStance)
		}
	}
}
