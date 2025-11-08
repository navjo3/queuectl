package tests

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"queuectl/internal/model"
	"queuectl/internal/store"
	"time"

	_ "modernc.org/sqlite"
)

// newStore creates a fresh test database in a temporary file
func newStore(t testingT) *store.Store {
	tmpDir := os.TempDir()
	tmpFile := filepath.Join(tmpDir, fmt.Sprintf("test_%d.db", time.Now().UnixNano()))
	
	st, err := store.NewStore(tmpFile)
	if err != nil {
		t.Fatalf("Failed to create test store: %v", err)
	}
	
	// Clean up the temp file after test
	t.Cleanup(func() {
		st.DB.Close()
		os.Remove(tmpFile)
		os.Remove(tmpFile + "-shm")
		os.Remove(tmpFile + "-wal")
	})
	
	return st
}

// enqueueTestJob directly inserts a job into the database for testing
// This should be used instead of calling st.Enqueue directly in tests
func enqueueTestJob(st *store.Store, id, command string, maxRetries int) error {
	ctx := context.Background()
	now := time.Now().UTC()
	
	_, err := st.DB.ExecContext(ctx, `
		INSERT INTO jobs (id, command, state, attempts, max_retries, created_at, updated_at, available_at)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?)
	`, id, command, "pending", 0, maxRetries,
		now.Format(time.RFC3339Nano),
		now.Format(time.RFC3339Nano),
		now.Format(time.RFC3339Nano),
	)
	
	return err
}

// getJob retrieves a job by ID from the database
func getJob(st *store.Store, id string) (*model.Job, error) {
	ctx := context.Background()
	var j model.Job
	var createdAtStr, updatedAtStr, availableAtStr string
	
	err := st.DB.QueryRowContext(ctx, `
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
		return nil, err
	}
	
	j.CreatedAt, _ = time.Parse(time.RFC3339Nano, createdAtStr)
	j.UpdatedAt, _ = time.Parse(time.RFC3339Nano, updatedAtStr)
	j.AvailableAt, _ = time.Parse(time.RFC3339Nano, availableAtStr)
	
	return &j, nil
}

// testingT is a minimal interface for testing.T to allow for easier testing
type testingT interface {
	Fatalf(format string, args ...interface{})
	Cleanup(func())
	Errorf(format string, args ...interface{})
	FailNow()
}


