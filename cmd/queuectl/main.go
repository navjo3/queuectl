package main

import (
	"os"
	"queuectl/internal/cli"
	"queuectl/internal/store"
)

func main() {
	st, err := store.NewStore("queue.db")
	if err != nil {
		panic(err)
	}
	//log.Println("DB Created")

	root := cli.NewRootCmd()
	root.AddCommand(cli.NewEnqueueCmd(st))
	root.AddCommand(cli.NewListCmd(st))
	root.AddCommand(cli.NewStatusCmd(st))
	root.AddCommand(cli.NewResetCmd(st))

	//worker cli's
	workerRoot := cli.NewWorkerRootCmd()
	workerRoot.AddCommand(cli.NewWorkerCmd(st))
	workerRoot.AddCommand(cli.NewWorkerStopCmd())
	root.AddCommand(workerRoot)

	//dlq cli's
	dlqRoot := cli.NewDLQRootCmd()
	dlqRoot.AddCommand(cli.NewDLQListCmd(st))
	dlqRoot.AddCommand(cli.NewDLQRetryCmd(st))
	root.AddCommand(dlqRoot)

	//config cli's
	configRoot := cli.NewConfigRootCmd()
	configRoot.AddCommand(cli.NewConfigSetCmd(st))
	configRoot.AddCommand(cli.NewConfigGetCmd(st))
	root.AddCommand(configRoot)

	if err := root.Execute(); err != nil {
		os.Exit(1)
	}
}
