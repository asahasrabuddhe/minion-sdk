// +build !windows

package sdk

import "os"

func RenameFile(oldpath, newpath string) error {
	return os.Rename(oldpath, newpath)
}

func RemoveFile(path string) error {
	return os.Remove(path)
}
