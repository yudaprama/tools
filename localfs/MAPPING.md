# TypeScript to Go Service Mapping

This document shows how the frontend TypeScript `localFileService` maps to the Go `localfs.Service`.

## Service Creation

**TypeScript:**
```typescript
import { localFileService } from '@/services/electron/localFileService';
```

**Go:**
```go
import "github.com/kawai-network/veridium/pkg/localfs"

service := localfs.NewService()
ctx := context.Background()
```

## File Operations

### List Files

**TypeScript:**
```typescript
const files = await localFileService.listLocalFiles({ path: '/path/to/dir' });
```

**Go:**
```go
files, err := service.ListFiles(ctx, localfs.ListFileParams{
    Path: "/path/to/dir",
})
```

### Read File

**TypeScript:**
```typescript
const result = await localFileService.readLocalFile({ path: '/path/to/file.txt' });
```

**Go:**
```go
result, err := service.ReadFile(ctx, localfs.ReadFileParams{
    Path: "/path/to/file.txt",
})
```

### Read File with Line Range

**TypeScript:**
```typescript
const result = await localFileService.readLocalFile({ 
    path: '/path/to/file.txt',
    loc: [10, 20]
});
```

**Go:**
```go
loc := [2]int{10, 20}
result, err := service.ReadFile(ctx, localfs.ReadFileParams{
    Path: "/path/to/file.txt",
    Loc:  &loc,
})
```

### Read Multiple Files

**TypeScript:**
```typescript
const results = await localFileService.readLocalFiles({ 
    paths: ['/file1.txt', '/file2.txt']
});
```

**Go:**
```go
results, err := service.ReadFiles(ctx, localfs.ReadFilesParams{
    Paths: []string{"/file1.txt", "/file2.txt"},
})
```

### Write File

**TypeScript:**
```typescript
await localFileService.writeFile({ 
    path: '/path/to/file.txt',
    content: 'Hello, World!'
});
```

**Go:**
```go
result, err := service.WriteFile(ctx, localfs.WriteFileParams{
    Path:    "/path/to/file.txt",
    Content: "Hello, World!",
})
```

### Edit File

**TypeScript:**
```typescript
const result = await localFileService.editLocalFile({ 
    file_path: '/path/to/file.txt',
    old_string: 'old',
    new_string: 'new',
    replace_all: true
});
```

**Go:**
```go
result, err := service.EditFile(ctx, localfs.EditFileParams{
    FilePath:   "/path/to/file.txt",
    OldString:  "old",
    NewString:  "new",
    ReplaceAll: true,
})
```

### Search Files

**TypeScript:**
```typescript
const results = await localFileService.searchLocalFiles({ 
    keywords: 'test',
    directory: '/path/to/search'
});
```

**Go:**
```go
results, err := service.SearchFiles(ctx, localfs.SearchFilesParams{
    Keywords:  "test",
    Directory: "/path/to/search",
})
```

### Move Files

**TypeScript:**
```typescript
const results = await localFileService.moveLocalFiles({ 
    items: [
        { oldPath: '/old/file1.txt', newPath: '/new/file1.txt' },
        { oldPath: '/old/file2.txt', newPath: '/new/file2.txt' }
    ]
});
```

**Go:**
```go
results, err := service.MoveFiles(ctx, localfs.MoveFilesParams{
    Items: []localfs.MoveFileParams{
        {OldPath: "/old/file1.txt", NewPath: "/new/file1.txt"},
        {OldPath: "/old/file2.txt", NewPath: "/new/file2.txt"},
    },
})
```

### Rename File

**TypeScript:**
```typescript
await localFileService.renameLocalFile({ 
    path: '/path/to/file.txt',
    newName: 'renamed.txt'
});
```

**Go:**
```go
result, err := service.RenameFile(ctx, localfs.RenameFileParams{
    Path:    "/path/to/file.txt",
    NewName: "renamed.txt",
})
```

### Open File

**TypeScript:**
```typescript
await localFileService.openLocalFile({ path: '/path/to/file.txt' });
```

**Go:**
```go
err := service.OpenFile(ctx, localfs.OpenFileParams{
    Path: "/path/to/file.txt",
})
```

### Open Folder

**TypeScript:**
```typescript
await localFileService.openLocalFolder({ 
    path: '/path/to/folder',
    isDirectory: true
});
```

**Go:**
```go
err := service.OpenFolder(ctx, localfs.OpenFolderParams{
    Path:        "/path/to/folder",
    IsDirectory: true,
})
```

### Open File or Folder (Helper)

**TypeScript:**
```typescript
await localFileService.openLocalFileOrFolder('/path/to/item', true);
```

**Go:**
```go
err := service.OpenFileOrFolder(ctx, "/path/to/item", true)
```

## Shell Commands

### Run Command

**TypeScript:**
```typescript
const result = await localFileService.runCommand({ 
    command: 'ls -la',
    run_in_background: false,
    timeout: 30
});
```

**Go:**
```go
result, err := service.RunCommand(ctx, localfs.RunCommandParams{
    Command:         "ls -la",
    RunInBackground: false,
    Timeout:         30,
})
```

### Run Command in Background

**TypeScript:**
```typescript
const result = await localFileService.runCommand({ 
    command: 'long-running-process',
    run_in_background: true
});
// Use result.shell_id to manage the process
```

**Go:**
```go
result, err := service.RunCommand(ctx, localfs.RunCommandParams{
    Command:         "long-running-process",
    RunInBackground: true,
})
// Use result.ShellID to manage the process
```

### Get Command Output

**TypeScript:**
```typescript
const result = await localFileService.getCommandOutput({ 
    shell_id: 'shell_12345'
});
```

**Go:**
```go
result, err := service.GetCommandOutput(ctx, localfs.GetCommandOutputParams{
    ShellID: "shell_12345",
})
```

### Kill Command

**TypeScript:**
```typescript
const result = await localFileService.killCommand({ 
    shell_id: 'shell_12345'
});
```

**Go:**
```go
result, err := service.KillCommand(ctx, localfs.KillCommandParams{
    ShellID: "shell_12345",
})
```

## Search & Find

### Grep Content

**TypeScript:**
```typescript
const result = await localFileService.grepContent({ 
    pattern: 'search term',
    path: '/path/to/search',
    '-i': true,
    '-n': true,
    'glob': '*.go',
    'output_mode': 'content'
});
```

**Go:**
```go
result, err := service.GrepContent(ctx, localfs.GrepContentParams{
    Pattern:    "search term",
    Path:       "/path/to/search",
    CaseI:      true,
    LineNum:    true,
    Glob:       "*.go",
    OutputMode: "content",
})
```

### Glob Files

**TypeScript:**
```typescript
const result = await localFileService.globFiles({ 
    pattern: '*.txt',
    path: '/path/to/search'
});
```

**Go:**
```go
result, err := service.GlobFiles(ctx, localfs.GlobFilesParams{
    Pattern: "*.txt",
    Path:    "/path/to/search",
})
```

## Type Mappings

### FileItem

**TypeScript:**
```typescript
interface LocalFileItem {
    name: string;
    path: string;
    size: number;
    type: string;
    isDirectory: boolean;
    contentType?: string;
    createdTime: Date;
    modifiedTime: Date;
    lastAccessTime: Date;
    metadata?: { [key: string]: any };
}
```

**Go:**
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

**TypeScript:**
```typescript
interface LocalReadFileResult {
    content: string;
    filename: string;
    fileType: string;
    charCount: number;
    lineCount: number;
    totalCharCount: number;
    totalLineCount: number;
    loc: [number, number];
    createdTime: Date;
    modifiedTime: Date;
}
```

**Go:**
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

**TypeScript:**
```typescript
interface RunCommandResult {
    success: boolean;
    output?: string;
    stdout?: string;
    stderr?: string;
    exit_code?: number;
    shell_id?: string;
    error?: string;
}
```

**Go:**
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

## Key Differences

1. **Context**: Go version requires `context.Context` for all operations (TypeScript uses promises)
2. **Error Handling**: Go uses explicit error returns, TypeScript uses try/catch
3. **Naming**: Go uses PascalCase for exported types, TypeScript uses camelCase
4. **Optional Fields**: Go uses pointers for optional fields (e.g., `*[2]int` for `Loc`)
5. **JSON Tags**: Go types include JSON tags for serialization compatibility

## Usage Pattern Comparison

**TypeScript (Frontend):**
```typescript
try {
    const result = await localFileService.readLocalFile({ path: '/file.txt' });
    console.log(result.content);
} catch (error) {
    console.error('Failed to read file:', error);
}
```

**Go (Backend):**
```go
result, err := service.ReadFile(ctx, localfs.ReadFileParams{
    Path: "/file.txt",
})
if err != nil {
    log.Printf("Failed to read file: %v", err)
    return
}
fmt.Println(result.Content)
```

## Integration Example

The Go service can be easily integrated into a backend API that mirrors the frontend service:

```go
// Backend API handler
func handleReadFile(w http.ResponseWriter, r *http.Request) {
    var params localfs.ReadFileParams
    json.NewDecoder(r.Body).Decode(&params)
    
    service := localfs.NewService()
    result, err := service.ReadFile(r.Context(), params)
    if err != nil {
        http.Error(w, err.Error(), http.StatusInternalServerError)
        return
    }
    
    json.NewEncoder(w).Encode(result)
}
```

This creates a seamless bridge between the TypeScript frontend and Go backend.

