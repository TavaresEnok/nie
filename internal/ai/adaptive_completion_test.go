package ai

import (
	"context"
	"nie/internal/domain"
	"testing"
)

func TestAdaptiveLeadsDoNotInventGapAfterScopedSupport(t *testing.T) {
	svc := NewService(Config{Provider: "none"})
	leads := svc.PlanAdaptiveLeads(context.Background(), "Huawei NE8000 BGP", []domain.Claim{{Status: "sustained", Applicability: "applicable"}})
	if len(leads) != 0 {
		t.Fatalf("afirmação sustentada gerou lacuna inexistente: %+v", leads)
	}
	if len(svc.PlanAdaptiveLeads(context.Background(), "new subject", nil)) == 0 {
		t.Fatal("campanha sem evidências precisa de pista exploratória")
	}
}
