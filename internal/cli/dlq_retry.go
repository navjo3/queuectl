package cli

import (
	"context"
	"fmt"
	"queuectl/internal/store"

	"github.com/spf13/cobra"
)

func NewDLQRetryCmd(st *store.Store) *cobra.Command {
	return &cobra.Command{
		Use:   "retry <jobID>",
		Short: "Move a job from DLQ back to the queue",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			id := args[0]
			if err := st.RetryDLQ(context.Background(), id); err != nil {
				return fmt.Errorf("retry failed: %w", err)
			}
			fmt.Println("Job returned to queue:", id)
			return nil
		},
	}
}
