package extract

import (
	"testing"
)

func TestExtractMentions(t *testing.T) {
	sampleDoc := `
		Configuração de redundância BGP EVPN e PPPoE dual-stack em caixa Huawei NE8000 e MikroTik CCR2004-16G-2S+.
		Para suporte a comunidades estendidas, consulte a RFC 4360 e a RFC 1997.
		No Datacom DM4270, o comando utilizado foi display bgp routing-table e display interface description.
		Testamos também na OLT ZTE C320 e Fiberhome AN5516 com Option 82 habilitada.
	`

	mentions := ExtractMentions(sampleDoc, 20)

	expectedTerms := []string{
		"RFC 4360",
		"RFC 1997",
		"NE8000",
		"CCR2004-16G-2S+",
		"DM4270",
		"BGP EVPN",
		"PPPoE dual-stack",
		"Option 82",
	}

	foundMap := make(map[string]bool)
	for _, m := range mentions {
		foundMap[m] = true
	}

	for _, exp := range expectedTerms {
		if !foundMap[exp] {
			t.Errorf("termo esperado %q não encontrado nas menções extraídas: %v", exp, mentions)
		}
	}
}

func TestExtractMentionsEmpty(t *testing.T) {
	if res := ExtractMentions(""); len(res) != 0 {
		t.Fatalf("esperava vazio, obteve %v", res)
	}
	if res := ExtractMentions("apenas texto comum sem nenhum termo de telecom ou rfc"); len(res) != 0 {
		t.Fatalf("esperava vazio, obteve %v", res)
	}
}
