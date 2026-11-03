package evidence

import (
	"testing"

	"nie/internal/domain"
)

func TestCorroborationRequiresMatchingVendorAndFeature(t *testing.T) {
	claim := &domain.Claim{ID: "claim"}
	evidences := []domain.Evidence{
		{ID: "one", Stance: "supports", VendorMatch: true, FeatureMatch: true, Notes: "Fonte: one.example"},
		{ID: "two", Stance: "supports", VendorMatch: false, FeatureMatch: true, Notes: "Fonte: two.example"},
		{ID: "three", Stance: "supports", VendorMatch: true, FeatureMatch: false, Notes: "Fonte: three.example"},
	}
	NewEngine().ResolveClaimStatus(claim, evidences)
	if claim.Status == "corroborated" || claim.Applicability == "applicable" {
		t.Fatalf("fontes fora do escopo não corroboram a afirmação: %+v", claim)
	}

	evidences[1].VendorMatch = true
	NewEngine().ResolveClaimStatus(claim, evidences)
	if claim.Status != "corroborated" || claim.Applicability != "indeterminate" {
		t.Fatalf("duas fontes correspondentes, sem modelo e firmware, devem corroborar sem declarar aplicabilidade: %+v", claim)
	}
}
