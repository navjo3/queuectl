package engine

import (
	"os"
	"strconv"
)

const pidFile = ".queuectl-pid"

func WritePID(pid int) error {
	return os.WriteFile(pidFile, []byte(strconv.Itoa(pid)), 0644)
}

func ReadPID() (int, error) {
	b, err := os.ReadFile(pidFile)
	if err != nil {
		return 0, err
	}
	pid, err := strconv.Atoi(string(b))
	if err != nil {
		return 0, err
	}
	return pid, nil
}

func RemovePID() {
	_ = os.Remove(pidFile)
}
