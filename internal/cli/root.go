package cli

import (
	"github.com/spf13/cobra"
)

func NewRootCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "queuectl",
		Short: "Job queue system",
	}
	return cmd
}
