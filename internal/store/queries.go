package store

import (
	"context"
	"queuectl/internal/model"
	"time"
)

func (s *Store) ListJobs(ctx context.Context, state string) ([]model.Job, error) {
	q := `
		SELECT id, command, state, attempts, max_retries,
		       created_at, updated_at, available_at
		FROM jobs
	`
	args := []any{}

	if state != "" {
		q += " WHERE state = ?"
		args = append(args, state)
	}

	rows, err := s.DB.QueryContext(ctx, q, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var result []model.Job
	for rows.Next() {
		var j model.Job
		var createdAtStr, updatedAtStr, availableAtStr string

		if err := rows.Scan(
			&j.ID, &j.Command, &j.State, &j.Attempts, &j.MaxRetries,
			&createdAtStr, &updatedAtStr, &availableAtStr,
		); err != nil {
			return nil, err
		}

		j.CreatedAt, _ = time.Parse(time.RFC3339Nano, createdAtStr)
		j.UpdatedAt, _ = time.Parse(time.RFC3339Nano, updatedAtStr)
		j.AvailableAt, _ = time.Parse(time.RFC3339Nano, availableAtStr)

		result = append(result, j)
	}
	return result, nil
}

func (s *Store) QueueStatus(ctx context.Context) (map[string]int, error) {
	stats := map[string]int{}
	states := []string{"pending", "processing", "completed", "dead"}

	for _, st := range states {
		var count int
		err := s.DB.QueryRowContext(ctx, `SELECT COUNT(*) FROM jobs WHERE state=?`, st).Scan(&count)
		if err != nil {
			return nil, err
		}
		stats[st] = count
	}
	return stats, nil
}
