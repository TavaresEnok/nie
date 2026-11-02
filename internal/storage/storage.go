package storage

import (
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
)

// Store grava bytes imutáveis em objects/sha256/aa/bb/<hash>.
type Store struct {
	root string
}

type Result struct {
	SHA256 string
	Size   int64
	Path   string
}

func New(root string) (*Store, error) {
	if root == "" {
		return nil, errors.New("data dir vazio")
	}
	for _, sub := range []string{"objects", "tmp"} {
		if err := os.MkdirAll(filepath.Join(root, sub), 0o750); err != nil {
			return nil, fmt.Errorf("criar %s: %w", sub, err)
		}
	}
	return &Store{root: root}, nil
}

func (s *Store) Path(hash string) string {
	if len(hash) < 4 {
		return filepath.Join(s.root, "objects", "sha256", hash)
	}
	return filepath.Join(s.root, "objects", "sha256", hash[0:2], hash[2:4], hash)
}

func (s *Store) Put(data []byte) (Result, error) {
	sum := sha256.Sum256(data)
	hash := hex.EncodeToString(sum[:])
	final := s.Path(hash)
	if fi, err := os.Stat(final); err == nil && fi.Size() == int64(len(data)) {
		return Result{SHA256: hash, Size: fi.Size(), Path: final}, nil
	}

	if err := os.MkdirAll(filepath.Dir(final), 0o750); err != nil {
		return Result{}, fmt.Errorf("criar diretório do objeto: %w", err)
	}
	name := make([]byte, 8)
	if _, err := rand.Read(name); err != nil {
		return Result{}, fmt.Errorf("gerar nome temporário: %w", err)
	}
	tmp := filepath.Join(s.root, "tmp", hash+"-"+hex.EncodeToString(name))
	f, err := os.OpenFile(tmp, os.O_CREATE|os.O_EXCL|os.O_WRONLY, 0o640)
	if err != nil {
		return Result{}, fmt.Errorf("criar temporário: %w", err)
	}
	cleanup := true
	defer func() {
		_ = f.Close()
		if cleanup {
			_ = os.Remove(tmp)
		}
	}()

	if _, err := f.Write(data); err != nil {
		return Result{}, fmt.Errorf("gravar objeto: %w", err)
	}
	if err := f.Sync(); err != nil {
		return Result{}, fmt.Errorf("fsync: %w", err)
	}
	if err := f.Close(); err != nil {
		return Result{}, fmt.Errorf("fechar objeto: %w", err)
	}
	if err := os.Rename(tmp, final); err != nil {
		// Outro worker pode ter gravado o mesmo hash entre o stat e o rename.
		if fi, statErr := os.Stat(final); statErr == nil && fi.Size() == int64(len(data)) {
			return Result{SHA256: hash, Size: fi.Size(), Path: final}, nil
		}
		return Result{}, fmt.Errorf("rename atômico: %w", err)
	}
	cleanup = false
	return Result{SHA256: hash, Size: int64(len(data)), Path: final}, nil
}

func (s *Store) Open(hash string) (io.ReadCloser, error) {
	if len(hash) < 4 {
		return nil, errors.New("hash inválido")
	}
	return os.Open(s.Path(hash))
}

func (s *Store) Has(hash string) bool {
	if len(hash) < 4 {
		return false
	}
	_, err := os.Stat(s.Path(hash))
	return err == nil
}
