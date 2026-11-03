package library

import (
	"fmt"
	"strings"
)

// DiffSummary compara duas extrações linha a linha, com teto, e devolve uma
// descrição curta. Textos iguais produzem string vazia. Não é um patch aplicável.
func DiffSummary(before, after string, sampleLines int) string {
	if before == after {
		return ""
	}
	if sampleLines < 1 {
		sampleLines = 6
	}
	oldLines := boundedLines(before)
	newLines := boundedLines(after)
	removed := missing(oldLines, newLines, sampleLines)
	added := missing(newLines, oldLines, sampleLines)
	var b strings.Builder
	fmt.Fprintf(&b, "%d linhas só na versão anterior, %d linhas só na versão nova", len(countOnly(oldLines, newLines)), len(countOnly(newLines, oldLines)))
	for _, line := range removed {
		fmt.Fprintf(&b, "\n- %s", line)
	}
	for _, line := range added {
		fmt.Fprintf(&b, "\n+ %s", line)
	}
	return b.String()
}

func boundedLines(value string) []string {
	lines := strings.Split(value, "\n")
	if len(lines) > 2000 {
		lines = lines[:2000]
	}
	out := make([]string, 0, len(lines))
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}
		if len(line) > 240 {
			line = line[:240]
		}
		out = append(out, line)
	}
	return out
}

func countOnly(left, right []string) []string {
	seen := map[string]int{}
	for _, line := range right {
		seen[line]++
	}
	var out []string
	for _, line := range left {
		if seen[line] > 0 {
			seen[line]--
			continue
		}
		out = append(out, line)
	}
	return out
}

func missing(left, right []string, limit int) []string {
	only := countOnly(left, right)
	if len(only) > limit {
		return only[:limit]
	}
	return only
}
