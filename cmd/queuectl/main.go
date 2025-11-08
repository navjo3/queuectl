package main

import (
	"fmt"
	"os"
	"queuectl/internal/cli"
	"queuectl/internal/store"
)

func main() {
	st, err := store.NewStore("queue.db")
	if err != nil {
		panic(err)
	}
	fmt.Println("Db created")

	root := cli.NewRootCmd()
	root.AddCommand(cli.NewEnqueueCmd(st))
	root.AddCommand(cli.NewListCmd(st))
	root.AddCommand(cli.NewStatusCmd(st))

	workerRoot := cli.NewWorkerRootCmd()
	workerRoot.AddCommand(cli.NewWorkerCmd(st))
	workerRoot.AddCommand(cli.NewWorkerStopCmd())
	root.AddCommand(workerRoot)

	if err := root.Execute(); err != nil {
		os.Exit(1)
	}
}
