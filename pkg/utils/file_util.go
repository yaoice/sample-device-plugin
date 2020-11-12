package utils

import (
	"os"
	"syscall"
)

func UnlinkFile(path string) error {
	_, err := os.Stat(path)
	if err != nil {
		return nil
	}
	return syscall.Unlink(path)
}
