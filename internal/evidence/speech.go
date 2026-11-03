package evidence

import (
	"regexp"
	"strings"
)

// Ato de fala da oração. Só speechAssertive pode sustentar um recurso.
// Os outros rejeitam a promoção a prova sem depender da frase exata do teste.
type speechKind int

const (
	speechAssertive speechKind = iota
	speechQuestion
	speechHypothesis
	speechNonFactive
	speechFalsity
)

func (k speechKind) String() string {
	switch k {
	case speechQuestion:
		return "question"
	case speechHypothesis:
		return "hypothesis"
	case speechNonFactive:
		return "nonfactive"
	case speechFalsity:
		return "falsity"
	default:
		return "assertive"
	}
}

var (
	hypothesisRE = regexp.MustCompile(`(?i)(?:^|[:\s])(?:if|whether|suppose[d]?|supposing|assuming|assumed|hypothetical(?:ly)?|hypothesis|hip[óo]tese|supondo|assumindo|caso|si)\b|\bse\s+(?:o|a|os|as|um|uma|houver|for)\b`)
	nonFactiveRE = regexp.MustCompile(`(?i)\b(?:unverified|unconfirmed|unverifiable|alleged(?:ly)?|rumou?red|reportedly|purported(?:ly)?|hearsay|n[aã]o\s+verificad[oa]|nao\s+verificad[oa]|n[aã]o\s+confirmad[oa]|nao\s+confirmad[oa]|alega[cç][aã]o|supost[oa]s?|sin\s+verificar|sin\s+confirmar|no\s+verificad[oa]|claim that|claims that|claimed that)\b`)
	falsityRE    = regexp.MustCompile(`(?i)\b(?:is|are|was|were)\s+(?:explicitly\s+|declared\s+|described\s+as\s+)?false\b|\b(?:described|declared|marked|labeled)\s+as\s+false\b|(?:^|[^0-9A-Za-z_])(?:é|e|está|esta|es|son)\s+(?:explicitamente\s+)?fals[oa]s?(?:[^0-9A-Za-z_]|$)|\b(?:is not true|isn't true|n[aã]o é verdade|nao e verdade|no es verdad|no es cierto)\b`)
)

// classifySpeech distingue afirmação, pergunta, hipótese, relato não factivo
// e negação da verdade da frase. A pontuação de interrogação é lida no texto
// original, porque o separador de orações a remove.
func classifySpeech(snippet, clause string) speechKind {
	lower := strings.ToLower(strings.TrimSpace(clause))
	if clauseIsQuestion(snippet, clause, lower) {
		return speechQuestion
	}
	if hypothesisRE.MatchString(lower) {
		return speechHypothesis
	}
	if nonFactiveRE.MatchString(lower) {
		return speechNonFactive
	}
	if falsityRE.MatchString(lower) {
		return speechFalsity
	}
	return speechAssertive
}

func clauseIsQuestion(snippet, clause, lower string) bool {
	if strings.Contains(clause, "?") || strings.Contains(clause, "¿") {
		return true
	}
	if idx := strings.Index(snippet, clause); idx >= 0 {
		rest := strings.TrimLeft(snippet[idx+len(clause):], " \t")
		if strings.HasPrefix(rest, "?") || strings.HasPrefix(rest, "¿") {
			return true
		}
	}
	body := lower
	if i := strings.Index(body, ":"); i >= 0 && i < 48 {
		body = strings.TrimSpace(body[i+1:])
	}
	return startsAsQuestion(body)
}

func startsAsQuestion(body string) bool {
	fields := strings.Fields(body)
	if len(fields) == 0 {
		return false
	}
	switch fields[0] {
	case "does", "do", "did", "is", "are", "was", "were", "can", "could", "should", "would", "will", "pode", "podem", "puede", "pueden":
		if len(fields) > 1 {
			next := fields[1]
			if next == "not" || next == "não" || next == "nao" || next == "no" || strings.HasPrefix(next, "n't") {
				return false
			}
		}
		return true
	case "which", "what", "who", "whom", "whose", "how", "why", "when", "where", "whether", "qual", "quais", "como", "cuál", "cual", "cómo":
		return true
	}
	return strings.HasPrefix(body, "por que") || strings.HasPrefix(body, "será que") || strings.HasPrefix(body, "sera que")
}
