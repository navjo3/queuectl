package cli

import (
	"context"
	"encoding/json"
	"fmt"
	"queuectl/internal/model"
	"queuectl/internal/store"
	"time"

	"github.com/spf13/cobra"
)

func NewEnqueueCmd(st *store.Store) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "enqueue '{\"id\":\"job1\",\"command\":\"sleep 2\"}'",
		Short: "Add a job to the queue",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			var j model.Job
			if err := json.Unmarshal([]byte(args[0]), &j); err != nil {
				return fmt.Errorf("invalid job json: %w", err)
			}

			// Fill defaults
			j.State = "pending"
			j.Attempts = 0
			j.CreatedAt = time.Now().UTC()
			j.UpdatedAt = j.CreatedAt
			j.AvailableAt = j.CreatedAt
			if j.MaxRetries == 0 {
				j.MaxRetries = 3
			}

			err := st.Enqueue(context.Background(), j)
			if err != nil {
				return err
			}

			fmt.Println("Job enqueued:", j.ID)
			return nil
		},
	}
	return cmd
}
