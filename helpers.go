package main

import (
	"os"
	"runtime"
	"syscall"
)

func GetDisks() []string {
	switch runtime.GOOS {
	case "windows":
		return getWindowsDrives()
	case "darwin": // macOS
		return getUnixMounts("/Volumes")
	case "linux":
		return getUnixMounts("/")
	default:
		return []string{"/"}
	}
}

// Windows-specific logic using bitmasking
func getWindowsDrives() []string {
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

// Unix-specific logic
func getUnixMounts(root string) []string {
	var mounts []string

	if runtime.GOOS == "darwin" {
		// On Mac, external disks and DMG mounts are in /Volumes
		entries, _ := os.ReadDir(root)
		mounts = append(mounts, "/") // Root disk
		for _, e := range entries {
			mounts = append(mounts, root+"/"+e.Name())
		}
	} else {
		// On Linux, you'd typically parse /proc/mounts for accuracy,
		// but for a simple list, we start with the root.
		mounts = append(mounts, "/")
	}

	return mounts
}
