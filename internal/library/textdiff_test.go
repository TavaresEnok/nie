package library

import "testing"

func TestDiffSummary(t *testing.T) {
	if DiffSummary("igual", "igual", 4) != "" {
		t.Fatal("texto igual não tem diff")
	}
	got := DiffSummary("linha antiga\ncomum", "linha nova\ncomum", 4)
	if !containsAll(got, "1 linhas só na versão anterior", "1 linhas só na versão nova", "- linha antiga", "+ linha nova") {
		t.Fatalf("resumo=%s", got)
	}
}

func containsAll(value string, parts ...string) bool {
	for _, part := range parts {
		if !contains(value, part) {
			return false
		}
	}
	return true
}

func contains(value, part string) bool {
	return len(value) >= len(part) && (value == part || len(part) == 0 || (len(value) > 0 && (func() bool { return stringIndex(value, part) >= 0 })()))
}

func stringIndex(value, part string) int {
	for i := 0; i+len(part) <= len(value); i++ {
		if value[i:i+len(part)] == part {
			return i
		}
	}
	return -1
}
