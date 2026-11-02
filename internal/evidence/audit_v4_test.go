package evidence

import (
	"nie/internal/domain"
	"testing"
)

// Auditing whether substring matching erroneously invents an additional feature.
func TestAuditV4_IncidentalWordWhatMustNotRequireHA(t *testing.T) {
	req := ParseRequirements("What is Huawei NE8000 VRP v8 BGP?")
	for _, f := range req.Features {
		if f == "HA" {
			t.Fatalf("HA detectada no inglês WHAT sem ser pedida; features=%v", req.Features)
		}
	}
}

// Explicit negative assertion in an official excerpt must not be treated as support.
func TestAuditV4_ExplicitDoesNotSupportCannotBecomeApplicable(t *testing.T) {
	e := NewEngine()
	ps := []domain.SearchResult{{PassageID: 1, DocumentID: "111", Domain: "support.huawei.com", Snippet: "Huawei NE8000 VRP v8 does not support BGP on this platform.", SHA256: "x"}}
	cs := e.BuildClaimsFromPassages("camp", "Huawei NE8000 VRP v8 BGP", ps)
	for _, c := range cs {
		if c.Status == "sustained" || c.Applicability == "applicable" {
			t.Fatalf("negative assertion promoted: status=%s applicability=%s; evidence=%+v", c.Status, c.Applicability, c.Evidences)
		}
	}
}

// A refutation should not be equated with demonstrated incompatibility, which needs scope matching.
func TestAuditV4_UnscopedRefutationNotAutomaticallyIncompatible(t *testing.T) {
	e := NewEngine()
	c := &domain.Claim{Title: "BGP VRP v8 no NE8000"}
	e.ResolveClaimStatus(c, []domain.Evidence{{Stance: "refutes", VendorMatch: false, ModelMatch: false, FirmwareMatch: false, FeatureMatch: false}})
	if c.Applicability == "incompatible" {
		t.Fatalf("unscoped refutation promoted to incompatible")
	}
}
