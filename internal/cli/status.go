package cli

import (
	"context"
	"fmt"
	"queuectl/internal/store"

	"github.com/spf13/cobra"
)

func NewStatusCmd(st *store.Store) *cobra.Command {
	return &cobra.Command{
		Use:   "status",
		Short: "Show queue status summary",
		RunE: func(cmd *cobra.Command, args []string) error {
			stats, err := st.QueueStatus(context.Background())
			if err != nil {
				return err
			}
			fmt.Println("Queue Status:")
			for state, count := range stats {
				fmt.Printf("  %-10s %d\n", state, count)
			}
			return nil
		},
	}
}
