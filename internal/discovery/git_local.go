package discovery

import (
	"context"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
)

type GitLocalFinder struct {
	baseDir string
}

func NewGitLocal(baseDir string) *GitLocalFinder {
	return &GitLocalFinder{baseDir: baseDir}
}

type LocalFileMatch struct {
	Path     string `json:"path"`
	Line     int    `json:"line"`
	Snippet  string `json:"snippet"`
	Revision string `json:"revision,omitempty"`
}

// Search scans local git repositories or config folders for target query keywords.
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

		ext := strings.ToLower(filepath.Ext(path))
		switch ext {
		case ".txt", ".md", ".cfg", ".conf", ".rsc", ".yaml", ".yml", ".json", ".go", ".py":
			// Processable text
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
		content := string(data)
		lowerContent := strings.ToLower(content)

		allMatched := true
		for _, term := range terms {
			if !strings.Contains(lowerContent, term) {
				allMatched = false
				break
			}
		}

		if allMatched {
			// Extract snippet and line number
			idx := strings.Index(lowerContent, terms[0])
			lineNum := strings.Count(content[:idx], "\n") + 1

			start := idx - 80
			if start < 0 {
				start = 0
			}
			end := idx + 200
			if end > len(content) {
				end = len(content)
			}
			snippet := strings.TrimSpace(content[start:end])
			results = append(results, LocalFileMatch{
				Path:     path,
				Line:     lineNum,
				Snippet:  fmt.Sprintf("...%s...", snippet),
				Revision: "HEAD",
			})
			if len(results) >= maxResults {
				return filepath.SkipAll
			}
		}
		return nil
	})

	return results, err
}
