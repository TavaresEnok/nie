package api

import (
	"strings"
	"testing"
)

func TestRetrievalQueriesDecomposeLongQuestion(t *testing.T) {
	q := "Como configurar redundância de BNG Huawei NE8000 VRP v8 com PPPoE e BGP?"
	got := retrievalQueries(q)
	if got[0] != q {
		t.Fatalf("a pergunta original deve vir primeiro: %v", got)
	}
	joined := strings.ToLower(strings.Join(got, "|"))
	for _, want := range []string{"ne8000 pppoe", "ne8000 bgp", "huawei bgp"} {
		if !strings.Contains(joined, want) {
			t.Errorf("faltou a consulta curta %q em %v", want, got)
		}
	}
	seen := map[string]bool{}
	for _, s := range got {
		if seen[strings.ToLower(s)] {
			t.Errorf("consulta repetida: %q", s)
		}
		seen[strings.ToLower(s)] = true
	}
	if one := retrievalQueries("metformina efeitos colaterais"); len(one) != 1 {
		t.Errorf("fora de telecom a pergunta é usada como está: %v", one)
	}
}
