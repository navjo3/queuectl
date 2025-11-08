package engine

import (
	"os"
)

const stopFile = ".queuectl-stop"

func ShouldStop() bool {
	_, err := os.Stat(stopFile)
	return err == nil
}

func CreateStopFile() error {
	return os.WriteFile(stopFile, []byte("stop"), 0644)
}

func RemoveStopFile() {
	_ = os.Remove(stopFile)
}

//this file is used for the cross-platform working of the stop function.
//since windows doesnot support stop signal mechanisms.
