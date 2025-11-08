package cli

import (
	"context"
	"fmt"
	"queuectl/internal/store"

	"github.com/spf13/cobra"
)

func NewConfigSetCmd(st *store.Store) *cobra.Command {
	return &cobra.Command{
		Use:   "set <key> <value>",
		Args:  cobra.ExactArgs(2),
		Short: "Set a config value",
		RunE: func(cmd *cobra.Command, args []string) error {
			key, value := args[0], args[1]
			if err := st.SetConfig(context.Background(), key, value); err != nil {
				return fmt.Errorf("failed to set config: %w", err)
			}
			fmt.Println("Updated:", key, "=", value)
			return nil
		},
	}
}
