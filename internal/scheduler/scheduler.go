package scheduler

import (
	"context"
	"fmt"
	"log/slog"
	"sync"
	"time"

	"nie/internal/domain"
	"nie/internal/store"
)

type Runner struct {
	store    *store.Store
	logger   *slog.Logger
	interval time.Duration
	stopCh   chan struct{}
	wg       sync.WaitGroup
}

func New(st *store.Store, logger *slog.Logger, interval time.Duration) *Runner {
	if interval < 5*time.Second {
		interval = 30 * time.Second
	}
	return &Runner{
		store:    st,
		logger:   logger,
		interval: interval,
		stopCh:   make(chan struct{}),
	}
}

func (r *Runner) Start(ctx context.Context) {
	r.wg.Add(1)
	go func() {
		defer r.wg.Done()
		ticker := time.NewTicker(r.interval)
		defer ticker.Stop()

		r.logger.Info("scheduler de autonomia contínua iniciado", "interval", r.interval)
		for {
			select {
			case <-ctx.Done():
				return
			case <-r.stopCh:
				return
			case <-ticker.C:
				if err := r.tick(ctx); err != nil {
					r.logger.Error("erro no ciclo do scheduler", "err", err)
				}
			}
		}
	}()
}

func (r *Runner) Stop() {
	close(r.stopCh)
	r.wg.Wait()
}

func (r *Runner) tick(ctx context.Context) error {
	schedules, err := r.store.ListDueSchedules(ctx)
	if err != nil {
		return fmt.Errorf("listar agendamentos vencidos: %w", err)
	}

	for _, s := range schedules {
		r.logger.Info("disparando campanha agendada", "schedule_id", s.ID, "nome", s.Name, "query", s.Query)
		camp, err := r.store.CreateCampaign(ctx, s.Query, s.SeedURLs, domain.Budget{
			MaxDepth:                 s.MaxDepth,
			MaxURLs:                  s.MaxURLs,
			MaxDomains:               s.MaxDomains,
			MaxBytes:                 1 << 30, // 1 GiB
			MinDomainIntervalSeconds: 10,
		})
		if err != nil {
			r.logger.Error("falha ao criar campanha agendada", "schedule_id", s.ID, "err", err)
			continue
		}

		nextRun := time.Now().Add(time.Duration(s.IntervalSeconds) * time.Second)
		if err := r.store.MarkScheduleRun(ctx, s.ID, camp.ID, nextRun); err != nil {
			r.logger.Error("falha ao atualizar próximo disparo do agendamento", "schedule_id", s.ID, "err", err)
		}

		// Registrar alerta informativo de disparo
		_ = r.store.CreateAlert(ctx, &camp.ID, fmt.Sprintf("Agendamento disparado: %s", s.Name), fmt.Sprintf("Campanha '%s' criada automaticamente.", camp.ID), "info")
	}
	return nil
}
