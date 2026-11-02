package evidence

import (
	"testing"

	"nie/internal/domain"
)

const negationQuery = "Huawei NE8000 VRP v8 BGP"

func claimsFor(t *testing.T, snippet string) []domain.Claim {
	t.Helper()
	ps := []domain.SearchResult{{PassageID: 1, DocumentID: "d1", Domain: "support.huawei.com", Snippet: snippet, SHA256: "h"}}
	cs := NewEngine().BuildClaimsFromPassages("camp", negationQuery, ps)
	if len(cs) == 0 {
		t.Fatalf("nenhum claim para %q", snippet)
	}
	return cs
}

// Negações explícitas em PT/EN/ES refutam e nunca viram suporte comprovado.
func TestNegatedSupportIsRefutationInAllLanguages(t *testing.T) {
	for _, snippet := range []string{
		"Huawei NE8000 VRP v8 does not support BGP on this platform.",
		"Huawei NE8000 VRP v8 doesn't support BGP in this release.",
		"BGP is not supported on Huawei NE8000 VRP v8.",
		"BGP cannot be configured on Huawei NE8000 VRP v8.",
		"Huawei NE8000 VRP v8 não suporta BGP nesta plataforma.",
		"No Huawei NE8000 VRP v8 o BGP não está disponível.",
		"El Huawei NE8000 VRP v8 no soporta BGP en esta plataforma.",
	} {
		for _, c := range claimsFor(t, snippet) {
			if c.Status == "sustained" || c.Status == "corroborated" || c.Applicability == "applicable" {
				t.Errorf("negação virou suporte (%q): %s/%s", snippet, c.Status, c.Applicability)
			}
			if c.Status != "contested" || c.Applicability != "incompatible" {
				t.Errorf("negação com escopo completo deve contestar e incompatibilizar (%q): %s/%s", snippet, c.Status, c.Applicability)
			}
		}
	}
}

// Enunciados condicionais ou parciais não sustentam o recurso por inteiro.
func TestQualifiedSupportIsNotProof(t *testing.T) {
	for _, snippet := range []string{
		"Huawei NE8000 VRP v8 supports BGP only on the M8 chassis.",
		"Huawei NE8000 VRP v8 supports BGP, except for the multihop mode.",
		"Huawei NE8000 VRP v8 supports BGP provided that the license is installed.",
		"O Huawei NE8000 VRP v8 suporta BGP apenas no chassi M8.",
		"El Huawei NE8000 VRP v8 admite BGP solo en el chasis M8.",
	} {
		for _, c := range claimsFor(t, snippet) {
			if c.Status == "sustained" || c.Applicability == "applicable" {
				t.Errorf("suporte condicional virou prova (%q): %s/%s", snippet, c.Status, c.Applicability)
			}
		}
	}
}

// Controle: a afirmação positiva e completa continua sendo aceita.
func TestPlainAffirmativeSupportStillSustained(t *testing.T) {
	for _, c := range claimsFor(t, "Huawei NE8000 VRP v8 supports BGP. Configure it with: bgp 65000 and peer 10.0.0.2 as-number 65001") {
		if c.Status != "sustained" || c.Applicability != "applicable" {
			t.Errorf("afirmação positiva oficial e completa deve ser sustained/applicable, obteve %s/%s", c.Status, c.Applicability)
		}
	}
}

// Negação sobre outro recurso ou outro release não contesta o cenário pedido.
func TestRefutationOutsideScopeDoesNotContest(t *testing.T) {
	for _, snippet := range []string{
		"Huawei NE8000 VRP v8 does not support PPPoE on this platform.",
		"Huawei NE8000 VRP v5 does not support BGP.",
		"Cisco ASR9000 does not support BGP flowspec in this image.",
	} {
		for _, c := range claimsFor(t, snippet) {
			if c.Status == "contested" || c.Applicability == "incompatible" {
				t.Errorf("refutação fora de escopo contestou o cenário (%q): %s/%s", snippet, c.Status, c.Applicability)
			}
		}
	}
}

func TestHasTokenUsesWordBoundaries(t *testing.T) {
	cases := []struct {
		text, tok string
		want      bool
	}{
		{"what is bgp", "ha", false},
		{"há suporte", "ha", false},
		{"bgp4 peering", "bgp", true},
		{"vlans on trunk", "vlan", true},
		{"oracle database", "acl", false},
		{"pppoe-server", "pppoe", true},
		{"ospfv3", "ospf", true},
		{"the bgpx tool", "bgp", false},
	}
	for _, tc := range cases {
		if got := hasToken(tc.text, tc.tok); got != tc.want {
			t.Errorf("hasToken(%q, %q) = %v, quer %v", tc.text, tc.tok, got, tc.want)
		}
	}
}

func TestParseRequirementsHAOnlyWhenRequested(t *testing.T) {
	hasFeature := func(q, feature string) bool {
		for _, f := range ParseRequirements(q).Features {
			if f == feature {
				return true
			}
		}
		return false
	}
	for q, want := range map[string]bool{
		"What is Huawei NE8000 VRP v8 BGP?":   false,
		"Há suporte a BGP no NE8000?":         false,
		"how do I configure ha on NE8000":     false,
		"BGP HA on Huawei NE8000":             true,
		"NE8000 BGP high availability":        true,
		"NE8000 BGP com alta disponibilidade": true,
	} {
		if got := hasFeature(q, "HA"); got != want {
			t.Errorf("HA em %q: %v, quer %v", q, got, want)
		}
	}
	// "acl" dentro de outra palavra não é o recurso ACL; como termo inteiro é.
	if hasFeature("Oracle database on a Cisco ASR9000", "ACL") {
		t.Error("ACL não deve casar dentro de 'Oracle'")
	}
	if !hasFeature("Cisco ASR9000 ACL for BGP", "ACL") {
		t.Error("ACL como termo inteiro deve ser reconhecida")
	}
}
