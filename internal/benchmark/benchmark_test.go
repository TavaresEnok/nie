package benchmark

import (
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
