package localfs

import (
	"context"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"sync"
	"time"
)

// Service provides local file system operations
type Service struct {
	// shellCommands stores running background commands
	shellCommands map[string]*exec.Cmd
	shellMu       sync.RWMutex
}

// NewService creates a new local file service
func NewService() *Service {
	return &Service{
		shellCommands: make(map[string]*exec.Cmd),
	}
}

// ============================================================================
// File Operations
// ============================================================================

// ListFiles lists files in a directory
func (s *Service) ListFiles(ctx context.Context, params ListLocalFileParams) ([]LocalFileItem, error) {
	entries, err := os.ReadDir(params.Path)
	if err != nil {
		return nil, fmt.Errorf("failed to read directory: %w", err)
	}

	items := make([]LocalFileItem, 0, len(entries))
	for _, entry := range entries {
		info, err := entry.Info()
		if err != nil {
			continue
		}

		fullPath := filepath.Join(params.Path, entry.Name())

		item := LocalFileItem{
			Name:         entry.Name(),
			Path:         fullPath,
			Size:         info.Size(),
			IsDirectory:  entry.IsDir(),
			ModifiedTime: info.ModTime(),
		}

		if entry.IsDir() {
			item.Type = "directory"
		} else {
			item.Type = "file"
			item.ContentType = getContentType(entry.Name())
		}

		// Get creation and access times (platform-specific)
		if stat := getFileTimes(info); stat != nil {
			item.CreatedTime = stat.CreatedTime
			item.LastAccessTime = stat.LastAccessTime
		}

		items = append(items, item)
	}

	return items, nil
}

// ReadFile reads a file with optional line range
func (s *Service) ReadFile(ctx context.Context, params LocalReadFileParams) (*LocalReadFileResult, error) {
	data, err := os.ReadFile(params.Path)
	if err != nil {
		return nil, fmt.Errorf("failed to read file: %w", err)
	}

	content := string(data)
	lines := strings.Split(content, "\n")
	totalLineCount := len(lines)
	totalCharCount := len(content)

	// Apply line range if specified
	var selectedContent string
	var startLine, endLine int

	if params.Loc != nil {
		startLine = params.Loc[0]
		endLine = params.Loc[1]

		if startLine < 1 {
			startLine = 1
		}
		if endLine > totalLineCount {
			endLine = totalLineCount
		}
		if startLine > endLine {
			startLine = endLine
		}

		selectedLines := lines[startLine-1 : endLine]
		selectedContent = strings.Join(selectedLines, "\n")
	} else {
		selectedContent = content
		startLine = 1
		endLine = totalLineCount
	}

	info, err := os.Stat(params.Path)
	if err != nil {
		return nil, fmt.Errorf("failed to stat file: %w", err)
	}

	result := &LocalReadFileResult{
		Content:        selectedContent,
		Filename:       filepath.Base(params.Path),
		FileType:       getContentType(params.Path),
		CharCount:      len(selectedContent),
		LineCount:      len(strings.Split(selectedContent, "\n")),
		TotalCharCount: totalCharCount,
		TotalLineCount: totalLineCount,
		Loc:            [2]int{startLine, endLine},
		ModifiedTime:   info.ModTime(),
	}

	if stat := getFileTimes(info); stat != nil {
		result.CreatedTime = stat.CreatedTime
	}

	return result, nil
}

// ReadFiles reads multiple files
func (s *Service) ReadFiles(ctx context.Context, params LocalReadFilesParams) ([]*LocalReadFileResult, error) {
	results := make([]*LocalReadFileResult, 0, len(params.Paths))

	for _, path := range params.Paths {
		result, err := s.ReadFile(ctx, LocalReadFileParams{Path: path})
		if err != nil {
			// Return error for individual file failure
			results = append(results, &LocalReadFileResult{
				Filename: filepath.Base(path),
			})
			continue
		}
		results = append(results, result)
	}

	return results, nil
}

// WriteFile writes content to a file
func (s *Service) WriteFile(ctx context.Context, params WriteLocalFileParams) (*WriteFileResult, error) {
	// Ensure directory exists
	dir := filepath.Dir(params.Path)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return &WriteFileResult{
			Path:    params.Path,
			Success: false,
			Error:   fmt.Sprintf("failed to create directory: %v", err),
		}, err
	}

	// Write file
	if err := os.WriteFile(params.Path, []byte(params.Content), 0644); err != nil {
		return &WriteFileResult{
			Path:    params.Path,
			Success: false,
			Error:   fmt.Sprintf("failed to write file: %v", err),
		}, err
	}

	return &WriteFileResult{
		Path:    params.Path,
		Success: true,
		Message: "File written successfully",
	}, nil
}

// EditFile performs search and replace in a file
func (s *Service) EditFile(ctx context.Context, params EditLocalFileParams) (*EditLocalFileResult, error) {
	data, err := os.ReadFile(params.FilePath)
	if err != nil {
		return &EditLocalFileResult{
			Success: false,
			Error:   fmt.Sprintf("failed to read file: %v", err),
		}, err
	}

	content := string(data)
	var newContent string
	var count int

	if params.ReplaceAll {
		newContent = strings.ReplaceAll(content, params.OldString, params.NewString)
		count = strings.Count(content, params.OldString)
	} else {
		newContent = strings.Replace(content, params.OldString, params.NewString, 1)
		if strings.Contains(content, params.OldString) {
			count = 1
		}
	}

	if count == 0 {
		return &EditLocalFileResult{
			Success:      false,
			Replacements: 0,
			Error:        "old_string not found in file",
		}, fmt.Errorf("old_string not found in file")
	}

	if err := os.WriteFile(params.FilePath, []byte(newContent), 0644); err != nil {
		return &EditLocalFileResult{
			Success: false,
			Error:   fmt.Sprintf("failed to write file: %v", err),
		}, err
	}

	return &EditLocalFileResult{
		Success:      true,
		Replacements: count,
	}, nil
}

// SearchFiles searches for files by keywords
func (s *Service) SearchFiles(ctx context.Context, params LocalSearchFilesParams) ([]LocalFileItem, error) {
	searchDir := params.Directory
	if searchDir == "" {
		searchDir = "."
	}

	var results []LocalFileItem
	keywords := strings.ToLower(params.Keywords)

	err := filepath.Walk(searchDir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return nil // Skip errors
		}

		// Check if filename contains keywords
		if strings.Contains(strings.ToLower(info.Name()), keywords) {
			item := LocalFileItem{
				Name:         info.Name(),
				Path:         path,
				Size:         info.Size(),
				IsDirectory:  info.IsDir(),
				ModifiedTime: info.ModTime(),
			}

			if info.IsDir() {
				item.Type = "directory"
			} else {
				item.Type = "file"
				item.ContentType = getContentType(info.Name())
			}

			if stat := getFileTimes(info); stat != nil {
				item.CreatedTime = stat.CreatedTime
				item.LastAccessTime = stat.LastAccessTime
			}

			results = append(results, item)
		}

		return nil
	})

	if err != nil {
		return nil, fmt.Errorf("failed to search files: %w", err)
	}

	return results, nil
}

// MoveFiles moves multiple files
func (s *Service) MoveFiles(ctx context.Context, params MoveLocalFilesParams) ([]LocalMoveFilesResultItem, error) {
	results := make([]LocalMoveFilesResultItem, 0, len(params.Items))

	for _, item := range params.Items {
		result := LocalMoveFilesResultItem{
			SourcePath: item.OldPath,
		}

		// Ensure destination directory exists
		destDir := filepath.Dir(item.NewPath)
		if err := os.MkdirAll(destDir, 0755); err != nil {
			result.Success = false
			result.Error = fmt.Sprintf("failed to create destination directory: %v", err)
			results = append(results, result)
			continue
		}

		// Move file
		if err := os.Rename(item.OldPath, item.NewPath); err != nil {
			result.Success = false
			result.Error = fmt.Sprintf("failed to move file: %v", err)
			results = append(results, result)
			continue
		}

		result.Success = true
		result.NewPath = item.NewPath
		results = append(results, result)
	}

	return results, nil
}

// RenameFile renames a file
func (s *Service) RenameFile(ctx context.Context, params RenameLocalFileParams) (*RenameLocalFileResult, error) {
	dir := filepath.Dir(params.Path)
	newPath := filepath.Join(dir, params.NewName)

	if err := os.Rename(params.Path, newPath); err != nil {
		return &RenameLocalFileResult{
			Success: false,
			Error:   fmt.Sprintf("failed to rename file: %v", err),
		}, err
	}

	return &RenameLocalFileResult{
		Success: true,
		NewPath: newPath,
	}, nil
}

// OpenFile opens a file with the default application
func (s *Service) OpenFile(ctx context.Context, params OpenLocalFileParams) error {
	return openFileWithDefaultApp(params.Path)
}

// OpenFolder opens a folder with the default file manager
func (s *Service) OpenFolder(ctx context.Context, params OpenLocalFolderParams) error {
	return openFolderWithDefaultApp(params.Path)
}

// ============================================================================
// Shell Commands
// ============================================================================

// RunCommand runs a shell command
func (s *Service) RunCommand(ctx context.Context, params RunCommandParams) (*RunCommandResult, error) {
	cmd := makeShellCmd(ctx, params.Command)

	if params.RunInBackground {
		// Start command in background
		shellID := generateShellID()
		s.shellMu.Lock()
		s.shellCommands[shellID] = cmd
		s.shellMu.Unlock()

		if err := cmd.Start(); err != nil {
			return &RunCommandResult{
				Success: false,
				Error:   fmt.Sprintf("failed to start command: %v", err),
			}, err
		}

		return &RunCommandResult{
			Success: true,
			ShellID: shellID,
		}, nil
	}

	// Run command synchronously
	output, err := cmd.CombinedOutput()
	exitCode := 0
	if err != nil {
		if exitErr, ok := err.(*exec.ExitError); ok {
			exitCode = exitErr.ExitCode()
		}
	}

	return &RunCommandResult{
		Success:  err == nil,
		Output:   string(output),
		Stdout:   string(output),
		ExitCode: exitCode,
		Error:    formatError(err),
	}, nil
}

// GetCommandOutput gets output from a running command
func (s *Service) GetCommandOutput(ctx context.Context, params GetCommandOutputParams) (*GetCommandOutputResult, error) {
	s.shellMu.RLock()
	cmd, exists := s.shellCommands[params.ShellID]
	s.shellMu.RUnlock()
	if !exists {
		return &GetCommandOutputResult{
			Success: false,
			Error:   "shell command not found",
		}, fmt.Errorf("shell command not found")
	}

	// Check if command is still running
	running := cmd.ProcessState == nil || !cmd.ProcessState.Exited()

	// Note: Getting real-time output requires capturing stdout/stderr during Start()
	// This is a simplified implementation
	return &GetCommandOutputResult{
		Success: true,
		Running: running,
		Output:  "",
		Stdout:  "",
		Stderr:  "",
	}, nil
}

// KillCommand kills a running command
func (s *Service) KillCommand(ctx context.Context, params KillCommandParams) (*KillCommandResult, error) {
	s.shellMu.RLock()
	cmd, exists := s.shellCommands[params.ShellID]
	s.shellMu.RUnlock()
	if !exists {
		return &KillCommandResult{
			Success: false,
			Error:   "shell command not found",
		}, fmt.Errorf("shell command not found")
	}

	if cmd.Process != nil {
		if err := cmd.Process.Kill(); err != nil {
			return &KillCommandResult{
				Success: false,
				Error:   fmt.Sprintf("failed to kill command: %v", err),
			}, err
		}
	}

	s.shellMu.Lock()
	delete(s.shellCommands, params.ShellID)
	s.shellMu.Unlock()

	return &KillCommandResult{
		Success: true,
	}, nil
}

// ============================================================================
// Search & Find
// ============================================================================

// GrepContent searches for content in files using grep-like functionality
func (s *Service) GrepContent(ctx context.Context, params GrepContentParams) (*GrepContentResult, error) {
	searchPath := params.Path
	if searchPath == "" {
		searchPath = "."
	}

	var matches []string
	totalMatches := 0

	err := filepath.Walk(searchPath, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return nil
		}

		if info.IsDir() {
			return nil
		}

		// Apply glob filter if specified
		if params.Glob != "" {
			matched, _ := filepath.Match(params.Glob, filepath.Base(path))
			if !matched {
				return nil
			}
		}

		// Read file content
		data, err := os.ReadFile(path)
		if err != nil {
			return nil
		}

		content := string(data)
		lines := strings.Split(content, "\n")

		// Search for pattern in each line
		for i, line := range lines {
			matched := false
			if params.CaseI {
				matched = strings.Contains(strings.ToLower(line), strings.ToLower(params.Pattern))
			} else {
				matched = strings.Contains(line, params.Pattern)
			}

			if matched {
				totalMatches++
				if params.OutputMode == "files_with_matches" {
					matches = append(matches, path)
					break
				} else if params.OutputMode == "count" {
					// Just count
				} else {
					// Default: content mode
					lineNum := i + 1
					if params.LineNum {
						matches = append(matches, fmt.Sprintf("%s:%d:%s", path, lineNum, line))
					} else {
						matches = append(matches, fmt.Sprintf("%s:%s", path, line))
					}
				}

				// Apply head limit
				if params.HeadLimit > 0 && len(matches) >= params.HeadLimit {
					return io.EOF
				}
			}
		}

		return nil
	})

	if err != nil && err != io.EOF {
		return &GrepContentResult{
			Success: false,
		}, fmt.Errorf("failed to grep content: %w", err)
	}

	return &GrepContentResult{
		Success:      true,
		Matches:      matches,
		TotalMatches: totalMatches,
	}, nil
}

// GlobFiles searches for files using glob patterns
func (s *Service) GlobFiles(ctx context.Context, params GlobFilesParams) (*GlobFilesResult, error) {
	searchPath := params.Path
	if searchPath == "" {
		searchPath = "."
	}

	pattern := filepath.Join(searchPath, params.Pattern)
	matches, err := filepath.Glob(pattern)
	if err != nil {
		return &GlobFilesResult{
			Success: false,
		}, fmt.Errorf("failed to glob files: %w", err)
	}

	return &GlobFilesResult{
		Success:    true,
		Files:      matches,
		TotalFiles: len(matches),
	}, nil
}

// ============================================================================
// Helper Methods
// ============================================================================

// OpenFileOrFolder opens a file or folder based on the type
func (s *Service) OpenFileOrFolder(ctx context.Context, path string, isDirectory bool) error {
	if isDirectory {
		return s.OpenFolder(ctx, OpenLocalFolderParams{Path: path, IsDirectory: true})
	}
	return s.OpenFile(ctx, OpenLocalFileParams{Path: path})
}

// ============================================================================
// Private Helper Functions
// ============================================================================

func getContentType(filename string) string {
	ext := strings.ToLower(filepath.Ext(filename))
	switch ext {
	case ".txt":
		return "text/plain"
	case ".md":
		return "text/markdown"
	case ".json":
		return "application/json"
	case ".xml":
		return "application/xml"
	case ".html", ".htm":
		return "text/html"
	case ".css":
		return "text/css"
	case ".js":
		return "application/javascript"
	case ".go":
		return "text/x-go"
	case ".py":
		return "text/x-python"
	case ".java":
		return "text/x-java"
	case ".c", ".h":
		return "text/x-c"
	case ".cpp", ".hpp":
		return "text/x-c++"
	case ".rs":
		return "text/x-rust"
	case ".ts":
		return "application/typescript"
	case ".tsx":
		return "text/typescript-jsx"
	case ".jsx":
		return "text/jsx"
	case ".pdf":
		return "application/pdf"
	case ".png":
		return "image/png"
	case ".jpg", ".jpeg":
		return "image/jpeg"
	case ".gif":
		return "image/gif"
	case ".svg":
		return "image/svg+xml"
	case ".zip":
		return "application/zip"
	case ".tar":
		return "application/x-tar"
	case ".gz":
		return "application/gzip"
	default:
		return "application/octet-stream"
	}
}

func formatError(err error) string {
	if err == nil {
		return ""
	}
	return err.Error()
}

func generateShellID() string {
	return fmt.Sprintf("shell_%d", time.Now().UnixNano())
}

// makeShellCmd builds a cross-platform shell command
func makeShellCmd(ctx context.Context, command string) *exec.Cmd {
	// Use OS-appropriate shell for portability
	switch runtime.GOOS {
	case "windows":
		return exec.CommandContext(ctx, "cmd", "/C", command)
	default:
		return exec.CommandContext(ctx, "sh", "-c", command)
	}
}
