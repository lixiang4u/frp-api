package utils

import (
	"os"
)

func FileExists(files ...string) bool {
	if len(files) == 0 {
		return false
	}
	for _, file := range files {
		_, err := os.Stat(file)
		if err != nil {
			return false
		}
	}
	return true
}
