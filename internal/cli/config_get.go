package cli

import (
	"context"
	"fmt"
	"queuectl/internal/store"

	"github.com/spf13/cobra"
)

func NewConfigGetCmd(st *store.Store) *cobra.Command {
	return &cobra.Command{
		Use:   "get <key>",
		Args:  cobra.ExactArgs(1),
		Short: "Get a default/current config value",
		RunE: func(cmd *cobra.Command, args []string) error {
			val, err := st.GetConfig(context.Background(), args[0])
			if err != nil {
				return err
			}
			if val == "" {
				fmt.Println("(not set)")
			} else {
				fmt.Println(val)
			}
			return nil
		},
	}
}
