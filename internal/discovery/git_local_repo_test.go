package discovery

import (
	"context"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

// initGitRepo cria um repositório Git real com um commit contendo files.
func initGitRepo(t *testing.T, files map[string]string) (dir, sha string) {
	t.Helper()
	if _, err := exec.LookPath("git"); err != nil {
		t.Skip("git não instalado")
	}
	dir = t.TempDir()
	run := func(args ...string) string {
		cmd := exec.Command("git", args...)
		cmd.Dir = dir
		out, err := cmd.CombinedOutput()
		if err != nil {
			t.Fatalf("git %v: %v\n%s", args, err, out)
		}
		return strings.TrimSpace(string(out))
	}
	run("init", "-q")
	run("config", "user.email", "t@example.org")
	run("config", "user.name", "t")
	run("config", "core.autocrlf", "false")
	for name, content := range files {
		if err := os.WriteFile(filepath.Join(dir, name), []byte(content), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	run("add", "-A")
	run("commit", "-q", "-m", "inicial")
	return dir, run("rev-parse", "HEAD")
}

func TestGitLocal_CommittedContentCitesRealCommit(t *testing.T) {
	dir, sha := initGitRepo(t, map[string]string{"core.cfg": "bgp 65000\n peer 10.0.0.2 as-number 65001\n"})
	got, err := NewGitLocal(dir).Search(context.Background(), "peer 10.0.0.2", 5)
	if err != nil || len(got) != 1 {
		t.Fatalf("esperava 1 match, obteve %v err=%v", got, err)
	}
	if got[0].Source != SourceCommit || got[0].Revision != sha || got[0].Dirty {
		t.Errorf("conteúdo commitado deve citar o commit real %s: %+v", sha, got[0])
	}
}

// Regressão da auditoria: texto presente só em alteração não commitada não pode
// ser atribuído ao SHA do último commit.
func TestGitLocal_UncommittedChangeNotAttributedToCommit(t *testing.T) {
	dir, sha := initGitRepo(t, map[string]string{"core.cfg": "bgp 65000\n peer 10.0.0.2 as-number 65001\n"})
	edited := "bgp 65000\n peer 10.0.0.2 as-number 65001\n peer 10.9.9.9 as-number 65999\n"
	if err := os.WriteFile(filepath.Join(dir, "core.cfg"), []byte(edited), 0o644); err != nil {
		t.Fatal(err)
	}
	finder := NewGitLocal(dir)

	got, _ := finder.Search(context.Background(), "10.9.9.9", 5)
	if len(got) != 1 {
		t.Fatalf("esperava 1 match no working tree, obteve %v", got)
	}
	m := got[0]
	if m.Source != SourceWorkingTree || m.Revision != "uncommitted" || !m.Dirty || m.BaseRevision != sha {
		t.Errorf("trecho só no working tree deve ser working_tree/uncommitted/dirty com base %s: %+v", sha, m)
	}

	// O que já estava no commit continua citando o commit, mesmo com o arquivo alterado.
	old, _ := finder.Search(context.Background(), "peer 10.0.0.2", 5)
	if len(old) != 1 || old[0].Source != SourceCommit || old[0].Revision != sha {
		t.Errorf("trecho presente no commit deve citá-lo: %+v", old)
	}
}

func TestGitLocal_UntrackedFileIsWorkingTreeOnly(t *testing.T) {
	dir, sha := initGitRepo(t, map[string]string{"README.md": "repo"})
	if err := os.WriteFile(filepath.Join(dir, "novo.rsc"), []byte("/routing bgp connection add name=novo-peer\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	got, _ := NewGitLocal(dir).Search(context.Background(), "novo-peer", 5)
	if len(got) != 1 || got[0].Source != SourceWorkingTree || got[0].Revision != "uncommitted" || !got[0].Dirty || got[0].BaseRevision != sha {
		t.Errorf("arquivo não rastreado não pode citar commit: %+v", got)
	}
}
