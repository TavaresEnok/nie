package main

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"os/signal"
	"strings"
	"syscall"

	"nie/internal/backup"
)

func main() {
	ctx, cancel := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer cancel()
	if len(os.Args) < 2 {
		fail("use: nie-maint backup [-config arquivo1,arquivo2] | restore SNAPSHOT")
	}
	switch os.Args[1] {
	case "backup":
		flags := flag.NewFlagSet("backup", flag.ExitOnError)
		configs := flags.String("config", "", "arquivos de configuração separados por vírgula")
		flags.Parse(os.Args[2:])
		opts := backup.Options{DatabaseURL: os.Getenv("NIE_DATABASE_URL"), DataDir: os.Getenv("NIE_DATA_DIR")}
		if opts.DatabaseURL == "" || opts.DataDir == "" {
			fail("defina NIE_DATABASE_URL e NIE_DATA_DIR")
		}
		if *configs != "" {
			opts.ConfigFiles = strings.Split(*configs, ",")
		}
		id, err := backup.Create(ctx, opts)
		if err != nil {
			fail(err.Error())
		}
		json.NewEncoder(os.Stdout).Encode(map[string]string{"snapshot_id": id})
	case "restore":
		if len(os.Args) != 3 {
			fail("use: nie-maint restore SNAPSHOT")
		}
		opts := backup.Options{DatabaseURL: os.Getenv("NIE_RESTORE_DATABASE_URL"), DataDir: os.Getenv("NIE_RESTORE_DATA_DIR")}
		if opts.DatabaseURL == "" || opts.DataDir == "" {
			fail("defina NIE_RESTORE_DATABASE_URL e NIE_RESTORE_DATA_DIR para destinos novos")
		}
		manifest, err := backup.Restore(ctx, opts, os.Args[2])
		if err != nil {
			fail(err.Error())
		}
		json.NewEncoder(os.Stdout).Encode(map[string]any{"restored": true, "tables": manifest.Tables, "files": len(manifest.Files)})
	default:
		fail("comando desconhecido")
	}
}

func fail(message string) { fmt.Fprintln(os.Stderr, message); os.Exit(1) }
