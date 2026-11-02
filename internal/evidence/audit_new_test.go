package evidence

import (
	"testing"

	"nie/internal/domain"
)

// Testes adversariais da terceira auditoria independente (02/11/2026).

// Duas fontes que apenas mencionam o assunto não sustentam a afirmação.
func TestNewAudit_MentionsOnlyMustNotCorroborate(t *testing.T) {
	e := NewEngine()
	c := &domain.Claim{Title: "Suporte PPPoE"}
	a, b := "distinct-a", "distinct-b"
	e.ResolveClaimStatus(c, []domain.Evidence{
		{ID: "e1", Stance: "mentions", ClusterID: &a, Notes: "Fonte: docs-a.example"},
		{ID: "e2", Stance: "mentions", ClusterID: &b, Notes: "Fonte: docs-b.example"},
	})
	if c.Status == "corroborated" || c.Applicability == "applicable" {
		t.Fatalf("mentions-only promoted to %s/%s", c.Status, c.Applicability)
	}
}

// Documento oficial genérico + texto comunitário com flags verdadeiras não
// podem ser fundidos em uma prova oficial específica.
func TestNewAudit_OfficialAndTechMatchMustBeSameEvidence(t *testing.T) {
	e := NewEngine()
	c := &domain.Claim{Title: "Suporte v8"}
	a, b := "official-generic", "community-specific"
	e.ResolveClaimStatus(c, []domain.Evidence{
		{ID: "e1", Stance: "supports", ClusterID: &a, IsOfficialDoc: true, Notes: "Fonte: docs.vendor.example"},
		{ID: "e2", Stance: "supports", ClusterID: &b, IsOfficialDoc: false, VendorMatch: true, ModelMatch: true, FirmwareMatch: true, FeatureMatch: true, Notes: "Fonte: blog.example"},
	})
	if c.Status == "sustained" || c.Applicability == "applicable" {
		t.Fatalf("cross-source evidence fusion promoted to %s/%s", c.Status, c.Applicability)
	}
}

// Lista de assuntos em página de índice não é prova de suporte.
func TestNewAudit_KeywordListNotProofOfFeature(t *testing.T) {
	e := NewEngine()
	q := "Huawei NE8000 VRP v8 BGP"
	ps := []domain.SearchResult{{PassageID: 101, DocumentID: "doc1", Domain: "support.huawei.com",
		Snippet: "Index of documentation: Huawei NE8000 VRP v8 BGP (title only; no configuration or compatibility statements on this page).", SHA256: "hash"}}
	cs := e.BuildClaimsFromPassages("camp", q, ps)
	for _, c := range cs {
		if c.Applicability == "applicable" {
			t.Fatalf("index-only keyword match marked applicable: status=%s; claim=%s", c.Status, c.Title)
		}
	}
}
