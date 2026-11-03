package archive

import (
	"context"
	"testing"
)

func TestAddRejectsPrivateURLBeforeCommand(t *testing.T) {
	err := Add(context.Background(), t.TempDir(), "http://127.0.0.1/admin")
	if err == nil {
		t.Fatal("URL privada não pode chegar ao ArchiveBox")
	}
}

func TestAddRequiresDirectory(t *testing.T) {
	err := Add(context.Background(), " ", "https://example.com/")
	if err == nil {
		t.Fatal("sem diretório o comando não deve existir")
	}
}
