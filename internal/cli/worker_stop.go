package cli

import (
	"fmt"
	"queuectl/internal/engine"

	"github.com/spf13/cobra"
)

func NewWorkerStopCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "stop",
		Short: "Gracefully stop running workers",
		RunE: func(cmd *cobra.Command, args []string) error {
			if err := engine.CreateStopFile(); err != nil {
				return fmt.Errorf("failed to request stop: %w", err)
			}
			fmt.Println("Stop requested. Workers will exit after finishing the current job.")
			return nil

		},
	}
}
