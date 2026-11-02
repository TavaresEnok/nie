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

// TestEvaluatorSyntheticFixtures (P0-04) valida o cálculo matemático do avaliador com fixtures sintéticas.
// Este teste NÃO representa nem deve ser anunciado como pesquisa em produção ou crawling real na internet;
// ele verifica exclusivamente que a função EvaluatePassage calcula o score correto quando comandos esperados estão presentes.
func TestEvaluatorSyntheticFixtures(t *testing.T) {
	ds, err := LoadEmbedded()
	if err != nil {
		t.Fatalf("falha ao carregar dataset de 20 perguntas: %v", err)
	}

	if len(ds.Questions) != 20 {
		t.Fatalf("esperava 20 perguntas no dataset, encontrou %d", len(ds.Questions))
	}

	for _, q := range ds.Questions {
		var syntheticText strings.Builder
		syntheticText.WriteString(q.Title + "\n")
		for _, cmd := range q.ExpectedCommands {
			syntheticText.WriteString(cmd + "\n")
		}

		score := ds.EvaluatePassage(q, syntheticText.String())
		if score < 1.0 {
			t.Errorf("fixture sintética para %s (%s) falhou no avaliador: %.2f", q.ID, q.Title, score)
		}
	}
	t.Logf("Verificação matemática do avaliador em 20 fixtures sintéticas de referência concluída (não representa pesquisa real)")
}

// TestBenchmark_EvaluatorRejectsNegativeAndContradictoryPassages (P0-04) valida que passagens
// irrelevantes ou de outros fabricantes retornam pontuação zero ou nula, evitando falsos positivos.
func TestBenchmark_EvaluatorRejectsNegativeAndContradictoryPassages(t *testing.T) {
	ds, err := LoadEmbedded()
	if err != nil {
		t.Fatalf("falha ao carregar dataset: %v", err)
	}

	// Teste com pergunta TEL-001 (Huawei BGP/BNG)
	q1 := ds.Questions[0]

	// Passagem de fabricante completamente divergente (MikroTik)
	divergentPassage := "/interface ethernet set [ find default-name=ether1 ] name=uplink"
	scoreDivergent := ds.EvaluatePassage(q1, divergentPassage)
	if scoreDivergent > 0.0 {
		t.Errorf("Passagem de fabricante totalmente divergente recebeu score positivo indevido: %.2f", scoreDivergent)
	}

	// Passagem vazia ou sem comandos técnicos
	emptyScore := ds.EvaluatePassage(q1, "Texto puramente genérico sobre computação em nuvem sem comandos CLI.")
	if emptyScore > 0.0 {
		t.Errorf("Passagem genérica recebeu score positivo indevido: %.2f", emptyScore)
	}
}
