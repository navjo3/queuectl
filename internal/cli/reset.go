package cli

import (
	"context"
	"fmt"
	"queuectl/internal/store"

	"github.com/spf13/cobra"
)

func NewResetCmd(st *store.Store) *cobra.Command {
	return &cobra.Command{
		Use:   "reset",
		Short: "Clear all jobs and DLQ entries (development only)",
		RunE: func(cmd *cobra.Command, args []string) error {

			if err := st.ResetQueue(context.Background()); err != nil {
				return fmt.Errorf("failed to clear jobs: %w", err)
			}
			if err := st.ResetDLQ(context.Background()); err != nil {
				return fmt.Errorf("failed to clear DLQ: %w", err)
			}

			fmt.Println("Queue and DLQ cleared.")
			return nil
		},
	}
}
