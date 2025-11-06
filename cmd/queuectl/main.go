package main

import (
	"fmt"
	"queuectl/internal/store"
)

func main() {
	st, err := store.NewStore("queue.db")
	if err != nil {
		panic(err)
	}
	fmt.Println("Db created")

	_ = st
}
