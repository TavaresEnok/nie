package research

import (
	"testing"

	"nie/internal/domain"
)

func TestMergePassagesKeepsFirstAndAddsNeighbor(t *testing.T) {
	base := []domain.SearchResult{{PassageID: 1, Text: "origem"}}
	extra := []domain.SearchResult{{PassageID: 1, Text: "repetido"}, {PassageID: 2, Text: "vizinho", Match: "neighbor"}}
	got := MergePassages(base, extra, 10)
	if len(got) != 2 || got[0].Text != "origem" || got[1].Match != "neighbor" {
		t.Fatalf("%+v", got)
	}
}
