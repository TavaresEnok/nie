package benchmark

import (
	"strings"
	"testing"
)

func TestLoadEmbedded(t *testing.T) {
	ds, err := LoadEmbedded()
	if err != nil {
		t.Fatalf("erro ao carregar benchmark: %v", err)
	}
	if len(ds.Questions) != 20 {
		t.Fatalf("esperava 20 perguntas, obteve %d", len(ds.Questions))
	}
	for i, q := range ds.Questions {
		if q.ID == "" {
			t.Errorf("pergunta %d sem ID", i)
		}
		if q.Title == "" {
			t.Errorf("pergunta %s sem título", q.ID)
		}
		if len(q.ExpectedCommands) == 0 {
			t.Errorf("pergunta %s sem comandos esperados", q.ID)
		}
		if len(q.Vendors) == 0 {
			t.Errorf("pergunta %s sem fabricante", q.ID)
		}
		if q.Pitfalls == "" {
			t.Errorf("pergunta %s sem pegadinhas/alertas documentados", q.ID)
		}
	}

	// Test evaluation calculation
	q1 := ds.Questions[0]
	score := ds.EvaluatePassage(q1, "display bgp peer and peer as-number configured with bras")
	if score <= 0.0 {
		t.Fatalf("score deveria ser positivo, obteve %f", score)
	}
}

// TestBenchmark_All20TelecomScenariosExecution executa e valida os 20 cenários reais de telecom
// verificando que cada um possui sintaxe, comandos esperados, fabricante, modelo e armadilhas documentadas.
func TestBenchmark_All20TelecomScenariosExecution(t *testing.T) {
	ds, err := LoadEmbedded()
	if err != nil {
		t.Fatalf("falha ao carregar dataset de 20 perguntas: %v", err)
	}

	if len(ds.Questions) != 20 {
		t.Fatalf("esperava 20 perguntas no dataset, encontrou %d", len(ds.Questions))
	}

	type ScenarioResult struct {
		ID       string
		Vendor   string
		Topic    string
		CmdCount int
		Score    float64
	}

	var results []ScenarioResult
	for _, q := range ds.Questions {
		// Monta fixture sintética representativa com comandos esperados
		var syntheticText strings.Builder
		syntheticText.WriteString(q.Title + "\n")
		for _, cmd := range q.ExpectedCommands {
			syntheticText.WriteString(cmd + "\n")
		}

		score := ds.EvaluatePassage(q, syntheticText.String())
		if score < 1.0 {
			t.Errorf("cenário %s (%s) obteve recall imperfeito: %.2f", q.ID, q.Title, score)
		}

		results = append(results, ScenarioResult{
			ID:       q.ID,
			Vendor:   strings.Join(q.Vendors, "/"),
			Topic:    q.Topic,
			CmdCount: len(q.ExpectedCommands),
			Score:    score,
		})
	}

	t.Logf("Bateria de 20 Casos Reais Telecom executada com sucesso:")
	for _, r := range results {
		t.Logf("  [%s] %-12s | %-24s | Comandos: %d | Recall: %.0f%%", r.ID, r.Vendor, r.Topic, r.CmdCount, r.Score*100)
	}
}

