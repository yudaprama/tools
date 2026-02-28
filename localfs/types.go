package localfs

import "time"

// ============================================================================
// File Types
// ============================================================================

// LocalFileItem represents a file or directory item
type LocalFileItem struct {
	Name           string                 `json:"name"`
	Path           string                 `json:"path"`
	Size           int64                  `json:"size"`
	Type           string                 `json:"type"`
	IsDirectory    bool                   `json:"isDirectory"`
	ContentType    string                 `json:"contentType,omitempty"`
	CreatedTime    time.Time              `json:"createdTime"`
	ModifiedTime   time.Time              `json:"modifiedTime"`
	LastAccessTime time.Time              `json:"lastAccessTime"`
	Metadata       map[string]interface{} `json:"metadata,omitempty"`
}

// ListLocalFileParams parameters for listing files
type ListLocalFileParams struct {
	Path string `json:"path"`
}

// LocalReadFileParams parameters for reading a file
type LocalReadFileParams struct {
	Path string  `json:"path"`
	Loc  *[2]int `json:"loc,omitempty"` // Optional line range [start, end]
}

// LocalReadFileResult result of reading a file
type LocalReadFileResult struct {
	Content        string    `json:"content"`
	Filename       string    `json:"filename"`
	FileType       string    `json:"fileType"`
	CharCount      int       `json:"charCount"`
	LineCount      int       `json:"lineCount"`
	TotalCharCount int       `json:"totalCharCount"`
	TotalLineCount int       `json:"totalLineCount"`
	Loc            [2]int    `json:"loc"`
	CreatedTime    time.Time `json:"createdTime"`
	ModifiedTime   time.Time `json:"modifiedTime"`
}

// LocalReadFilesParams parameters for reading multiple files
type LocalReadFilesParams struct {
	Paths []string `json:"paths"`
}

// WriteLocalFileParams parameters for writing a file
type WriteLocalFileParams struct {
	Path    string `json:"path"`
	Content string `json:"content"`
}

// WriteFileResult result of writing a file
type WriteFileResult struct {
	Path    string `json:"path"`
	Success bool   `json:"success"`
	Message string `json:"message"`
	Error   string `json:"error,omitempty"`
}

// EditLocalFileParams parameters for editing a file
type EditLocalFileParams struct {
	FilePath   string `json:"file_path"`
	OldString  string `json:"old_string"`
	NewString  string `json:"new_string"`
	ReplaceAll bool   `json:"replace_all,omitempty"`
}

// EditLocalFileResult result of editing a file
type EditLocalFileResult struct {
	Success      bool   `json:"success"`
	Replacements int    `json:"replacements"`
	Error        string `json:"error,omitempty"`
}

// LocalSearchFilesParams parameters for searching files
type LocalSearchFilesParams struct {
	Keywords  string `json:"keywords"`
	Directory string `json:"directory,omitempty"`
}

// MoveLocalFileParams parameters for moving a single file
type MoveLocalFileParams struct {
	OldPath string `json:"oldPath"`
	NewPath string `json:"newPath"`
}

// MoveLocalFilesParams parameters for moving multiple files
type MoveLocalFilesParams struct {
	Items []MoveLocalFileParams `json:"items"`
}

// LocalMoveFilesResultItem result of moving a single file
type LocalMoveFilesResultItem struct {
	SourcePath string `json:"sourcePath"`
	NewPath    string `json:"newPath,omitempty"`
	Success    bool   `json:"success"`
	Error      string `json:"error,omitempty"`
}

// RenameLocalFileParams parameters for renaming a file
type RenameLocalFileParams struct {
	Path    string `json:"path"`
	NewName string `json:"newName"`
}

// RenameLocalFileResult result of renaming a file
type RenameLocalFileResult struct {
	NewPath string `json:"newPath"`
	Success bool   `json:"success"`
	Error   string `json:"error,omitempty"`
}

// OpenLocalFileParams parameters for opening a file
type OpenLocalFileParams struct {
	Path string `json:"path"`
}

// OpenLocalFolderParams parameters for opening a folder
type OpenLocalFolderParams struct {
	Path        string `json:"path"`
	IsDirectory bool   `json:"isDirectory,omitempty"`
}

// ============================================================================
// Shell Command Types
// ============================================================================

// RunCommandParams parameters for running a command
type RunCommandParams struct {
	Command         string `json:"command"`
	Description     string `json:"description,omitempty"`
	RunInBackground bool   `json:"run_in_background,omitempty"`
	Timeout         int    `json:"timeout,omitempty"` // Timeout in seconds
}

// RunCommandResult result of running a command
type RunCommandResult struct {
	Success  bool   `json:"success"`
	Output   string `json:"output,omitempty"`
	Stdout   string `json:"stdout,omitempty"`
	Stderr   string `json:"stderr,omitempty"`
	ExitCode int    `json:"exit_code,omitempty"`
	ShellID  string `json:"shell_id,omitempty"`
	Error    string `json:"error,omitempty"`
}

// GetCommandOutputParams parameters for getting command output
type GetCommandOutputParams struct {
	ShellID string `json:"shell_id"`
	Filter  string `json:"filter,omitempty"`
}

// GetCommandOutputResult result of getting command output
type GetCommandOutputResult struct {
	Success bool   `json:"success"`
	Output  string `json:"output"`
	Stdout  string `json:"stdout"`
	Stderr  string `json:"stderr"`
	Running bool   `json:"running"`
	Error   string `json:"error,omitempty"`
}

// KillCommandParams parameters for killing a command
type KillCommandParams struct {
	ShellID string `json:"shell_id"`
}

// KillCommandResult result of killing a command
type KillCommandResult struct {
	Success bool   `json:"success"`
	Error   string `json:"error,omitempty"`
}

// ============================================================================
// Search & Find Types
// ============================================================================

// GrepContentParams parameters for grep content search
type GrepContentParams struct {
	Pattern    string `json:"pattern"`
	Path       string `json:"path,omitempty"`
	Type       string `json:"type,omitempty"`
	Glob       string `json:"glob,omitempty"`
	OutputMode string `json:"output_mode,omitempty"` // "content", "files_with_matches", "count"
	Multiline  bool   `json:"multiline,omitempty"`
	CaseI      bool   `json:"-i,omitempty"`
	LineNum    bool   `json:"-n,omitempty"`
	ContextA   int    `json:"-A,omitempty"` // Lines after
	ContextB   int    `json:"-B,omitempty"` // Lines before
	ContextC   int    `json:"-C,omitempty"` // Lines around
	HeadLimit  int    `json:"head_limit,omitempty"`
}

// GrepContentResult result of grep content search
type GrepContentResult struct {
	Success      bool     `json:"success"`
	Matches      []string `json:"matches"`
	TotalMatches int      `json:"total_matches"`
}

// GlobFilesParams parameters for glob file search
type GlobFilesParams struct {
	Pattern string `json:"pattern"`
	Path    string `json:"path,omitempty"`
}

// GlobFilesResult result of glob file search
type GlobFilesResult struct {
	Success    bool     `json:"success"`
	Files      []string `json:"files"`
	TotalFiles int      `json:"total_files"`
}
