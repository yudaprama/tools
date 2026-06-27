# localfs Quick Start Guide

## Installation

```bash
go get github.com/kawai-network/veridium/pkg/localfs
```

## Basic Usage

```go
package main

import (
    "context"
    "fmt"
    "log"
    
    "github.com/kawai-network/veridium/pkg/localfs"
)

func main() {
    // Create service
    service := localfs.NewService()
    ctx := context.Background()
    
    // Your code here
}
```

## Common Operations

### Read a File

```go
result, err := service.ReadFile(ctx, localfs.ReadFileParams{
    Path: "/path/to/file.txt",
})
if err != nil {
    log.Fatal(err)
}
fmt.Println(result.Content)
```

### Write a File

```go
_, err := service.WriteFile(ctx, localfs.WriteFileParams{
    Path:    "/path/to/file.txt",
    Content: "Hello, World!",
})
```

### List Directory

```go
files, err := service.ListFiles(ctx, localfs.ListFileParams{
    Path: "/path/to/directory",
})
for _, file := range files {
    fmt.Printf("%s (%d bytes)\n", file.Name, file.Size)
}
```

### Edit File (Search & Replace)

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

### Run Shell Command

```go
result, err := service.RunCommand(ctx, localfs.RunCommandParams{
    Command: "ls -la",
})
fmt.Println(result.Output)
```

### Search Files by Name

```go
results, err := service.SearchFiles(ctx, localfs.SearchFilesParams{
    Keywords:  "test",
    Directory: "/path/to/search",
})
```

### Search Content (Grep)

```go
result, err := service.GrepContent(ctx, localfs.GrepContentParams{
    Pattern: "search term",
    Path:    "/path/to/search",
    CaseI:   true, // case insensitive
})
```

### Find Files by Pattern (Glob)

```go
result, err := service.GlobFiles(ctx, localfs.GlobFilesParams{
    Pattern: "*.txt",
    Path:    "/path/to/search",
})
```

## Advanced Usage

### Read Specific Lines

```go
loc := [2]int{10, 20} // lines 10-20
result, err := service.ReadFile(ctx, localfs.ReadFileParams{
    Path: "/path/to/file.txt",
    Loc:  &loc,
})
```

### Move Multiple Files

```go
results, err := service.MoveFiles(ctx, localfs.MoveFilesParams{
    Items: []localfs.MoveFileParams{
        {OldPath: "/old/file1.txt", NewPath: "/new/file1.txt"},
        {OldPath: "/old/file2.txt", NewPath: "/new/file2.txt"},
    },
})
```

### Background Command

```go
// Start command in background
result, err := service.RunCommand(ctx, localfs.RunCommandParams{
    Command:         "long-running-process",
    RunInBackground: true,
})
shellID := result.ShellID

// Get output later
output, err := service.GetCommandOutput(ctx, localfs.GetCommandOutputParams{
    ShellID: shellID,
})

// Kill if needed
_, err = service.KillCommand(ctx, localfs.KillCommandParams{
    ShellID: shellID,
})
```

## Error Handling

Always check errors:

```go
result, err := service.ReadFile(ctx, params)
if err != nil {
    log.Printf("Operation failed: %v", err)
    return
}

// Also check result status for some operations
if !result.Success {
    log.Printf("Operation unsuccessful: %s", result.Error)
    return
}
```

## Context Usage

Use context for cancellation and timeouts:

```go
// With timeout
ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
defer cancel()

result, err := service.RunCommand(ctx, localfs.RunCommandParams{
    Command: "slow-command",
})

// With cancellation
ctx, cancel := context.WithCancel(context.Background())
go func() {
    // Cancel after some condition
    cancel()
}()

result, err := service.ReadFile(ctx, params)
```

## Tips

1. **Always use context**: Pass `context.Background()` at minimum
2. **Check errors**: All operations return errors
3. **Close resources**: Context cancellation cleans up background commands
4. **Path separators**: Use `filepath.Join()` for cross-platform paths
5. **Large files**: Use line range (`Loc`) to read portions of large files

## More Examples

See:
- `service_test.go` - Comprehensive test examples
- `example_test.go` - Documented examples
- `examples/basic/main.go` - Complete working example
- `README.md` - Full documentation
- `MAPPING.md` - TypeScript to Go mapping

## Help

For more information:
- Read the [full README](README.md)
- Check the [API documentation](https://pkg.go.dev/github.com/kawai-network/veridium/pkg/localfs)
- See [examples](examples/basic/main.go)

