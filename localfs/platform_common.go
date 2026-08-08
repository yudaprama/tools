package localfs

import "time"

// FileTimeStat contains file time information
type FileTimeStat struct {
	CreatedTime    time.Time
	LastAccessTime time.Time
}
