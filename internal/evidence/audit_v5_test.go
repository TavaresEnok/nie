package evidence

import (
	"nie/internal/domain"
	"testing"
)

// Testes adversariais adicionados pela auditoria; nao fazem parte da suite original.
func TestAuditV5_InferredFirmwareFromModelCannotValidateRelease(t *testing.T) {
	q := "Huawei NE8000 VRP v8 BGP"
	p := domain.SearchResult{PassageID: 1, DocumentID: "doc1", Domain: "support.huawei.com", Snippet: "Huawei NE8000 supports BGP. Configure with bgp 65000.", SHA256: "sha"}
	cs := NewEngine().BuildClaimsFromPassages("camp", q, []domain.SearchResult{p})
	for _, c := range cs {
		if c.Applicability == "applicable" {
			t.Errorf("sem firmware VRP v8 no trecho, mas validado: %s/%s; evidencia: %+v", c.Status, c.Applicability, c.Evidences[0])
		}
	}
}
func TestAuditV5_NegativeAboutOtherFeatureCannotRejectBGP(t *testing.T) {
	q := "Huawei NE8000 VRP v8 BGP"
	p := domain.SearchResult{PassageID: 1, DocumentID: "doc2", Domain: "support.huawei.com", Snippet: "Huawei NE8000 VRP v8 supports BGP. The Telnet service is not supported in this release.", SHA256: "sha"}
	cs := NewEngine().BuildClaimsFromPassages("camp", q, []domain.SearchResult{p})
	for _, c := range cs {
		if c.Applicability == "incompatible" {
			t.Errorf("negacao do Telnet refutou BGP: %s/%s; stance=%s", c.Status, c.Applicability, c.Evidences[0].Stance)
		}
	}
}
func TestAuditV5_SeparateFeatureMentionCannotValidateCombinedFunctionality(t *testing.T) {
	q := "Huawei NE8000 VRP v8 BGP PPPoE"
	p := domain.SearchResult{PassageID: 1, DocumentID: "doc3", Domain: "support.huawei.com", Snippet: "Huawei NE8000 VRP v8 supports BGP. PPPoE configuration guide listed under related topics.", SHA256: "sha"}
	cs := NewEngine().BuildClaimsFromPassages("camp", q, []domain.SearchResult{p})
	for _, c := range cs {
		if c.Applicability == "applicable" {
			t.Errorf("menção PPPoE tratada como suporte conjunto: %s/%s; evidência=%+v", c.Status, c.Applicability, c.Evidences[0])
		}
	}
}
