//go:build windows

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
	stat, ok := info.Sys().(*syscall.Win32FileAttributeData)
	if !ok {
		return &FileTimeStat{
			CreatedTime:    info.ModTime(),
			LastAccessTime: info.ModTime(),
		}
	}

	return &FileTimeStat{
		CreatedTime:    time.Unix(0, stat.CreationTime.Nanoseconds()),
		LastAccessTime: time.Unix(0, stat.LastAccessTime.Nanoseconds()),
	}
}

// openFileWithDefaultApp opens a file with the default application on Windows
func openFileWithDefaultApp(path string) error {
	cmd := exec.Command("cmd", "/c", "start", "", path)
	return cmd.Run()
}

// openFolderWithDefaultApp opens a folder with Explorer on Windows
func openFolderWithDefaultApp(path string) error {
	cmd := exec.Command("explorer", path)
	return cmd.Run()
}
