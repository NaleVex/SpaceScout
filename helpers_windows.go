//go:build windows

package main

import (
	"os/exec"
	"runtime"
	"syscall"
)

func GetDisks() []string {
	switch runtime.GOOS {
	case "windows":
		return GetWindowsDrives()
	default:
		return []string{"/"}
	}
}

func openFolder(path string) error {
	var cmd *exec.Cmd
	switch runtime.GOOS {
	case "windows":
		cmd = exec.Command("explorer", path)
	case "darwin":
		cmd = exec.Command("open", path)
	default: // Linux
		cmd = exec.Command("xdg-open", path)
	}
	return cmd.Run()
}

func GetWindowsDrives() []string {
	var drives []string
	kernel32 := syscall.MustLoadDLL("kernel32.dll")
	getLogicalDrives := kernel32.MustFindProc("GetLogicalDrives")

	// This returns a bitmask (bit 0 = A:, bit 2 = C:, etc.)
	bitmask, _, _ := getLogicalDrives.Call()

	for i := 0; i < 26; i++ {
		if (bitmask >> uint(i) & 1) == 1 {
			drive := string('A'+i) + ":\\"
			drives = append(drives, drive)
		}
	}
	return drives
}
