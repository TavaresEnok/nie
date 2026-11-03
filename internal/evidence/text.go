package evidence

import (
	"regexp"
	"strings"
)

// isLetterByte trata bytes não ASCII como letras, para que "há" ou "alta" não
// casem com o token "ha" por causa de um caractere acentuado.
func isLetterByte(b byte) bool {
	return (b >= 'a' && b <= 'z') || (b >= 'A' && b <= 'Z') || b >= 0x80
}

func isAlnumByte(b byte) bool {
	return isLetterByte(b) || (b >= '0' && b <= '9')
}

// hasToken informa se tok aparece em lower como termo inteiro. O início exige
// um limite não alfanumérico; o fim aceita dígitos ("bgp4"), versão ("ospfv3") e o
// plural "s" ("vlans"), mas não outras letras ("what" não contém "ha").
// lower deve estar em minúsculas.
func hasToken(lower, tok string) bool {
	if tok == "" {
		return false
	}
	for from := 0; from < len(lower); {
		j := strings.Index(lower[from:], tok)
		if j < 0 {
			return false
		}
		j += from
		end := j + len(tok)
		before := j == 0 || !isAlnumByte(lower[j-1])
		after := end == len(lower) || !isLetterByte(lower[end]) ||
			(lower[end] == 's' && (end+1 == len(lower) || !isLetterByte(lower[end+1]))) ||
			(lower[end] == 'v' && end+1 < len(lower) && lower[end+1] >= '0' && lower[end+1] <= '9')
		if before && after {
			return true
		}
		from = j + 1
	}
	return false
}

var (
	// "HA" só conta como recurso quando aparece como sigla em maiúsculas ou por extenso.
	haAcronymRE = regexp.MustCompile(`\bHA\b`)
	haPhraseRE  = regexp.MustCompile(`(?i)\b(high[- ]availability|alta disponibilidade|alta disponibilidad)\b`)

	// Negação de suporte/disponibilidade em inglês, português e espanhol: o
	// verbo de negação seguido (a até três palavras) de um verbo de suporte.
	negatedSupportRE = regexp.MustCompile(`(?i)(` +
		`\b(?:does not|doesn't|do not|don't|did not|cannot|can not|can't|could not|will not|won't|is not|isn't|are not|aren't|was not|were not|not|no longer|never)\s+(?:[\p{L}\p{N}-]+\s+){0,3}?(?:support|supports|supported|available|applicable|allowed|possible|compatible|implemented|enabled|enable|work|works|configure|configured|configurable|exist|exists)\b` +
		`|\b(?:unsupported|lacks?|lacking|without support|no support|not supported)\b` +
		`|\b(?:não|nao|nunca|jamais)\s+(?:[\p{L}\p{N}-]+\s+){0,3}?(?:suporta\p{L}*|suportad\p{L}+|disponível|disponivel|permite\p{L}*|compatível|compativel|implementad\p{L}+|funciona\p{L}*|configur\p{L}+|existe\p{L}*)` +
		`|\b(?:sem suporte|não há suporte|nao ha suporte)\b` +
		`|\bno\s+(?:[\p{L}\p{N}-]+\s+){0,3}?(?:soporta\p{L}*|admite\p{L}*|disponible|compatible|permite\p{L}*|funciona\p{L}*|implementad\p{L}+)` +
		`|\bsin soporte\b)`)

	// Palavras que declaram a obsolescência ou erro do procedimento.
	refuteKeywordRE = regexp.MustCompile(`(?i)\b(deprecated|obsolete|obsolet[oa]|descontinuad[oa]|syntax error|incompatible|incompatível|incompativel)\b`)

	// Negação passiva (o recurso aparece ANTES do predicado negativo).
	passiveNegationRE = regexp.MustCompile(`(?i)(` +
		`\b(?:is not|isn't|are not|aren't|was not|were not|cannot be|can not be|can't be|could not be|will not be|won't be)\s+(?:[\p{L}\p{N}-]+\s+){0,2}?(?:supported|available|applicable|allowed|possible|compatible|implemented|enabled|configured|configurable)\b` +
		`|\b(?:is|are|was|were)\s+(?:[\p{L}\p{N}-]+\s+){0,2}?(?:unsupported|deprecated|obsolete|incompatible)\b` +
		`|\b(?:não|nao|nunca|jamais)\s+(?:é|e|são|sao|está|esta|estão|estao|foi|foram|pode ser|podem ser)\s+(?:[\p{L}\p{N}-]+\s+){0,2}?(?:suportad\p{L}+|disponível|disponivel|compatível|compativel|implementad\p{L}+|configurad\p{L}+|permitid\p{L}+|obsolet\p{L}+|descontinuad\p{L}+|incompatível|incompativel)\b` +
		`|\b(?:é|e|são|sao|está|esta)\s+(?:[\p{L}\p{N}-]+\s+){0,2}?(?:obsolet[oa]s?|descontinuad[oa]s?|incompatível|incompativel)\b` +
		`|\bno\s+(?:es|son|está|esta|están|estan|puede ser|pueden ser)\s+(?:[\p{L}\p{N}-]+\s+){0,2}?(?:soportad\p{L}+|admitid\p{L}+|disponibles?|compatibles?|implementad\p{L}+|configurad\p{L}+|obsolet\p{L}+)\b` +
		`|\b(?:es|son|está|esta)\s+(?:[\p{L}\p{N}-]+\s+){0,2}?(?:obsolet[oa]s?|incompatibles?)\b)`)

	// Negação ativa ou preposicional (o recurso negado aparece DEPOIS do marcador).
	activeNegationRE = regexp.MustCompile(`(?i)(` +
		`\b(?:does not|doesn't|do not|don't|did not|cannot|can not|can't|could not|will not|won't|never)\s+(?:[\p{L}\p{N}-]+\s+){0,2}?(?:support|supports|allow|allows|enable|enables|implement|implements|work|works|configure|have|provide)\b` +
		`|\b(?:lacks?|lacking|without(?:\s+support\s+for)?|no\s+support\s+for|not\s+supporting)\b` +
		`|\b(?:não|nao|nunca|jamais)\s+(?:[\p{L}\p{N}-]+\s+){0,2}?(?:suporta\p{L}*|permite\p{L}*|implementa\p{L}*|funciona\p{L}*|configura\p{L}*|possui|tem|oferece)\b` +
		`|\b(?:sem(?:\s+suporte\s+(?:a|ao|para))?|não há suporte(?:\s+(?:a|ao|para))?|nao ha suporte(?:\s+(?:a|ao|para))?)\b` +
		`|\bno\s+(?:[\p{L}\p{N}-]+\s+){0,2}?(?:soporta\p{L}*|admite\p{L}*|permite\p{L}*|funciona\p{L}*|implementa\p{L}*|configura\p{L}*)\b` +
		`|\b(?:sin(?:\s+soporte(?:\s+para)?)?)\b)`)

	// Marcadores explícitos de conflito/divergência documental no próprio trecho.
	conflictMarkerRE = regexp.MustCompile(`(?i)\b(conflict(?:ing|s)?|contradict(?:ory|ing|s|ion)?|disputed|inconclusive|conflit(?:o|ante|antes)|contradit[óo]ri[oa]s?|diverg(?:em|ente|entes|ência|encia)|inconclusiv[oa]|en conflicto|contradictori[oa]s?)\b`)

	// Enunciados condicionais ou parciais: não sustentam o recurso por inteiro.
	qualifiedRE = regexp.MustCompile(`(?i)\b(only|except|unless|limited to|partial|partially|provided that|as long as|depending on|apenas|somente|exceto|salvo|limitad[oa]|parcial|parcialmente|desde que|dependendo|solo|únicamente|excepto|siempre que)\b`)
)

// requestsHA informa se a pergunta pede alta disponibilidade (sigla "HA" em
// maiúsculas ou a expressão por extenso). A palavra comum "ha"/"what" não conta.
func requestsHA(query string) bool {
	return haAcronymRE.MatchString(query) || haPhraseRE.MatchString(query)
}

// snippetHasFeature informa se o trecho trata do recurso. HA exige a sigla ou
// a expressão por extenso; os demais recursos exigem o termo inteiro.
func snippetHasFeature(feature, snippet, lowerSnippet string) bool {
	if strings.EqualFold(feature, "HA") {
		return haAcronymRE.MatchString(snippet) || haPhraseRE.MatchString(snippet)
	}
	return hasToken(lowerSnippet, strings.ToLower(feature))
}

// snippetNegatesSupport: o trecho nega suporte/disponibilidade.
func snippetNegatesSupport(lowerSnippet string) bool {
	return negatedSupportRE.MatchString(lowerSnippet)
}

// snippetRefutes: negação explícita ou palavra de obsolescência/erro.
func snippetRefutes(lowerSnippet string) bool {
	return snippetNegatesSupport(lowerSnippet) || refuteKeywordRE.MatchString(lowerSnippet)
}

// snippetIsQualified: o enunciado é condicional ou parcial.
func snippetIsQualified(lowerSnippet string) bool {
	return qualifiedRE.MatchString(lowerSnippet)
}
