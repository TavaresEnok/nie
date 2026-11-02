package evidence

import (
	"regexp"
	"strings"
)

// Análise por oração (auditoria V5). Flags calculadas sobre o trecho inteiro
// misturavam afirmações diferentes: a negação do Telnet refutava o BGP, e
// "suporta BGP" + "guia de PPPoE" virava suporte conjunto. Aqui cada oração é
// uma afirmação atômica, e um recurso só é sustentado ou refutado pela oração
// que de fato fala dele.

// clauseSplitRE separa orações: fim de frase (pontuação seguida de espaço ou
// fim do texto, o que preserva versões e IPs como "v8.2" e "10.0.0.2"),
// quebras de linha e conjunções adversativas em EN/PT/ES.
var clauseSplitRE = regexp.MustCompile(`(?i)(?:[.!?;]+(?:\s+|$)|\n+|,?\s+(?:but|however|whereas|although|mas|porém|porem|contudo|entretanto|pero|sin embargo|aunque)\s+)`)

func splitClauses(snippet string) []string {
	var out []string
	for _, c := range clauseSplitRE.Split(snippet, -1) {
		if c = strings.TrimSpace(c); c != "" {
			out = append(out, c)
		}
	}
	return out
}

// passageAssertion resume o que o trecho afirma sobre os recursos pedidos.
type passageAssertion struct {
	// CoAsserted: uma mesma oração afirma, sem negação nem condição, todos os
	// recursos pedidos. Recursos citados em orações separadas não provam que
	// funcionam juntos.
	CoAsserted bool
	// Refuted: uma oração que cita um recurso pedido o nega ou o declara
	// obsoleto/incompatível.
	Refuted bool
}

// analyzeAssertion avalia cada oração do trecho. cliCommands são os comandos
// que o classificador reconheceu no trecho, usados para saber se uma oração
// contém sintaxe de configuração.
func analyzeAssertion(snippet string, features, cliCommands []string) passageAssertion {
	var a passageAssertion
	if len(features) == 0 {
		return a
	}
	for _, clause := range splitClauses(snippet) {
		lower := strings.ToLower(clause)

		mentionsAny, mentionsAll := false, true
		for _, f := range features {
			if snippetHasFeature(f, clause, lower) {
				mentionsAny = true
			} else {
				mentionsAll = false
			}
		}
		if !mentionsAny {
			continue // a oração fala de outra coisa (ex.: Telnet)
		}
		if snippetRefutes(lower) {
			a.Refuted = true
			continue
		}
		if !mentionsAll || snippetIsQualified(lower) {
			continue
		}
		hasCLI := false
		for _, cmd := range cliCommands {
			if cmd != "" && strings.Contains(lower, strings.ToLower(cmd)) {
				hasCLI = true
				break
			}
		}
		if assertsContent(lower, hasCLI) {
			a.CoAsserted = true
		}
	}
	return a
}

// firmwareAliases lista as grafias aceitas como menção explícita de cada
// release. O firmware nunca é inferido a partir do modelo: "NE8000" sozinho
// não prova "VRP v8".
var firmwareAliases = map[string][]string{
	"VRP v8":      {"vrp v8", "vrpv8", "vrp-v8", "vrp8", "vrp v800", "v800r"},
	"VRP v5":      {"vrp v5", "vrpv5", "vrp-v5", "vrp5"},
	"RouterOS v7": {"routeros v7", "routeros 7", "routerosv7", "ros v7", "ros7"},
	"RouterOS v6": {"routeros v6", "routeros 6", "routerosv6", "ros v6", "ros6"},
	"IOS-XR":      {"ios-xr", "ios xr", "iosxr"},
	"Junos":       {"junos"},
	"TiMOS":       {"timos"},
}

// conflictingFirmware: releases que, se citadas explicitamente, indicam que o
// trecho trata de outra versão do mesmo sistema.
var conflictingFirmware = map[string]string{
	"VRP v8":      "VRP v5",
	"VRP v5":      "VRP v8",
	"RouterOS v7": "RouterOS v6",
	"RouterOS v6": "RouterOS v7",
}

// mentionsFirmware informa se o trecho cita explicitamente a release.
func mentionsFirmware(lowerSnippet, firmware string) bool {
	aliases, ok := firmwareAliases[firmware]
	if !ok {
		aliases = []string{strings.ToLower(firmware)}
	}
	for _, a := range aliases {
		if hasToken(lowerSnippet, a) {
			return true
		}
	}
	return false
}

// mentionsConflictingFirmware informa se o trecho cita uma release diferente
// da pedida (e não cita a pedida).
func mentionsConflictingFirmware(lowerSnippet, firmware string) bool {
	other, ok := conflictingFirmware[firmware]
	return ok && mentionsFirmware(lowerSnippet, other) && !mentionsFirmware(lowerSnippet, firmware)
}
