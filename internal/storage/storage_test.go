package storage

import (
	"crypto/sha256"
	"encoding/hex"
	"io"
	"os"
	"path/filepath"
	"testing"
)

func TestPutOpenAndDedupe(t *testing.T) {
	root := t.TempDir()
	s, err := New(root)
	if err != nil {
		t.Fatal(err)
	}
	data := []byte("conteúdo imutável do NIE\n")
	sum := sha256.Sum256(data)
	wantHash := hex.EncodeToString(sum[:])

	res, err := s.Put(data)
	if err != nil {
		t.Fatal(err)
	}
	if res.SHA256 != wantHash {
		t.Fatalf("hash = %s, quer %s", res.SHA256, wantHash)
	}
	if res.Size != int64(len(data)) {
		t.Fatalf("size = %d, quer %d", res.Size, len(data))
	}
	if want := filepath.Join(root, "objects", "sha256", wantHash[0:2], wantHash[2:4], wantHash); res.Path != want {
		t.Fatalf("path = %s, quer %s", res.Path, want)
	}
	onDisk, err := os.ReadFile(res.Path)
	if err != nil {
		t.Fatal(err)
	}
	if string(onDisk) != string(data) {
		t.Fatal("bytes gravados não conferem")
	}

	again, err := s.Put(data)
	if err != nil {
		t.Fatal(err)
	}
	if again.Path != res.Path {
		t.Fatal("segunda gravação deveria reutilizar o objeto")
	}

	reader, err := s.Open(wantHash)
	if err != nil {
		t.Fatal(err)
	}
	got, err := io.ReadAll(reader)
	reader.Close()
	if err != nil {
		t.Fatal(err)
	}
	if string(got) != string(data) {
		t.Fatal("conteúdo lido não confere")
	}
	if !s.Has(wantHash) {
		t.Fatal("Has deveria encontrar o objeto")
	}
	if s.Has("0000") {
		t.Fatal("Has não deveria encontrar hash inexistente")
	}
}
