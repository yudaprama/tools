//go:build darwin

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
		CreatedTime:    time.Unix(stat.Birthtimespec.Sec, stat.Birthtimespec.Nsec),
		LastAccessTime: time.Unix(stat.Atimespec.Sec, stat.Atimespec.Nsec),
	}
}

// openFileWithDefaultApp opens a file with the default application on macOS
func openFileWithDefaultApp(path string) error {
	cmd := exec.Command("open", path)
	return cmd.Run()
}

// openFolderWithDefaultApp opens a folder with Finder on macOS
func openFolderWithDefaultApp(path string) error {
	cmd := exec.Command("open", path)
	return cmd.Run()
}
