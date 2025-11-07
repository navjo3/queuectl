package cli

import (
	"context"
	"fmt"
	"queuectl/internal/engine"
	"queuectl/internal/store"
	"strconv"

	"github.com/spf13/cobra"
)

func NewWorkerCmd(st *store.Store) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "worker start",
		Short: "Start worker processes",
		RunE: func(cmd *cobra.Command, args []string) error {
			countStr, _ := cmd.Flags().GetString("count")
			count, err := strconv.Atoi(countStr)
			if err != nil || count < 1 {
				return fmt.Errorf("invalid worker count: %s", countStr)
			}

			ctx, cancel := context.WithCancel(context.Background())
			defer cancel()

			for i := 0; i < count; i++ {
				go engine.NewWorker(st).Run(ctx)
			}

			fmt.Printf("Started %d workers.\nPress Ctrl+C to stop.\n", count)
			<-ctx.Done()
			return nil
		},
	}

	cmd.Flags().String("count", "1", "number of workers to start")

	return cmd
}
