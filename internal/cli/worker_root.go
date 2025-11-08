package cli

import "github.com/spf13/cobra"

func NewWorkerRootCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "worker",
		Short: "Manage worker processes",
	}
}
