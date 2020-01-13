// +build windows

package sdk

import (
	"fmt"
	"os"
	"syscall"
)

var (
	startupInfo syscall.StartupInfo
	processInfo syscall.ProcessInformation
)

func RenameFile(oldpath, newpath string) error {
	argv, err := syscall.UTF16PtrFromString(fmt.Sprintf("%s\\system32\\cmd.exe /C ren %s %s", os.Getenv("windir"), oldpath, newpath))
	if err != nil {
		return err
	}

	return syscall.CreateProcess(nil, argv, nil, nil, true, 0, nil, nil, &startupInfo, &processInfo)
}

func RemoveFile(path string) error {
	argv, err := syscall.UTF16PtrFromString(fmt.Sprintf("%s\\system32\\cmd.exe /C del %s", os.Getenv("windir"), path))
	if err != nil {
		return err
	}

	return syscall.CreateProcess(nil, argv, nil, nil, true, 0, nil, nil, &startupInfo, &processInfo)
}
