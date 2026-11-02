package discovery

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// TestGitLocal_FindsConfigNotPresentInReadme verifica o critério de aceite P1-05:
// Encontrar comando em arquivo .cfg ou .rsc não presente no README, indicando caminho e linha/revisão.
func TestGitLocal_FindsConfigNotPresentInReadme(t *testing.T) {
	tempDir := t.TempDir()

	// 1. Cria README genérico sem o comando
	readmeContent := "# Telecom Repo\nRepositório de automação de rede.\nNenhum detalhe de BGP ou OLT aqui.\n"
	if err := os.WriteFile(filepath.Join(tempDir, "README.md"), []byte(readmeContent), 0644); err != nil {
		t.Fatal(err)
	}

	// 2. Cria arquivo .cfg com configuração técnica específica
	cfgContent := `
! Configuração Roteador Huawei
sysname NE8000-Core
#
interface GigabitEthernet0/1/0
 ip address 10.0.0.1 255.255.255.252
#
bgp 65000
 peer 10.0.0.2 as-number 65001
 peer 10.0.0.2 enable
#
`
	if err := os.WriteFile(filepath.Join(tempDir, "core_router.cfg"), []byte(cfgContent), 0644); err != nil {
		t.Fatal(err)
	}

	// 3. Cria arquivo .rsc do RouterOS
	rscContent := `/routing bgp connection
add name=bgp-peer-isp remote.address=192.0.2.1/32 .as=65100 local.as=65000 routing-table=main
`
	if err := os.WriteFile(filepath.Join(tempDir, "mikrotik_bgp.rsc"), []byte(rscContent), 0644); err != nil {
		t.Fatal(err)
	}

	finder := NewGitLocal(tempDir)
	ctx := context.Background()

	// Busca por "peer 10.0.0.2" que só existe no .cfg
	matches, err := finder.Search(ctx, "peer 10.0.0.2", 10)
	if err != nil {
		t.Fatalf("Search falhou: %v", err)
	}
	if len(matches) == 0 {
		t.Fatalf("Nenhum match encontrado para 'peer 10.0.0.2' no arquivo .cfg")
	}

	m := matches[0]
	if !strings.HasSuffix(m.Path, "core_router.cfg") {
		t.Errorf("Esperava match em core_router.cfg, obteve %s", m.Path)
	}
	if m.Line <= 0 {
		t.Errorf("Linha deveria ser > 0, obteve %d", m.Line)
	}
	if m.Revision == "" {
		t.Errorf("Revisão não deveria ser vazia")
	}

	// 4. Cria mock de .git com commit SHA verificável de 40 caracteres (P1-07)
	expectedSHA := "360d9b7b577eb65984ff7f7115333ec8090c3062"
	gitDir := filepath.Join(tempDir, ".git")
	refsDir := filepath.Join(gitDir, "refs", "heads")
	if err := os.MkdirAll(refsDir, 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(gitDir, "HEAD"), []byte("ref: refs/heads/main\n"), 0644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(refsDir, "main"), []byte(expectedSHA+"\n"), 0644); err != nil {
		t.Fatal(err)
	}

	// Executa nova busca e verifica que Revision agora é o SHA real do commit, NÃO a string "HEAD"
	matchesWithGit, err := finder.Search(ctx, "peer 10.0.0.2", 10)
	if err != nil {
		t.Fatalf("Search com git falhou: %v", err)
	}
	if len(matchesWithGit) == 0 {
		t.Fatalf("Nenhum match encontrado")
	}
	if matchesWithGit[0].Revision != expectedSHA {
		t.Errorf("P1-07: Esperava commit SHA '%s', obteve '%s'", expectedSHA, matchesWithGit[0].Revision)
	}

	// Busca por "bgp-peer-isp" que só existe no .rsc
	matchesRSC, err := finder.Search(ctx, "bgp-peer-isp", 10)
	if err != nil {
		t.Fatalf("Search RSC falhou: %v", err)
	}
	if len(matchesRSC) == 0 {
		t.Fatalf("Nenhum match encontrado para 'bgp-peer-isp' no arquivo .rsc")
	}
	if !strings.HasSuffix(matchesRSC[0].Path, "mikrotik_bgp.rsc") {
		t.Errorf("Esperava match em mikrotik_bgp.rsc, obteve %s", matchesRSC[0].Path)
	}
}
