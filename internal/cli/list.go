package cli

import (
	"context"
	"fmt"
	"queuectl/internal/store"

	"github.com/spf13/cobra"
)

func NewListCmd(st *store.Store) *cobra.Command {
	var state string

	cmd := &cobra.Command{
		Use:   "list",
		Short: "List jobs in the queue",
		RunE: func(cmd *cobra.Command, args []string) error {
			jobs, err := st.ListJobs(context.Background(), state)
			if err != nil {
				return err
			}

			if len(jobs) == 0 {
				fmt.Println("No jobs found.")
				return nil
			}

			for _, j := range jobs {
				fmt.Printf("%s | %-10s | attempts=%d/%d | %s\n",
					j.ID, j.State, j.Attempts, j.MaxRetries, j.Command)
			}
			return nil
		},
	}

	cmd.Flags().StringVar(&state, "state", "", "Filter by job state (pending,processing,completed,dead)")
	return cmd
}
