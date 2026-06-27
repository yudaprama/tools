# localfs - Local File System Service

A comprehensive Go package for local file system operations, providing a service-oriented interface for file management, shell commands, and content search.

## Features

- **File Operations**: Read, write, edit, list, search, move, and rename files
- **Shell Commands**: Execute commands synchronously or in background
- **Content Search**: Grep-like content search and glob pattern matching
- **Cross-Platform**: Supports macOS, Linux, and Windows
- **Type-Safe**: Strongly typed parameters and results
- **Context-Aware**: All operations support context for cancellation and timeouts

## Installation

```bash
go get github.com/kawai-network/veridium/pkg/localfs
```

## Quick Start

```go
package main

import (
    "context"
    "fmt"
    "log"
    
    "github.com/kawai-network/veridium/pkg/localfs"
)

func main() {
    // Create service instance
    service := localfs.NewService()
    ctx := context.Background()
    
    // Write a file
    _, err := service.WriteFile(ctx, localfs.WriteFileParams{
        Path:    "/tmp/hello.txt",
        Content: "Hello, World!",
    })
    if err != nil {
        log.Fatal(err)
    }
    
    // Read the file
    result, err := service.ReadFile(ctx, localfs.ReadFileParams{
        Path: "/tmp/hello.txt",
    })
    if err != nil {
        log.Fatal(err)
    }
    
    fmt.Printf("Content: %s\n", result.Content)
    fmt.Printf("Lines: %d\n", result.LineCount)
}
```

## API Documentation

### File Operations

#### ListFiles

List all files and directories in a path.

```go
files, err := service.ListFiles(ctx, localfs.ListFileParams{
    Path: "/path/to/directory",
})
```

#### ReadFile

Read a file with optional line range.

```go
// Read entire file
result, err := service.ReadFile(ctx, localfs.ReadFileParams{
    Path: "/path/to/file.txt",
})

// Read specific line range (lines 10-20)
loc := [2]int{10, 20}
result, err := service.ReadFile(ctx, localfs.ReadFileParams{
    Path: "/path/to/file.txt",
    Loc:  &loc,
})
```

#### ReadFiles

Read multiple files at once.

```go
results, err := service.ReadFiles(ctx, localfs.ReadFilesParams{
    Paths: []string{
        "/path/to/file1.txt",
        "/path/to/file2.txt",
    },
})
```

#### WriteFile

Write content to a file (creates directories if needed).

```go
result, err := service.WriteFile(ctx, localfs.WriteFileParams{
    Path:    "/path/to/file.txt",
    Content: "File content here",
})
```

#### EditFile

Search and replace content in a file.

```go
// Replace first occurrence
result, err := service.EditFile(ctx, localfs.EditFileParams{
    FilePath:   "/path/to/file.txt",
    OldString:  "old text",
    NewString:  "new text",
    ReplaceAll: false,
})

// Replace all occurrences
result, err := service.EditFile(ctx, localfs.EditFileParams{
    FilePath:   "/path/to/file.txt",
    OldString:  "old text",
    NewString:  "new text",
    ReplaceAll: true,
})
```

#### SearchFiles

Search for files by name keywords.

```go
results, err := service.SearchFiles(ctx, localfs.SearchFilesParams{
    Keywords:  "test",
    Directory: "/path/to/search",
})
```

#### MoveFiles

Move multiple files to new locations.

```go
results, err := service.MoveFiles(ctx, localfs.MoveFilesParams{
    Items: []localfs.MoveFileParams{
        {OldPath: "/old/path/file1.txt", NewPath: "/new/path/file1.txt"},
        {OldPath: "/old/path/file2.txt", NewPath: "/new/path/file2.txt"},
    },
})
```

#### RenameFile

Rename a file.

```go
result, err := service.RenameFile(ctx, localfs.RenameFileParams{
    Path:    "/path/to/oldname.txt",
    NewName: "newname.txt",
})
```

#### OpenFile / OpenFolder

Open a file or folder with the default system application.

```go
// Open file
err := service.OpenFile(ctx, localfs.OpenFileParams{
    Path: "/path/to/file.txt",
})

// Open folder
err := service.OpenFolder(ctx, localfs.OpenFolderParams{
    Path: "/path/to/folder",
})

// Open either (helper method)
err := service.OpenFileOrFolder(ctx, "/path/to/item", isDirectory)
```

### Shell Commands

#### RunCommand

Execute a shell command.

```go
// Run synchronously
result, err := service.RunCommand(ctx, localfs.RunCommandParams{
    Command: "ls -la",
})

// Run in background
result, err := service.RunCommand(ctx, localfs.RunCommandParams{
    Command:         "long-running-process",
    RunInBackground: true,
    Description:     "Processing data",
})
// Use result.ShellID to manage the background process
```

#### GetCommandOutput

Get output from a running background command.

```go
result, err := service.GetCommandOutput(ctx, localfs.GetCommandOutputParams{
    ShellID: "shell_12345",
})
```

#### KillCommand

Kill a running background command.

```go
result, err := service.KillCommand(ctx, localfs.KillCommandParams{
    ShellID: "shell_12345",
})
```

### Search & Find

#### GrepContent

Search for content in files (grep-like functionality).

```go
// Basic search
result, err := service.GrepContent(ctx, localfs.GrepContentParams{
    Pattern: "search term",
    Path:    "/path/to/search",
})

// Advanced search with options
result, err := service.GrepContent(ctx, localfs.GrepContentParams{
    Pattern:    "search term",
    Path:       "/path/to/search",
    CaseI:      true,  // Case insensitive
    LineNum:    true,  // Show line numbers
    Glob:       "*.go", // Only search .go files
    OutputMode: "files_with_matches", // Only show filenames
    HeadLimit:  100,   // Limit to first 100 matches
})
```

#### GlobFiles

Search for files using glob patterns.

```go
result, err := service.GlobFiles(ctx, localfs.GlobFilesParams{
    Pattern: "*.txt",
    Path:    "/path/to/search",
})
```

## Types

### FileItem

```go
type FileItem struct {
    Name           string
    Path           string
    Size           int64
    Type           string
    IsDirectory    bool
    ContentType    string
    CreatedTime    time.Time
    ModifiedTime   time.Time
    LastAccessTime time.Time
    Metadata       map[string]interface{}
}
```

### ReadFileResult

```go
type ReadFileResult struct {
    Content        string
    Filename       string
    FileType       string
    CharCount      int
    LineCount      int
    TotalCharCount int
    TotalLineCount int
    Loc            [2]int
    CreatedTime    time.Time
    ModifiedTime   time.Time
}
```

### RunCommandResult

```go
type RunCommandResult struct {
    Success  bool
    Output   string
    Stdout   string
    Stderr   string
    ExitCode int
    ShellID  string
    Error    string
}
```

## Platform Support

The package supports the following platforms:

- **macOS**: Uses `open` command for file/folder opening
- **Linux**: Uses `xdg-open` command for file/folder opening
- **Windows**: Uses `cmd /c start` and `explorer` for file/folder opening

Platform-specific file time information (creation time, access time) is automatically handled.

## Error Handling

All operations return errors following Go conventions. Check errors before using results:

```go
result, err := service.ReadFile(ctx, params)
if err != nil {
    log.Printf("Failed to read file: %v", err)
    return
}
// Use result
```

Many result types also include a `Success` field and optional `Error` field for operation status:

```go
result, err := service.WriteFile(ctx, params)
if err != nil || !result.Success {
    log.Printf("Write failed: %s", result.Error)
    return
}
```

## Testing

Run tests:

```bash
go test ./pkg/localfs
```

Run tests with coverage:

```bash
go test -cover ./pkg/localfs
```

## Examples

See `service_test.go` for comprehensive examples of all operations.

## License

Part of the Veridium project.

