package model

import "time"

type Job struct {
	ID          string
	Command     string
	State       string
	Attempts    int
	MaxRetries  int
	CreatedAt   time.Time
	UpdatedAt   time.Time
	AvailableAt time.Time
}
