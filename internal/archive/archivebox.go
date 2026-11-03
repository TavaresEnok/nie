package archive

import (
	"context"
	"fmt"
	"os/exec"
	"strings"
	"time"

	"nie/internal/urlutil"
)

// Add pede ao ArchiveBox uma captura seletiva da URL. O catálogo do NIE não
// passa a ser o ArchiveBox. Sem diretório configurado, a função não executa nada.
func Add(ctx context.Context, dataDir, pageURL string) error {
	dataDir = strings.TrimSpace(dataDir)
	if dataDir == "" {
		return fmt.Errorf("ArchiveBox não configurado")
	}
	u, _, err := urlutil.ParseNormalized(pageURL)
	if err != nil {
		return err
	}
	if err := urlutil.ValidateCrawlURL(u); err != nil {
		return err
	}
	ctx, cancel := context.WithTimeout(ctx, 2*time.Minute)
	defer cancel()
	cmd := exec.CommandContext(ctx, "archivebox", "add", "--update", u.String())
	cmd.Dir = dataDir
	out, err := cmd.CombinedOutput()
	if err != nil {
		msg := strings.TrimSpace(string(out))
		if msg == "" {
			msg = err.Error()
		}
		return fmt.Errorf("archivebox: %s", msg)
	}
	return nil
}
