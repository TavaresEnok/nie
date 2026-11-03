package storage

import (
	"errors"
	"os"
	"testing"
)

func TestCorruptedObjectNeverReturnedOrOverwritten(t *testing.T) {
	s, err := New(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	original := []byte("original")
	result, err := s.Put(original)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(result.Path, []byte("modified"), 0600); err != nil {
		t.Fatal(err)
	}
	if _, err := s.Open(result.SHA256); !errors.Is(err, ErrIntegrity) {
		t.Fatalf("objeto adulterado aceito: %v", err)
	}
	if _, err := s.Put(original); !errors.Is(err, ErrIntegrity) {
		t.Fatalf("corrupção sobrescrita silenciosamente: %v", err)
	}
	if _, err := s.Open("../../outside"); err == nil {
		t.Fatal("hash inválido aceito")
	}
}
