package research

import (
	"context"
	"testing"

	"nie/internal/domain"
	"nie/internal/store"
)

func TestPlanRoundDefersSubquestionsWhileClaimGapExists(t *testing.T) {
	claims := []domain.Claim{{Title: "BGP", VendorName: "Huawei", Status: "documented"}}
	plan := PlanRound(context.Background(), "Huawei NE8000 VRP v8 BGP", claims, nil, nil)
	if plan.Decision != "continue" || plan.Reason != "evidence_gaps" {
		t.Fatalf("plano=%+v", plan)
	}
	if len(plan.Questions) < 4 {
		t.Fatalf("subperguntas=%d", len(plan.Questions))
	}
	for _, q := range plan.Questions {
		if q.Status != "open" {
			t.Fatalf("subpergunta não deveria ser buscada junto da lacuna: %+v", q)
		}
	}
	for _, lead := range plan.Leads {
		if lead.Query == plan.Questions[0].Text {
			t.Fatal("subpergunta entrou na mesma rodada da lacuna de claim")
		}
	}
}

func TestPlanRoundSearchesOpenSubquestionsAfterGapsClose(t *testing.T) {
	claims := []domain.Claim{{Title: "BGP", Status: "sustained"}}
	plan := PlanRound(context.Background(), "Huawei NE8000 VRP v8 BGP", claims, []domain.SearchResult{{Snippet: "Huawei NE8000 VRP v8 supports BGP."}}, nil)
	if plan.Reason != "open_subquestions" || len(plan.Leads) < 4 {
		t.Fatalf("plano=%+v", plan)
	}
}

func TestPlanRoundAddressesSubquestionOnlyWhenDistinctiveTermsSharePassage(t *testing.T) {
	query := "Huawei NE8000 VRP v8 BGP"
	open := PlanRound(context.Background(), query, nil, nil, nil)
	target := open.Questions[0].Text
	partial := PlanRound(context.Background(), query, []domain.Claim{{Title: "BGP", Status: "sustained"}}, []domain.SearchResult{{Snippet: "display"}}, nil)
	if partial.Questions[0].Status == "addressed" {
		t.Fatal("um único termo distintivo não cobre a subpergunta")
	}
	claims := []domain.Claim{{Title: "BGP", Status: "sustained"}}
	full := PlanRound(context.Background(), query, claims, []domain.SearchResult{{Snippet: target}}, map[string]bool{store.NormalizeQuery(target): true})
	if full.Questions[0].Status != "addressed" {
		t.Fatalf("status=%s", full.Questions[0].Status)
	}
	for _, lead := range full.Leads {
		if lead.Query == target {
			t.Fatal("subpergunta coberta foi proposta de novo")
		}
	}
}

func TestPlanRoundStopsAfterSubquestionsWereQueued(t *testing.T) {
	query := "biblioteca open source historico"
	claims := []domain.Claim{{Title: "historico", Status: "sustained"}}
	first := PlanRound(context.Background(), query, claims, nil, nil)
	searched := map[string]bool{}
	for _, q := range first.Questions {
		searched[store.NormalizeQuery(q.Text)] = true
	}
	second := PlanRound(context.Background(), query, claims, nil, searched)
	if second.Decision != "stop" || second.Reason != "subquestions_searched" {
		t.Fatalf("plano=%+v", second)
	}
}
