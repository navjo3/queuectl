package store

import (
	"context"
	"queuectl/internal/model"
	"time"
)

func (s *Store) ListDLQ(ctx context.Context) ([]model.Job, error) {
	rows, err := s.DB.QueryContext(ctx, `
		SELECT id, command, attempts, max_retries, created_at, updated_at
		FROM dlq
		ORDER BY failed_at DESC
	`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var jobs []model.Job

	for rows.Next() {
		var j model.Job
		var createdAtStr, updatedAtStr string

		err := rows.Scan(
			&j.ID,
			&j.Command,
			&j.Attempts,
			&j.MaxRetries,
			&createdAtStr,
			&updatedAtStr,
		)
		if err != nil {
			return nil, err
		}

		j.State = "dead"
		j.CreatedAt, _ = time.Parse(time.RFC3339Nano, createdAtStr)
		j.UpdatedAt, _ = time.Parse(time.RFC3339Nano, updatedAtStr)

		jobs = append(jobs, j)
	}

	return jobs, nil
}

func (s *Store) RetryDLQ(ctx context.Context, jobID string) error {
	// Move job back with attempts reset
	_, err := s.DB.ExecContext(ctx, `
		INSERT INTO jobs (id, command, state, attempts, max_retries, created_at, updated_at, available_at)
		SELECT id, command, 'pending', 0, max_retries, created_at, datetime('now'), datetime('now')
		FROM dlq WHERE id=?;
	`, jobID)
	if err != nil {
		return err
	}

	// Remove from DLQ
	_, err = s.DB.ExecContext(ctx, `DELETE FROM dlq WHERE id=?`, jobID)
	return err
}
