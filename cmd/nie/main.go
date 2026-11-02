package main

import (
	"context"
	"errors"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"

	"nie/internal/api"
	"nie/internal/config"
	"nie/internal/crawler"
	"nie/internal/discovery"
	"nie/internal/scheduler"
	"nie/internal/storage"
	"nie/internal/store"
	"nie/internal/version"
	"nie/internal/worker"
)

func main() {
	cfg := config.Load()
	logger := newLogger(cfg.LogLevel)
	logger.Info("iniciando NIE", "version", version.Version, "addr", cfg.Addr, "data", cfg.DataDir, "workers", cfg.WorkerCount)

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	st, err := connect(ctx, cfg.DatabaseURL, logger)
	if err != nil {
		logger.Error("banco indisponível", "err", err)
		os.Exit(1)
	}
	defer st.Close()
	if err := st.Migrate(ctx); err != nil {
		logger.Error("migrações", "err", err)
		os.Exit(1)
	}

	objects, err := storage.New(cfg.DataDir)
	if err != nil {
		logger.Error("storage", "err", err)
		os.Exit(1)
	}

	fetcher := crawler.NewFetcher(cfg.UserAgent, cfg.RequestTimeout, cfg.MaxHTMLBytes, cfg.MaxPDFBytes, cfg.MaxTextBytes)
	robots := crawler.NewRobots(fetcher, cfg.UserAgent, logger)
	var searx *discovery.SearXNG
	if cfg.SearXNGURL != "" {
		searx = discovery.NewSearXNG(cfg.SearXNGURL)
		logger.Info("descoberta web habilitada", "searxng", cfg.SearXNGURL)
	}
	gh := discovery.NewGitHub(cfg.GitHubToken, cfg.GitHubEnabled)
	if cfg.GitHubEnabled {
		logger.Info("descoberta GitHub habilitada", "token", cfg.GitHubToken != "")
	}

	w := worker.New(st, objects, fetcher, robots, searx, gh, cfg, logger)
	go w.Run(ctx)

	if cfg.SchedulerEnabled {
		sched := scheduler.New(st, logger, cfg.SchedulerInterval)
		sched.Start(ctx)
		defer sched.Stop()
	}

	handler := api.New(st, cfg, logger)
	srv := &http.Server{
		Addr:              cfg.Addr,
		Handler:           handler,
		ReadHeaderTimeout: 10 * time.Second,
		ReadTimeout:       30 * time.Second,
		WriteTimeout:      60 * time.Second,
		IdleTimeout:       120 * time.Second,
	}
	errCh := make(chan error, 1)
	go func() {
		logger.Info("API disponível", "url", "http://"+cfg.Addr, "docs", "http://"+cfg.Addr+"/api/docs")
		if err := srv.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			errCh <- err
		}
	}()

	select {
	case <-ctx.Done():
		logger.Info("encerrando por sinal")
	case err := <-errCh:
		logger.Error("servidor HTTP", "err", err)
	}

	shutdownCtx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()
	if err := srv.Shutdown(shutdownCtx); err != nil {
		logger.Error("shutdown", "err", err)
	}
}

func connect(ctx context.Context, dsn string, logger *slog.Logger) (*store.Store, error) {
	var last error
	for i := 0; i < 30; i++ {
		if ctx.Err() != nil {
			return nil, ctx.Err()
		}
		st, err := store.New(ctx, dsn)
		if err == nil {
			pingCtx, cancel := context.WithTimeout(ctx, 3*time.Second)
			err = st.Ping(pingCtx)
			cancel()
			if err == nil {
				return st, nil
			}
			st.Close()
		}
		last = err
		logger.Warn("aguardando PostgreSQL", "attempt", i+1, "err", err)
		select {
		case <-ctx.Done():
			return nil, ctx.Err()
		case <-time.After(2 * time.Second):
		}
	}
	return nil, last
}

func newLogger(level string) *slog.Logger {
	var l slog.Level
	switch strings.ToLower(level) {
	case "debug":
		l = slog.LevelDebug
	case "warn":
		l = slog.LevelWarn
	case "error":
		l = slog.LevelError
	default:
		l = slog.LevelInfo
	}
	return slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{Level: l}))
}
