package ai

import (
	"context"
	"strings"
	"testing"

	"nie/internal/domain"
)

func TestPlanSubquestionsDeterministic(t *testing.T) {
	svc := NewService(Config{Provider: "none"})
	subq := svc.PlanSubquestions(context.Background(), "Huawei NE8000 BNG PPPoE")
	if len(subq) < 4 {
		t.Fatalf("esperava pelo menos 4 subperguntas, obteve %d", len(subq))
	}
	for _, q := range subq {
		if !strings.Contains(q, "Huawei NE8000 BNG PPPoE") {
			t.Errorf("subpergunta não inclui query original: %s", q)
		}
	}
}

func TestGenerateInvestigationDossier(t *testing.T) {
	svc := NewService(Config{Provider: "none"})
	campaign := domain.Campaign{
		ID:    "camp-123",
		Query: "MikroTik BGP RouterOS v7",
	}
	claims := []domain.Claim{
		{
			Title:         "MikroTik BGP Connection",
			Statement:     "Configura BGP no ROS v7",
			Status:        "sustained",
			Applicability: "confirmed",
			Evidences: []domain.Evidence{
				{
					Excerpt:       "/routing bgp connection add name=p1",
					IsOfficialDoc: true,
					Notes:         "Doc oficial MikroTik",
				},
			},
		},
	}
	passages := []domain.SearchResult{
		{
			Domain:  "help.mikrotik.com",
			Snippet: "MikroTik RouterOS v7:\n/routing bgp connection add name=p1 as=65000",
		},
	}

	dossier := svc.GenerateInvestigationDossier(context.Background(), campaign, claims, passages)
	if len(dossier.Markdown) == 0 {
		t.Errorf("markdown do dossiê não pode ser vazio")
	}
	if len(dossier.CLICommands) == 0 {
		t.Errorf("esperava comandos CLI identificados no dossiê")
	}
	if dossier.CompatibilityMatrix["MikroTik"] != "RouterOS v7" {
		t.Errorf("esperava RouterOS v7 na matriz, obteve %s", dossier.CompatibilityMatrix["MikroTik"])
	}
}

func TestPlanAdaptiveLeads(t *testing.T) {
	svc := NewService(Config{Provider: "none"})
	claims := []domain.Claim{
		{
			Title:      "BGP Syntax",
			VendorName: "Huawei",
			Status:     "contested",
		},
		{
			Title:      "RADIUS Accounting",
			VendorName: "MikroTik",
			Status:     "documented",
		},
	}

	leads := svc.PlanAdaptiveLeads(context.Background(), "BGP RADIUS failover", claims)
	if len(leads) != 2 {
		t.Fatalf("esperava 2 pistas investigativas adaptativas, obteve %d", len(leads))
	}

	// Lead 1: Contradição -> deprecation/migration
	if !strings.Contains(leads[0].SearchLead, "deprecated") {
		t.Errorf("Pista de contradição deveria focar em deprecated/migration: %+v", leads[0])
	}
	// Lead 2: Documented sem firmware -> firmware compatibility
	if !strings.Contains(leads[1].SearchLead, "firmware compatibility") {
		t.Errorf("Pista de documentação sem firmware deveria focar em firmware compatibility: %+v", leads[1])
	}
}
