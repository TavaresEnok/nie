// Package backup coordinates PostgreSQL snapshots and immutable CAS objects
// with Restic. Restore only publishes into an empty database and a new directory.
package backup

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"io/fs"
	"net/url"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"github.com/jackc/pgx/v5"
	"nie/internal/storage"
)

type Options struct {
	DatabaseURL string
	DataDir     string
	ConfigFiles []string
	KeepDaily   int
	KeepWeekly  int
	KeepMonthly int
}

// RequireOffsiteRepository recusa repositório vazio ou local. A cópia de
// backup só conta quando o Restic aponta para fora deste host.
func RequireOffsiteRepository() error {
	repo := strings.TrimSpace(os.Getenv("RESTIC_REPOSITORY"))
	password := strings.TrimSpace(os.Getenv("RESTIC_PASSWORD"))
	if repo == "" || password == "" {
		return errors.New("defina RESTIC_REPOSITORY remoto e RESTIC_PASSWORD")
	}
	lower := strings.ToLower(repo)
	if strings.HasPrefix(repo, "/") || strings.HasPrefix(repo, ".") || strings.HasPrefix(lower, "file:") || strings.HasPrefix(lower, "local:") {
		return errors.New("repositório local não conta como cópia fora do host")
	}
	return nil
}

type File struct {
	Path   string `json:"path"`
	SHA256 string `json:"sha256"`
	Size   int64  `json:"size"`
}

type Table struct {
	Rows   int64  `json:"rows"`
	SHA256 string `json:"sha256"`
}
type Manifest struct {
	Format     string           `json:"format"`
	CreatedAt  time.Time        `json:"created_at"`
	PostgreSQL string           `json:"postgresql"`
	Tables     map[string]Table `json:"tables"`
	Files      []File           `json:"files"`
}

type objectReference struct {
	hash string
	size *int64
}

func references(ctx context.Context, tx pgx.Tx) ([]objectReference, error) {
	rows, err := tx.Query(ctx, `SELECT hash,max(size) FROM (SELECT sha256 AS hash,size FROM documents UNION ALL SELECT input_sha256,NULL::bigint FROM extraction_runs WHERE input_sha256 IS NOT NULL) r GROUP BY hash ORDER BY hash`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var refs []objectReference
	for rows.Next() {
		var ref objectReference
		if err := rows.Scan(&ref.hash, &ref.size); err != nil {
			return nil, err
		}
		refs = append(refs, ref)
	}
	return refs, rows.Err()
}

func pgCommand(ctx context.Context, name, dsn string, args ...string) (*exec.Cmd, error) {
	cfg, err := pgx.ParseConfig(dsn)
	if err != nil {
		return nil, errors.New("URL PostgreSQL inválida")
	}
	command := exec.CommandContext(ctx, name, append([]string{"--dbname=" + cfg.Database}, args...)...)
	sslMode := "prefer"
	if u, err := url.Parse(dsn); err == nil && u.Query().Get("sslmode") != "" {
		sslMode = u.Query().Get("sslmode")
	}
	command.Env = append(os.Environ(), "PGHOST="+cfg.Host, fmt.Sprintf("PGPORT=%d", cfg.Port), "PGUSER="+cfg.User, "PGPASSWORD="+cfg.Password, "PGDATABASE="+cfg.Database, "PGSSLMODE="+sslMode, "PGAPPNAME=nie-maint")
	return command, nil
}

func run(command *exec.Cmd) ([]byte, error) {
	var stderr bytes.Buffer
	command.Stderr = &stderr
	output, err := command.Output()
	if err != nil {
		return nil, fmt.Errorf("%s falhou; confira versão, conexão e permissões (%T)", filepath.Base(command.Path), err)
	}
	return output, nil
}

func digestFile(name string) (File, error) {
	f, err := os.Open(name)
	if err != nil {
		return File{}, err
	}
	defer f.Close()
	h := sha256.New()
	n, err := io.Copy(h, f)
	if err != nil {
		return File{}, err
	}
	return File{SHA256: hex.EncodeToString(h.Sum(nil)), Size: n}, nil
}

func catalog(ctx context.Context, tx pgx.Tx) (map[string]Table, error) {
	if _, err := tx.Exec(ctx, `SET LOCAL TIME ZONE 'UTC'`); err != nil {
		return nil, err
	}
	rows, err := tx.Query(ctx, `SELECT tablename FROM pg_tables WHERE schemaname='public' ORDER BY tablename`)
	if err != nil {
		return nil, err
	}
	var names []string
	for rows.Next() {
		var name string
		if err := rows.Scan(&name); err != nil {
			rows.Close()
			return nil, err
		}
		names = append(names, name)
	}
	rows.Close()
	if err := rows.Err(); err != nil {
		return nil, err
	}
	out := map[string]Table{}
	for _, name := range names {
		rows, err := tx.Query(ctx, `SELECT to_jsonb(t)::text FROM `+pgx.Identifier{"public", name}.Sanitize()+` t ORDER BY to_jsonb(t)::text COLLATE "C"`)
		if err != nil {
			return nil, err
		}
		h := sha256.New()
		var n int64
		for rows.Next() {
			var value string
			if err := rows.Scan(&value); err != nil {
				rows.Close()
				return nil, err
			}
			io.WriteString(h, value+"\n")
			n++
		}
		rows.Close()
		if err := rows.Err(); err != nil {
			return nil, err
		}
		out[name] = Table{Rows: n, SHA256: hex.EncodeToString(h.Sum(nil))}
	}
	return out, nil
}

// Create uses the same exported snapshot for pg_dump, table checksums and CAS
// references. Hard links avoid duplicating the corpus on the staging disk.
func Create(ctx context.Context, opts Options) (string, error) {
	root, err := filepath.Abs(opts.DataDir)
	if err != nil {
		return "", err
	}
	objects, err := storage.New(root)
	if err != nil {
		return "", err
	}
	stage, err := os.MkdirTemp(root, ".nie-backup-")
	if err != nil {
		return "", err
	}
	defer os.RemoveAll(stage)
	bundle := filepath.Join(stage, "bundle")
	if err := os.MkdirAll(bundle, 0700); err != nil {
		return "", err
	}
	conn, err := pgx.Connect(ctx, opts.DatabaseURL)
	if err != nil {
		return "", errors.New("conexão de backup indisponível")
	}
	defer conn.Close(ctx)
	tx, err := conn.BeginTx(ctx, pgx.TxOptions{IsoLevel: pgx.RepeatableRead, AccessMode: pgx.ReadOnly})
	if err != nil {
		return "", err
	}
	defer tx.Rollback(ctx)
	if _, err := tx.Exec(ctx, `SELECT pg_advisory_xact_lock_shared(918273645)`); err != nil {
		return "", err
	}
	manifest := Manifest{Format: "nie-backup-v1", CreatedAt: time.Now().UTC()}
	var snapshot string
	if err := tx.QueryRow(ctx, `SELECT pg_export_snapshot(),version()`).Scan(&snapshot, &manifest.PostgreSQL); err != nil {
		return "", err
	}
	manifest.Tables, err = catalog(ctx, tx)
	if err != nil {
		return "", err
	}
	if _, ok := manifest.Tables["extraction_runs"]; !ok {
		return "", errors.New("banco precisa da migração de extrações versionadas")
	}
	refs, err := references(ctx, tx)
	if err != nil {
		return "", err
	}
	for _, ref := range refs {
		hash := ref.hash
		f, err := objects.OpenVerified(hash)
		if err != nil {
			return "", fmt.Errorf("original ou derivado sem integridade: %w", err)
		}
		stat, err := f.Stat()
		f.Close()
		if err != nil {
			return "", err
		}
		if !stat.Mode().IsRegular() || (ref.size != nil && stat.Size() != *ref.size) {
			return "", fmt.Errorf("objeto diverge do catálogo: %s", hash)
		}
		relative := filepath.Join("data", "objects", "sha256", hash[:2], hash[2:4], hash)
		target := filepath.Join(bundle, relative)
		if err := os.MkdirAll(filepath.Dir(target), 0700); err != nil {
			return "", err
		}
		if err := os.Link(objects.Path(hash), target); err != nil {
			return "", fmt.Errorf("staging exige hard links no filesystem do acervo: %w", err)
		}
		manifest.Files = append(manifest.Files, File{Path: filepath.ToSlash(relative), SHA256: hash, Size: stat.Size()})
	}
	dumpPath := filepath.Join(bundle, "database.dump")
	dump, err := pgCommand(ctx, "pg_dump", opts.DatabaseURL, "--format=custom", "--no-owner", "--no-acl", "--snapshot="+snapshot, "--file="+dumpPath)
	if err != nil {
		return "", err
	}
	if _, err := run(dump); err != nil {
		return "", err
	}
	file, err := digestFile(dumpPath)
	if err != nil {
		return "", err
	}
	file.Path = "database.dump"
	manifest.Files = append(manifest.Files, file)
	for i, path := range opts.ConfigFiles {
		relative := filepath.Join("data", "recovered-config", fmt.Sprintf("%03d-%s", i, filepath.Base(path)))
		target := filepath.Join(bundle, relative)
		if err := os.MkdirAll(filepath.Dir(target), 0700); err != nil {
			return "", err
		}
		source, err := os.Open(path)
		if err != nil {
			return "", err
		}
		dest, err := os.OpenFile(target, os.O_CREATE|os.O_EXCL|os.O_WRONLY, 0600)
		if err != nil {
			source.Close()
			return "", err
		}
		_, copyErr := io.Copy(dest, source)
		closeErr := dest.Close()
		source.Close()
		if copyErr != nil {
			return "", copyErr
		}
		if closeErr != nil {
			return "", closeErr
		}
		file, err := digestFile(target)
		if err != nil {
			return "", err
		}
		file.Path = filepath.ToSlash(relative)
		manifest.Files = append(manifest.Files, file)
	}
	if err := tx.Commit(ctx); err != nil {
		return "", err
	}
	manifestBytes, err := json.MarshalIndent(manifest, "", "  ")
	if err != nil {
		return "", err
	}
	if err := os.WriteFile(filepath.Join(bundle, "manifest.json"), manifestBytes, 0600); err != nil {
		return "", err
	}
	restic := exec.CommandContext(ctx, "restic", "backup", "--json", "--tag", "nie-backup-v1", bundle)
	output, err := run(restic)
	if err != nil {
		return "", err
	}
	decoder := json.NewDecoder(bytes.NewReader(output))
	var id string
	for {
		var event struct {
			SnapshotID string `json:"snapshot_id"`
		}
		err := decoder.Decode(&event)
		if errors.Is(err, io.EOF) {
			break
		}
		if err != nil {
			return "", err
		}
		if event.SnapshotID != "" {
			id = event.SnapshotID
		}
	}
	if id == "" {
		return "", errors.New("Restic não retornou ID do snapshot")
	}
	if opts.KeepDaily > 0 || opts.KeepWeekly > 0 || opts.KeepMonthly > 0 {
		forget := exec.CommandContext(ctx, "restic", "forget", "--prune", "--tag", "nie-backup-v1",
			"--keep-daily", fmt.Sprintf("%d", opts.KeepDaily),
			"--keep-weekly", fmt.Sprintf("%d", opts.KeepWeekly),
			"--keep-monthly", fmt.Sprintf("%d", opts.KeepMonthly))
		if _, err := run(forget); err != nil {
			return "", err
		}
	}
	return id, nil
}

// Restore refuses existing data directories and non-empty target databases.
// It verifies every file, then all table values, before publishing the data dir.
func Restore(ctx context.Context, opts Options, snapshot string) (Manifest, error) {
	var manifest Manifest
	if len(snapshot) < 8 || len(snapshot) > 64 {
		return manifest, errors.New("informe ID explícito do snapshot Restic")
	}
	if _, err := hex.DecodeString(snapshot); err != nil {
		return manifest, errors.New("ID de snapshot inválido")
	}
	target, err := filepath.Abs(opts.DataDir)
	if err != nil {
		return manifest, err
	}
	if _, err := os.Stat(target); !errors.Is(err, os.ErrNotExist) {
		return manifest, errors.New("diretório de restauração deve ser novo")
	}
	conn, err := pgx.Connect(ctx, opts.DatabaseURL)
	if err != nil {
		return manifest, errors.New("conexão de restauração indisponível")
	}
	defer conn.Close(ctx)
	var count int
	if err := conn.QueryRow(ctx, `SELECT count(*) FROM pg_tables WHERE schemaname NOT IN ('pg_catalog','information_schema')`).Scan(&count); err != nil {
		return manifest, err
	}
	if count != 0 {
		return manifest, errors.New("banco de restauração deve estar vazio")
	}
	if err := os.MkdirAll(filepath.Dir(target), 0700); err != nil {
		return manifest, err
	}
	stage, err := os.MkdirTemp(filepath.Dir(target), ".nie-restore-")
	if err != nil {
		return manifest, err
	}
	defer os.RemoveAll(stage)
	if _, err := run(exec.CommandContext(ctx, "restic", "restore", snapshot, "--target", stage)); err != nil {
		return manifest, err
	}
	var manifests []string
	if err := filepath.WalkDir(stage, func(path string, entry fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if entry.Type()&os.ModeSymlink != 0 {
			return errors.New("snapshot contém link simbólico")
		}
		if entry.Name() == "manifest.json" && !entry.IsDir() {
			manifests = append(manifests, path)
		}
		return nil
	}); err != nil {
		return manifest, err
	}
	if len(manifests) != 1 {
		return manifest, errors.New("snapshot não contém um único manifesto NIE")
	}
	bundle := filepath.Dir(manifests[0])
	content, err := os.ReadFile(manifests[0])
	if err != nil {
		return manifest, err
	}
	if err := json.Unmarshal(content, &manifest); err != nil {
		return manifest, err
	}
	if manifest.Format != "nie-backup-v1" {
		return manifest, errors.New("formato de backup desconhecido")
	}
	seen := map[string]bool{}
	entries := map[string]File{}
	for _, file := range manifest.Files {
		relative := filepath.FromSlash(file.Path)
		if filepath.IsAbs(relative) || filepath.VolumeName(relative) != "" || strings.Contains(file.Path, "\\") || filepath.ToSlash(filepath.Clean(relative)) != file.Path || strings.HasPrefix(filepath.Clean(relative), "..") || seen[file.Path] {
			return manifest, errors.New("caminho inválido ou duplicado no manifesto")
		}
		seen[file.Path] = true
		entries[file.Path] = file
		path := filepath.Join(bundle, relative)
		stat, err := os.Lstat(path)
		if err != nil || !stat.Mode().IsRegular() {
			return manifest, errors.New("arquivo ausente ou não regular no backup")
		}
		actual, err := digestFile(path)
		if err != nil {
			return manifest, err
		}
		if actual.SHA256 != file.SHA256 || actual.Size != file.Size {
			return manifest, fmt.Errorf("integridade divergente: %s", file.Path)
		}
	}
	if !seen["database.dump"] {
		return manifest, errors.New("dump ausente do manifesto")
	}
	if err := filepath.WalkDir(bundle, func(path string, entry fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if entry.IsDir() {
			return nil
		}
		relative, err := filepath.Rel(bundle, path)
		if err != nil {
			return err
		}
		if filepath.ToSlash(relative) != "manifest.json" && !seen[filepath.ToSlash(relative)] {
			return errors.New("snapshot contém arquivo fora do manifesto")
		}
		return nil
	}); err != nil {
		return manifest, err
	}
	restore, err := pgCommand(ctx, "pg_restore", opts.DatabaseURL, "--single-transaction", "--exit-on-error", "--no-owner", "--no-acl", filepath.Join(bundle, "database.dump"))
	if err != nil {
		return manifest, err
	}
	if _, err := run(restore); err != nil {
		return manifest, err
	}
	tx, err := conn.BeginTx(ctx, pgx.TxOptions{IsoLevel: pgx.RepeatableRead, AccessMode: pgx.ReadOnly})
	if err != nil {
		return manifest, err
	}
	defer tx.Rollback(ctx)
	actual, err := catalog(ctx, tx)
	if err != nil {
		return manifest, err
	}
	names := make([]string, 0, len(manifest.Tables))
	for name := range manifest.Tables {
		names = append(names, name)
	}
	sort.Strings(names)
	if len(actual) != len(manifest.Tables) {
		return manifest, errors.New("conjunto de tabelas restaurado diverge do manifesto")
	}
	for _, name := range names {
		if actual[name] != manifest.Tables[name] {
			return manifest, fmt.Errorf("conteúdo restaurado diverge na tabela %s", name)
		}
	}
	refs, err := references(ctx, tx)
	if err != nil {
		return manifest, err
	}
	for _, ref := range refs {
		if len(ref.hash) != 64 {
			return manifest, errors.New("hash inválido no catálogo restaurado")
		}
		relative := fmt.Sprintf("data/objects/sha256/%s/%s/%s", ref.hash[:2], ref.hash[2:4], ref.hash)
		entry, ok := entries[relative]
		if !ok || entry.SHA256 != ref.hash || (ref.size != nil && entry.Size != *ref.size) {
			return manifest, fmt.Errorf("objeto referenciado ausente ou divergente: %s", ref.hash)
		}
	}
	if err := tx.Commit(ctx); err != nil {
		return manifest, err
	}
	data := filepath.Join(bundle, "data")
	if err := os.MkdirAll(data, 0700); err != nil {
		return manifest, err
	}
	if err := os.Rename(data, target); err != nil {
		return manifest, err
	}
	return manifest, nil
}
