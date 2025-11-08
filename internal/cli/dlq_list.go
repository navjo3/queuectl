package cli

import (
	"context"
	"fmt"
	"queuectl/internal/store"

	"github.com/spf13/cobra"
)

func NewDLQListCmd(st *store.Store) *cobra.Command {
	return &cobra.Command{
		Use:   "list",
		Short: "List jobs in the dead letter queue",
		RunE: func(cmd *cobra.Command, args []string) error {
			jobs, err := st.ListDLQ(context.Background())
			if err != nil {
				return err
			}

			if len(jobs) == 0 {
				fmt.Println("No jobs in DLQ.")
				return nil
			}

			for _, j := range jobs {
				fmt.Printf("%s | attempts=%d/%d | command=%s\n",
					j.ID, j.Attempts, j.MaxRetries, j.Command)
			}
			return nil
		},
	}
}
