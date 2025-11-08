package store

import (
	"context"
)

func (s *Store) ResetQueue(ctx context.Context) error {
	_, err := s.DB.ExecContext(ctx, `DELETE FROM jobs;`)
	return err
}

func (s *Store) ResetDLQ(ctx context.Context) error {
	_, err := s.DB.ExecContext(ctx, `DELETE FROM dlq;`)
	return err
}
