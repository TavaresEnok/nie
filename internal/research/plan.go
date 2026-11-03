package research

import (
	"context"
	"strings"
	"unicode"

	"nie/internal/ai"
	"nie/internal/domain"
	"nie/internal/store"
)

// Question é uma subpergunta estável da campanha. O ciclo automático a
// persiste e só a envia à descoberta quando as lacunas dos claims não geram
// outra consulta. Status addressed significa que os termos distintivos
// apareceram no mesmo trecho; não certifica que a subpergunta foi respondida.
type Question struct {
	Ordinal      int
	Text         string
	Origin       string
	Status       string
	EvidenceNote string
}

// Lead é uma consulta proposta para a rodada, ainda sem provedor.
type Lead struct {
	Query  string
	Reason string
}

// RoundPlan é a decisão determinística de uma avaliação. O LLM fica de fora
// deste caminho para a fila continuar quando não há modelo.
type RoundPlan struct {
	Decision  string
	Reason    string
	Leads     []Lead
	Questions []Question
}

// PlanRound decompõe o objetivo e escolhe o que buscar agora.
// searched contém consultas de descoberta já existentes, normalizadas.
func PlanRound(ctx context.Context, query string, claims []domain.Claim, passages []domain.SearchResult, searched map[string]bool) RoundPlan {
	svc := ai.NewService(ai.Config{Provider: "none"})
	subquestions := svc.PlanSubquestions(ctx, query)
	gaps := svc.PlanAdaptiveLeads(ctx, query, claims)
	if searched == nil {
		searched = map[string]bool{}
	}

	plan := RoundPlan{Decision: "continue", Reason: "evidence_gaps"}
	anyAddressed := false
	anyOpen := false
	for i, text := range subquestions {
		q := Question{Ordinal: i + 1, Text: text, Origin: "planner", Status: "open", EvidenceNote: "ainda não enviada à descoberta"}
		switch {
		case coveredByPassage(text, query, passages):
			q.Status = "addressed"
			q.EvidenceNote = "termos distintivos aparecem no mesmo trecho; isso não certifica a resposta"
			anyAddressed = true
		case searched[store.NormalizeQuery(text)]:
			q.Status = "queued"
			q.EvidenceNote = "consulta já encaminhada à descoberta; a cobertura do trecho não foi reconhecida"
		default:
			anyOpen = true
		}
		plan.Questions = append(plan.Questions, q)
	}

	for _, gap := range gaps {
		plan.Leads = append(plan.Leads, Lead{Query: gap.SearchLead, Reason: gap.EvidenceGap})
	}
	if len(plan.Leads) == 0 {
		plan.Reason = "open_subquestions"
		for _, q := range plan.Questions {
			if q.Status == "open" {
				plan.Leads = append(plan.Leads, Lead{Query: q.Text, Reason: "subpergunta planejada ainda sem descoberta"})
			}
		}
	}
	if len(plan.Leads) == 0 {
		plan.Decision = "stop"
		switch {
		case len(plan.Questions) > 0 && !anyOpen && anyAddressed && allStatus(plan.Questions, "addressed"):
			plan.Reason = "subquestions_addressed"
		case len(plan.Questions) > 0 && !anyOpen:
			plan.Reason = "subquestions_searched"
		default:
			plan.Reason = "no_detected_gaps"
		}
	}
	return plan
}

func allStatus(questions []Question, status string) bool {
	if len(questions) == 0 {
		return false
	}
	for _, q := range questions {
		if q.Status != status {
			return false
		}
	}
	return true
}

func coveredByPassage(question, original string, passages []domain.SearchResult) bool {
	distinctive := distinctiveTokens(question, original)
	if len(distinctive) < 2 {
		return false
	}
	for _, passage := range passages {
		hay := strings.ToLower(passage.Snippet)
		ok := true
		for _, token := range distinctive {
			if !strings.Contains(hay, token) {
				ok = false
				break
			}
		}
		if ok {
			return true
		}
	}
	return false
}

func distinctiveTokens(question, original string) []string {
	known := map[string]bool{}
	for _, token := range tokens(original) {
		known[token] = true
	}
	seen := map[string]bool{}
	var out []string
	for _, token := range tokens(question) {
		if known[token] || len(token) < 4 || seen[token] {
			continue
		}
		seen[token] = true
		out = append(out, token)
	}
	return out
}

func tokens(value string) []string {
	return strings.FieldsFunc(strings.ToLower(value), func(r rune) bool {
		return !unicode.IsLetter(r) && !unicode.IsDigit(r) && r != '-'
	})
}
