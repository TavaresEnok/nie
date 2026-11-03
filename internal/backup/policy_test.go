package backup

import (
	"testing"
)

func TestRequireOffsiteRepository(t *testing.T) {
	t.Setenv("RESTIC_REPOSITORY", "")
	t.Setenv("RESTIC_PASSWORD", "")
	if err := RequireOffsiteRepository(); err == nil {
		t.Fatal("repositório vazio deveria ser recusado")
	}
	t.Setenv("RESTIC_REPOSITORY", "/var/backup")
	t.Setenv("RESTIC_PASSWORD", "secret")
	if err := RequireOffsiteRepository(); err == nil {
		t.Fatal("caminho local deveria ser recusado")
	}
	t.Setenv("RESTIC_REPOSITORY", "s3:s3.amazonaws.com/nie-backups")
	if err := RequireOffsiteRepository(); err != nil {
		t.Fatal(err)
	}
}
