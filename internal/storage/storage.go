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

var ErrIntegrity = errors.New("objeto CAS corrompido")

func validHash(hash string) bool {
	if len(hash) != 64 {
		return false
	}
	for _, r := range hash {
		if !(r >= '0' && r <= '9' || r >= 'a' && r <= 'f') {
			return false
		}
	}
	return true
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
	if f, err := s.OpenVerified(hash); err == nil {
		_ = f.Close()
		return Result{SHA256: hash, Size: int64(len(data)), Path: final}, nil
	} else if !errors.Is(err, os.ErrNotExist) {
		return Result{}, err
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
	// Hard-link publication is atomic and never overwrites an existing object.
	if err := os.Link(tmp, final); err != nil {
		if existing, verifyErr := s.OpenVerified(hash); verifyErr == nil {
			existing.Close()
			return Result{SHA256: hash, Size: int64(len(data)), Path: final}, nil
		}
		return Result{}, fmt.Errorf("publicar objeto imutável: %w", err)
	}
	return Result{SHA256: hash, Size: int64(len(data)), Path: final}, nil
}

func (s *Store) Open(hash string) (io.ReadCloser, error) {
	return s.OpenVerified(hash)
}

func (s *Store) OpenVerified(hash string) (*os.File, error) {
	if !validHash(hash) {
		return nil, errors.New("hash inválido")
	}
	f, err := os.Open(s.Path(hash))
	if err != nil {
		return nil, err
	}
	h := sha256.New()
	if _, err := io.Copy(h, f); err != nil {
		f.Close()
		return nil, err
	}
	if hex.EncodeToString(h.Sum(nil)) != hash {
		f.Close()
		return nil, fmt.Errorf("%w: %s", ErrIntegrity, hash)
	}
	if _, err := f.Seek(0, io.SeekStart); err != nil {
		f.Close()
		return nil, err
	}
	return f, nil
}

func (s *Store) Has(hash string) bool {
	if !validHash(hash) {
		return false
	}
	_, err := os.Stat(s.Path(hash))
	return err == nil
}
