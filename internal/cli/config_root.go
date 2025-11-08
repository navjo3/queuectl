package cli

import "github.com/spf13/cobra"

func NewConfigRootCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "config",
		Short: "Basic configuration: set, get",
	}
}
