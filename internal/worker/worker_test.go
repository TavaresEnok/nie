package worker

import (
	"testing"

	"nie/internal/extract"
)

// TestMentionDiscoveryOutsideTelecom garante que a descoberta por menções sem
// link funciona em assuntos fora de redes (critério de arquitetura genérica).
func TestMentionDiscoveryOutsideTelecom(t *testing.T) {
	text := "O estudo (doi 10.1056/NEJMoa2034577) cita a norma ISO 27001:2022, " +
		"a vulnerabilidade CVE-2024-3094 e o preprint arXiv:2303.08774."
	mentions := extract.ExtractMentions(text, 25)

	want := map[string]bool{"10.1056/NEJMoa2034577": false, "ISO 27001:2022": false, "CVE-2024-3094": false, "arXiv:2303.08774": false}
	for _, m := range mentions {
		if _, ok := want[m]; ok {
			want[m] = true
			if !isSearchableMention(m) {
				t.Errorf("menção %q deveria gerar nova consulta", m)
			}
		}
	}
	for m, found := range want {
		if !found {
			t.Errorf("menção %q não extraída; obtidas: %v", m, mentions)
		}
	}
}

func TestComputeFrontierPriorityPrefersQueryTermsAndPenalizesNoise(t *testing.T) {
	q := "metformina efeitos colaterais"
	relevant := computeFrontierPriority("https://example.org/metformina/guia", 1, q)
	neutral := computeFrontierPriority("https://example.org/outra-pagina", 1, q)
	noise := computeFrontierPriority("https://example.org/login", 1, q)
	if !(relevant < neutral && neutral < noise) {
		t.Errorf("prioridade (menor = antes): relevante=%d neutra=%d ruído=%d", relevant, neutral, noise)
	}
}
