// +build !windows

package sdk

import "os"

func RemoveFile(path string) error {
	return os.Remove(path)
}
