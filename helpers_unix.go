//go:build darwin || linux

package main

import (
	"os"
	"os/exec"
	"runtime"
)

func GetDisks() []string {
	switch runtime.GOOS {
	case "darwin": // macOS
		return getUnixMounts("/Volumes")
	case "linux":
		return getUnixMounts("/")
	default:
		return []string{"/"}
	}
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
