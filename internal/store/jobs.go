package store

import (
	"context"
	"database/sql"
	"fmt"
	"math"
	"queuectl/internal/model"
	"time"
)

func (s *Store) Enqueue(ctx context.Context, j model.Job) error {
	now := time.Now().UTC()

	if j.CreatedAt.IsZero() {
		j.CreatedAt = now
	}
	if j.UpdatedAt.IsZero() {
		j.UpdatedAt = now
	}
	if j.AvailableAt.IsZero() {
		j.AvailableAt = now
	}
	if j.State == "" {
		j.State = "pending"
	}
	if j.MaxRetries == 0 {
		// read from config if needed later, for now default to 3
		j.MaxRetries = 3
	}

	_, err := s.DB.ExecContext(ctx, `
INSERT INTO jobs (id, command, state, attempts, max_retries, created_at, updated_at, available_at)
VALUES (?, ?, ?, ?, ?, ?, ?, ?)
`, j.ID, j.Command, j.State, j.Attempts, j.MaxRetries,
		j.CreatedAt.Format(time.RFC3339Nano),
		j.UpdatedAt.Format(time.RFC3339Nano),
		j.AvailableAt.Format(time.RFC3339Nano),
	)

	if err != nil {
		return fmt.Errorf("enqueue failed: %w", err)
	}
	return nil
}

func (s *Store) ClaimOne(ctx context.Context, now time.Time) (*model.Job, error) {
	// SERIALIZABLE = does the safe row-locking we need in SQLite
	tx, err := s.DB.BeginTx(ctx, &sql.TxOptions{Isolation: sql.LevelSerializable})
	if err != nil {
		return nil, fmt.Errorf("begin tx: %w", err)
	}
	defer tx.Rollback()

	var id string
	err = tx.QueryRowContext(ctx, `
		SELECT id 
		FROM jobs
		WHERE state='pending'
		  AND available_at <= ?
		ORDER BY created_at ASC
		LIMIT 1
	`, now.Format(time.RFC3339Nano)).Scan(&id)

	if err == sql.ErrNoRows {
		return nil, nil // no job available
	}
	if err != nil {
		return nil, fmt.Errorf("select pending job: %w", err)
	}

	// Try to mark job as processing — this is the *claim* step
	res, err := tx.ExecContext(ctx, `
		UPDATE jobs
		SET state='processing', updated_at=?
		WHERE id=? AND state='pending'
	`, now.Format(time.RFC3339Nano), id)
	if err != nil {
		return nil, fmt.Errorf("claim update: %w", err)
	}

	// If 0 rows were updated -> another worker grabbed it first
	rows, _ := res.RowsAffected()
	if rows != 1 {
		return nil, nil
	}

	// Load job fields
	var createdAtStr, updatedAtStr, availableAtStr string
	j := &model.Job{}

	err = tx.QueryRowContext(ctx, `
		SELECT id, command, state, attempts, max_retries,
		       created_at, updated_at, available_at
		FROM jobs
		WHERE id=?
	`, id).Scan(
		&j.ID,
		&j.Command,
		&j.State,
		&j.Attempts,
		&j.MaxRetries,
		&createdAtStr,
		&updatedAtStr,
		&availableAtStr,
	)
	if err != nil {
		return nil, fmt.Errorf("reload job after claim: %w", err)
	}

	// Parse timestamps
	j.CreatedAt, _ = time.Parse(time.RFC3339Nano, createdAtStr)
	j.UpdatedAt, _ = time.Parse(time.RFC3339Nano, updatedAtStr)
	j.AvailableAt, _ = time.Parse(time.RFC3339Nano, availableAtStr)

	if err := tx.Commit(); err != nil {
		return nil, fmt.Errorf("tx commit: %w", err)
	}

	return j, nil
}

func (s *Store) Complete(ctx context.Context, id string, now time.Time) error {
	_, err := s.DB.ExecContext(ctx, `
		UPDATE jobs SET state='completed', updated_at=?
		WHERE id=? AND state='processing'
	`, now.Format(time.RFC3339Nano), id)
	return err
}

func (s *Store) FailRetry(ctx context.Context, j *model.Job, now time.Time, base, capSeconds int, execErr error) (bool, error) {
	newAttempts := j.Attempts + 1
	if newAttempts >= j.MaxRetries {
		// Move to DLQ
		_, err := s.DB.ExecContext(ctx, `
			INSERT INTO dlq(id, command, attempts, max_retries, last_error, failed_at, created_at, updated_at)
			SELECT id, command, ?, max_retries, ?, ?, created_at, ?
			FROM jobs WHERE id=?;
		`, newAttempts, execErr.Error(), now.Format(time.RFC3339Nano), now.Format(time.RFC3339Nano), j.ID)
		if err != nil {
			return false, err
		}
		_, err = s.DB.ExecContext(ctx, `DELETE FROM jobs WHERE id=?`, j.ID)
		return true, err
	}

	// Retry with exponential backoff: delay = base^attempts
	delay := time.Duration(int(math.Pow(float64(base), float64(newAttempts)))) * time.Second

	capDur := time.Duration(capSeconds) * time.Second
	if delay > capDur {
		delay = capDur
	}

	available := now.Add(delay)

	_, err := s.DB.ExecContext(ctx, `
		UPDATE jobs
		SET attempts=?, state='pending', available_at=?, updated_at=?
		WHERE id=? AND state='processing'
	`, newAttempts, available.Format(time.RFC3339Nano), now.Format(time.RFC3339Nano), j.ID)

	return false, err
}
