package discovery

import (
	"bytes"
	"context"
	"fmt"
	"io/fs"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

type GitLocalFinder struct {
	baseDir string
}

func NewGitLocal(baseDir string) *GitLocalFinder {
	return &GitLocalFinder{baseDir: baseDir}
}

// Origem do trecho encontrado.
const (
	// SourceCommit: o trecho existe no blob do commit citado em Revision.
	SourceCommit = "commit"
	// SourceWorkingTree: o trecho só existe no diretório de trabalho (arquivo
	// modificado ou não rastreado, ou diretório que não é um repositório Git).
	// Revision é "uncommitted" e BaseRevision traz o commit do qual o
	// diretório divergiu, quando há.
	SourceWorkingTree = "working_tree"
)

type LocalFileMatch struct {
	Path     string `json:"path"`
	Line     int    `json:"line"`
	Snippet  string `json:"snippet"`
	Revision string `json:"revision,omitempty"`
	// Source diz de onde o trecho veio (SourceCommit ou SourceWorkingTree).
	Source string `json:"source"`
	// Dirty indica que o arquivo difere do commit (modificado ou não rastreado).
	Dirty bool `json:"dirty,omitempty"`
	// BaseRevision é o HEAD do repositório quando o trecho vem do working tree.
	BaseRevision string `json:"base_revision,omitempty"`
}

const uncommitted = "uncommitted"

// gitState descreve o repositório em baseDir: o commit HEAD e os arquivos que
// divergem dele (modificados, staged ou não rastreados), com caminhos relativos
// a baseDir e separador "/".
type gitState struct {
	head  string
	dirty map[string]bool
	ok    bool
}

func (g *GitLocalFinder) git(ctx context.Context, args ...string) ([]byte, error) {
	cmd := exec.CommandContext(ctx, "git", args...)
	cmd.Dir = g.baseDir
	return cmd.Output()
}

// loadGitState consulta o Git de verdade; nunca confia em arquivos de .git
// lidos à mão, que não provam que o conteúdo foi commitado.
func (g *GitLocalFinder) loadGitState(ctx context.Context) gitState {
	out, err := g.git(ctx, "rev-parse", "HEAD")
	if err != nil {
		return gitState{}
	}
	head := strings.TrimSpace(string(out))
	if len(head) != 40 {
		return gitState{}
	}
	st := gitState{head: head, dirty: map[string]bool{}, ok: true}

	// Modificados ou staged em relação ao HEAD.
	if out, err := g.git(ctx, "diff", "--relative", "--name-only", "-z", "HEAD"); err == nil {
		for _, p := range bytes.Split(out, []byte{0}) {
			if len(p) > 0 {
				st.dirty[string(p)] = true
			}
		}
	} else {
		return gitState{}
	}
	// Não rastreados (respeitando .gitignore).
	if out, err := g.git(ctx, "ls-files", "--others", "--exclude-standard", "-z"); err == nil {
		for _, p := range bytes.Split(out, []byte{0}) {
			if len(p) > 0 {
				st.dirty[string(p)] = true
			}
		}
	} else {
		return gitState{}
	}
	return st
}

// committedBlob devolve o conteúdo do arquivo no commit HEAD (ok=false se o
// arquivo não existe nesse commit).
func (g *GitLocalFinder) committedBlob(ctx context.Context, rel string) (string, bool) {
	out, err := g.git(ctx, "show", "HEAD:./"+rel)
	if err != nil {
		return "", false
	}
	return string(out), true
}

// findTerms procura todos os termos no conteúdo e devolve linha e trecho do
// primeiro termo.
func findTerms(content string, terms []string) (int, string, bool) {
	lower := strings.ToLower(content)
	for _, term := range terms {
		if !strings.Contains(lower, term) {
			return 0, "", false
		}
	}
	idx := strings.Index(lower, terms[0])
	lineNum := strings.Count(content[:idx], "\n") + 1
	start := idx - 80
	if start < 0 {
		start = 0
	}
	end := idx + 200
	if end > len(content) {
		end = len(content)
	}
	return lineNum, fmt.Sprintf("...%s...", strings.TrimSpace(content[start:end])), true
}

// Search busca os termos em repositórios ou pastas de configuração locais.
//
// A proveniência é verdadeira: um trecho só recebe Revision=<sha> se existe no
// blob daquele commit. Trechos que só existem em arquivos modificados ou não
// rastreados voltam como working_tree, Dirty, Revision "uncommitted".
func (g *GitLocalFinder) Search(ctx context.Context, query string, maxResults int) ([]LocalFileMatch, error) {
	if maxResults <= 0 {
		maxResults = 20
	}
	if g.baseDir == "" {
		return nil, nil
	}
	if _, err := os.Stat(g.baseDir); os.IsNotExist(err) {
		return nil, nil
	}

	terms := strings.Fields(strings.ToLower(query))
	if len(terms) == 0 {
		return nil, nil
	}

	state := g.loadGitState(ctx)

	var results []LocalFileMatch
	err := filepath.WalkDir(g.baseDir, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return nil
		}
		if ctx.Err() != nil {
			return ctx.Err()
		}
		if d.IsDir() {
			if d.Name() == ".git" || d.Name() == "node_modules" {
				return filepath.SkipDir
			}
			return nil
		}

		switch strings.ToLower(filepath.Ext(path)) {
		case ".txt", ".md", ".cfg", ".conf", ".rsc", ".yaml", ".yml", ".json", ".go", ".py":
		default:
			return nil
		}

		info, err := d.Info()
		if err != nil || info.Size() > 2<<20 { // 2 MiB limit
			return nil
		}
		data, err := os.ReadFile(path)
		if err != nil {
			return nil
		}
		rel, relErr := filepath.Rel(g.baseDir, path)
		rel = filepath.ToSlash(rel)

		match := LocalFileMatch{Path: path}
		switch {
		case !state.ok || relErr != nil:
			// Sem repositório Git verificável: só dá para afirmar o working tree.
			line, snip, ok := findTerms(string(data), terms)
			if !ok {
				return nil
			}
			match.Line, match.Snippet, match.Source, match.Revision = line, snip, SourceWorkingTree, uncommitted

		case !state.dirty[rel]:
			// Arquivo idêntico ao HEAD: o trecho pertence a esse commit.
			line, snip, ok := findTerms(string(data), terms)
			if !ok {
				return nil
			}
			match.Line, match.Snippet, match.Source, match.Revision = line, snip, SourceCommit, state.head

		default:
			// Arquivo divergente: prefere o que está de fato no commit.
			if blob, inHead := g.committedBlob(ctx, rel); inHead {
				if line, snip, ok := findTerms(blob, terms); ok {
					match.Line, match.Snippet, match.Source, match.Revision = line, snip, SourceCommit, state.head
					match.Dirty = true // o arquivo local difere, mas o trecho é do commit
					break
				}
			}
			line, snip, ok := findTerms(string(data), terms)
			if !ok {
				return nil
			}
			match.Line, match.Snippet, match.Source, match.Revision = line, snip, SourceWorkingTree, uncommitted
			match.Dirty, match.BaseRevision = true, state.head
		}

		results = append(results, match)
		if len(results) >= maxResults {
			return filepath.SkipAll
		}
		return nil
	})

	return results, err
}
