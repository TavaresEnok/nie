package extract

import (
	"strings"
	"testing"
	"unicode/utf8"
)

func TestPassageOffsetsReferToExactUTF8Text(t *testing.T) {
	input := "  Cabeçalho\n\n" + strings.Repeat("configuração αβγ é válida  ", 100) + "\n\nÚltima linha  "
	passages := MakePassages(input, nil, 201)
	previous := 0
	for _, p := range passages {
		if !utf8.ValidString(p.Text) || p.StartOffset < previous || p.EndOffset > len(input) || input[p.StartOffset:p.EndOffset] != p.Text {
			t.Fatalf("offset ou UTF-8 inválido: %+v", p)
		}
		previous = p.EndOffset
	}
	if len(passages) < 2 {
		t.Fatal("fixture não gerou múltiplos trechos")
	}
}
