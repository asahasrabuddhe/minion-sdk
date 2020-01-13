// +build windows

package sdk

import (
	"fmt"
	"os"
	"syscall"
)

func RemoveFile(path string) error {
	var startupInfo syscall.StartupInfo
	var processInfo syscall.ProcessInformation

	argv, err := syscall.UTF16PtrFromString(fmt.Sprintf("%s\\system32\\cmd.exe /C del %s", os.Getenv("windir"), path))
	if err != nil {
		return err
	}

	return syscall.CreateProcess(nil, argv, bil, nil, true, 0, nil, nil, &startupInfo, &processInfo)
}
