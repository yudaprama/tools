//go:build linux

package localfs

import (
	"os"
	"os/exec"
	"syscall"
	"time"
)

// FileTimeStat contains file time information
type FileTimeStat struct {
	CreatedTime    time.Time
	LastAccessTime time.Time
}

// getFileTimes extracts file times from os.FileInfo
func getFileTimes(info os.FileInfo) *FileTimeStat {
	stat, ok := info.Sys().(*syscall.Stat_t)
	if !ok {
		return &FileTimeStat{
			CreatedTime:    info.ModTime(),
			LastAccessTime: info.ModTime(),
		}
	}

	return &FileTimeStat{
		// Linux doesn't have birth time in standard stat
		CreatedTime:    time.Unix(stat.Ctim.Sec, stat.Ctim.Nsec),
		LastAccessTime: time.Unix(stat.Atim.Sec, stat.Atim.Nsec),
	}
}

// openFileWithDefaultApp opens a file with the default application on Linux
func openFileWithDefaultApp(path string) error {
	cmd := exec.Command("xdg-open", path)
	return cmd.Run()
}

// openFolderWithDefaultApp opens a folder with the default file manager on Linux
func openFolderWithDefaultApp(path string) error {
	cmd := exec.Command("xdg-open", path)
	return cmd.Run()
}
