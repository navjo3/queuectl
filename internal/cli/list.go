package cli

import (
	"context"
	"fmt"
	"queuectl/internal/store"

	"github.com/spf13/cobra"
)

func NewListCmd(st *store.Store) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "list",
		Short: "List jobs in the queue",
		RunE: func(cmd *cobra.Command, args []string) error {
			rows, err := st.DB.QueryContext(context.Background(),
				"SELECT id, state, attempts, command FROM jobs ORDER BY created_at ASC")
			if err != nil {
				return err
			}
			defer rows.Close()

			for rows.Next() {
				var id, state, command string
				var attempts int
				if err := rows.Scan(&id, &state, &attempts, &command); err != nil {
					return err
				}
				fmt.Printf("%s | %s | attempts=%d | %s\n", id, state, attempts, command)
			}
			return nil
		},
	}
	return cmd
}
