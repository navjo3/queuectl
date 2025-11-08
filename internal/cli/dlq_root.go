package cli

import "github.com/spf13/cobra"

func NewDLQRootCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "dlq",
		Short: "Manage dead letter queue",
	}
}
