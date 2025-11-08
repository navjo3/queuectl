package store

import (
	"context"
	"database/sql"
	"strconv"
)

// file for config cli functions
func (s *Store) SetConfig(ctx context.Context, key, value string) error {
	_, err := s.DB.ExecContext(ctx, `
		INSERT INTO config (key, value) VALUES (?, ?)
		ON CONFLICT(key) DO UPDATE SET value=excluded.value
	`, key, value)
	return err
}

func (s *Store) GetConfig(ctx context.Context, key string) (string, error) {
	var val string
	err := s.DB.QueryRowContext(ctx, `SELECT value FROM config WHERE key=?`, key).Scan(&val)
	if err == sql.ErrNoRows {
		return "", nil
	}
	return val, err
}

func (s *Store) AllConfig(ctx context.Context) (map[string]string, error) {
	rows, err := s.DB.QueryContext(ctx, `SELECT key, value FROM config`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	result := map[string]string{}
	for rows.Next() {
		var k, v string
		_ = rows.Scan(&k, &v)
		result[k] = v
	}
	return result, nil
}

func (s *Store) MustGetInt(key string, defaultVal int) int {
	val, err := s.GetConfig(context.Background(), key)
	if err != nil || val == "" {
		return defaultVal
	}
	n, err := strconv.Atoi(val)
	if err != nil {
		return defaultVal
	}
	return n
}
