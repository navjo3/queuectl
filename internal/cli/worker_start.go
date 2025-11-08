package cli

import (
	"context"
	"fmt"
	"os"
	"os/signal"

	"queuectl/internal/engine"
	"queuectl/internal/store"
	"strconv"

	"github.com/spf13/cobra"
)

func NewWorkerCmd(st *store.Store) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "start",
		Short: "Start worker processes",
		RunE: func(cmd *cobra.Command, args []string) error {
			engine.RemoveStopFile()

			countStr, _ := cmd.Flags().GetString("count")
			count, err := strconv.Atoi(countStr)
			if err != nil || count < 1 {
				return fmt.Errorf("invalid worker count: %s", countStr)
			}

			ctx, cancel := context.WithCancel(context.Background())

			// Start workers
			for i := 0; i < count; i++ {
				go engine.NewWorker(st).Run(ctx)
			}

			fmt.Printf("Started %d workers (PID: %d). Use `queuectl worker stop` to stop.\n", count, os.Getpid())

			// Handle OS signals for graceful shutdown
			sigCh := make(chan os.Signal, 1)
			signal.Notify(sigCh, os.Interrupt)

			<-sigCh // wait for stop signal
			fmt.Println("Stopping workers gracefully...")
			cancel()
			return nil
		},
	}

	cmd.Flags().String("count", "1", "number of workers to start")
	return cmd
}
