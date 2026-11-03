package store

import (
	"context"
	"errors"
	"time"

	"nie/internal/domain"
)

var ErrLeaseLost = errors.New("lease do job perdido ou substituído")

func (s *Store) RenewClaimedJob(ctx context.Context, job *domain.Job, lease time.Duration) error {
	tag, err := s.pool.Exec(ctx, `UPDATE jobs SET lease_until=now()+$3::interval,updated_at=now() WHERE id=$1 AND attempts=$2 AND lease_generation=$4 AND status='running' AND lease_until>now()`, job.ID, job.Attempts, lease, job.LeaseGeneration)
	if err != nil {
		return err
	}
	if tag.RowsAffected() != 1 {
		return ErrLeaseLost
	}
	return nil
}

func (s *Store) CompleteClaimedJob(ctx context.Context, job *domain.Job) error {
	return s.transitionClaimedJob(ctx, job, "done", time.Now(), nil, false)
}
func (s *Store) DeadClaimedJob(ctx context.Context, job *domain.Job, cause error) error {
	return s.transitionClaimedJob(ctx, job, "dead", time.Now(), cause, false)
}
func (s *Store) RetryClaimedJob(ctx context.Context, job *domain.Job, delay time.Duration, cause error) error {
	return s.transitionClaimedJob(ctx, job, "pending", time.Now().Add(delay), cause, false)
}
func (s *Store) DeferClaimedJob(ctx context.Context, job *domain.Job, at time.Time) error {
	return s.transitionClaimedJob(ctx, job, "pending", at, nil, true)
}

func (s *Store) transitionClaimedJob(ctx context.Context, job *domain.Job, status string, at time.Time, cause error, deferred bool) error {
	message := ""
	if cause != nil {
		message = cause.Error()
	}
	tag, err := s.pool.Exec(ctx, `UPDATE jobs SET status=$3,scheduled_at=$4,last_error=NULLIF($5,''),lease_until=NULL,updated_at=now(),attempts=CASE WHEN $6 THEN GREATEST(attempts-1,0) ELSE attempts END WHERE id=$1 AND attempts=$2 AND lease_generation=$7 AND status='running' AND lease_until>now()`, job.ID, job.Attempts, status, at, message, deferred, job.LeaseGeneration)
	if err != nil {
		return err
	}
	if tag.RowsAffected() != 1 {
		return ErrLeaseLost
	}
	return nil
}
