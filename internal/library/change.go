package library

import "strings"

// Change descreve a relação entre duas capturas da mesma URL.
// Unchanged não é prova de que o conteúdo remoto continua igual quando a
// comparação não teve um hash anterior.
type Change string

const (
	ChangeFirst     Change = "first"
	ChangeUnchanged Change = "unchanged"
	ChangeUpdated   Change = "updated"
)

// Compare captura a decisão de preservação: o hash anterior permanece no
// acervo e só há atualização quando os bytes mudam.
func Compare(previousSHA, nextSHA string) Change {
	previousSHA = strings.TrimSpace(previousSHA)
	nextSHA = strings.TrimSpace(nextSHA)
	switch {
	case previousSHA == "":
		return ChangeFirst
	case previousSHA == nextSHA:
		return ChangeUnchanged
	default:
		return ChangeUpdated
	}
}
