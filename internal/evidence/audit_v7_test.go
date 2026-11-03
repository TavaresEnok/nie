package evidence

import (
	"strings"
	"testing"

	"nie/internal/domain"
)

// Auditoria V7: pergunta, hipótese, frase declarada falsa e alegação não
// verificada não podem virar sustained/applicable. Os textos são sintéticos.
func TestAuditV7_AssertionsAreNotProof(t *testing.T) {
	eng := NewEngine()
	query := "Huawei NE8000 VRP v8 BGP"
	cases := []string{
		"Training question: Does Huawei NE8000 VRP v8 support BGP?",
		"Exercise: if Huawei NE8000 VRP v8 supports BGP, which route would be preferred?",
		"The statement 'Huawei NE8000 VRP v8 supports BGP' is false.",
		"It is an unverified claim that Huawei NE8000 VRP v8 supports BGP.",
		"Example configuration explicitly described as false: Huawei NE8000 VRP v8 supports BGP.",
		"O Huawei NE8000 VRP v8 suporta BGP?",
		"Se o Huawei NE8000 VRP v8 suportar BGP, qual rota seria preferida?",
		"A afirmação 'Huawei NE8000 VRP v8 suporta BGP' é falsa.",
		"É uma alegação não verificada que o Huawei NE8000 VRP v8 suporta BGP.",
		"¿Huawei NE8000 VRP v8 soporta BGP?",
		"Si Huawei NE8000 VRP v8 soporta BGP, ¿qué ruta se prefiere?",
		"La afirmación 'Huawei NE8000 VRP v8 soporta BGP' es falsa.",
		"Es una alegación no verificada que Huawei NE8000 VRP v8 soporta BGP.",
	}
	for _, snippet := range cases {
		claims := eng.BuildClaimsFromPassages("camp-v7", query, []domain.SearchResult{{
			PassageID: 1, DocumentID: "doc-v7", Domain: "support.huawei.com", Snippet: snippet, SHA256: "sha-v7",
		}})
		for _, c := range claims {
			if c.Status == "sustained" || c.Status == "corroborated" || c.Applicability == "applicable" {
				t.Errorf("ato não assertivo promovido a prova (%q): %s/%s ev=%+v", snippet, c.Status, c.Applicability, c.Evidences)
			}
			for _, ev := range c.Evidences {
				if ev.Stance == "supports" {
					t.Errorf("ato não assertivo recebeu supports (%q): %+v", snippet, ev)
				}
			}
		}
	}
}

func TestAuditV7_LabValidationRequiredForApplicable(t *testing.T) {
	eng := NewEngine()
	claim := &domain.Claim{Title: "BGP", OperationalValidation: "lab_validated"}
	eng.ResolveClaimStatus(claim, []domain.Evidence{{
		ID: "e1", Stance: "supports", IsOfficialDoc: true,
		VendorMatch: true, ModelMatch: true, FirmwareMatch: true, FeatureMatch: true,
		Notes: "Fonte: support.huawei.com",
	}})
	if claim.Status != "sustained" || claim.Applicability != "applicable" {
		t.Fatalf("laboratório no escopo oficial: %s/%s", claim.Status, claim.Applicability)
	}
}

func TestAuditV7_DirectAssertionStillSustains(t *testing.T) {
	eng := NewEngine()
	claims := eng.BuildClaimsFromPassages("camp-v7-ok", "Huawei NE8000 VRP v8 BGP", []domain.SearchResult{{
		PassageID: 1, DocumentID: "doc-ok", Domain: "support.huawei.com",
		Snippet: "Huawei NE8000 VRP v8 supports BGP.", SHA256: "sha-ok",
	}})
	if len(claims) != 1 {
		t.Fatalf("claims = %d", len(claims))
	}
	if claims[0].Status != "sustained" || claims[0].Applicability != "indeterminate" {
		t.Fatalf("afirmação direta sem laboratório: %s/%s", claims[0].Status, claims[0].Applicability)
	}
	if strings.Contains(claims[0].Statement, "Procedimento e diretrizes documentadas") {
		t.Fatalf("enunciado ainda trata a consulta como procedimento documentado: %s", claims[0].Statement)
	}
}

func TestAuditV7_EvidenceUsesCanonicalText(t *testing.T) {
	eng := NewEngine()
	raw := "Huawei NE8000 VRP v8 supports BGP."
	start, end := 12, 48
	claims := eng.BuildClaimsFromPassages("camp-v7-text", "Huawei NE8000 VRP v8 BGP", []domain.SearchResult{{
		PassageID: 7, DocumentID: "doc-text", Domain: "support.huawei.com", SHA256: "sha-text",
		Text: raw, Snippet: "… <mark>Huawei</mark> NE8000 …", StartOffset: &start, EndOffset: &end,
	}})
	if len(claims) != 1 || len(claims[0].Evidences) != 1 {
		t.Fatalf("claims = %+v", claims)
	}
	ev := claims[0].Evidences[0]
	if ev.Excerpt != raw {
		t.Fatalf("excerpt = %q", ev.Excerpt)
	}
	if strings.Contains(ev.Excerpt, "<mark>") || strings.Contains(ev.Excerpt, "…") {
		t.Fatalf("excerpt de apresentação: %q", ev.Excerpt)
	}
	if ev.SourceSpan != "doc:doc-text:p:7:bytes:12-48" {
		t.Fatalf("span = %s", ev.SourceSpan)
	}
	if claims[0].Status != "sustained" {
		t.Fatalf("status = %s", claims[0].Status)
	}
}

func TestAuditV7_CompoundStatementNamesEachFeature(t *testing.T) {
	eng := NewEngine()
	claims := eng.BuildClaimsFromPassages("camp-v7-compound", "Huawei NE8000 VRP v8 BGP PPPoE RADIUS", []domain.SearchResult{{
		PassageID: 1, DocumentID: "doc-part", Domain: "support.huawei.com",
		Snippet: "Huawei NE8000 VRP v8 supports BGP.", SHA256: "sha-part",
	}})
	var compound *domain.Claim
	for i := range claims {
		if strings.Contains(claims[i].Statement, "BGP") && strings.Contains(claims[i].Statement, "PPPOE") && strings.Contains(claims[i].Statement, "RADIUS") {
			compound = &claims[i]
		}
		if claims[i].Applicability == "applicable" {
			t.Fatalf("menção só de BGP certificou aplicabilidade: %+v", claims[i])
		}
	}
	if compound == nil {
		t.Fatalf("proposição composta ausente: %+v", claims)
	}
}
