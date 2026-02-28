package localfs

import (
	"context"
	"os"
	"path/filepath"
	"testing"
)

func TestService_WriteAndReadFile(t *testing.T) {
	service := NewService()
	ctx := context.Background()

	// Create temp directory
	tmpDir := t.TempDir()
	testFile := filepath.Join(tmpDir, "test.txt")
	testContent := "Hello, World!\nThis is a test file."

	// Test WriteFile
	writeResult, err := service.WriteFile(ctx, WriteLocalFileParams{
		Path:    testFile,
		Content: testContent,
	})
	if err != nil {
		t.Fatalf("WriteFile failed: %v", err)
	}
	if !writeResult.Success {
		t.Fatalf("WriteFile returned success=false: %s", writeResult.Error)
	}

	// Test ReadFile
	readResult, err := service.ReadFile(ctx, LocalReadFileParams{
		Path: testFile,
	})
	if err != nil {
		t.Fatalf("ReadFile failed: %v", err)
	}
	if readResult.Content != testContent {
		t.Errorf("Expected content %q, got %q", testContent, readResult.Content)
	}
	if readResult.LineCount != 2 {
		t.Errorf("Expected 2 lines, got %d", readResult.LineCount)
	}
}

func TestService_ReadFileWithLineRange(t *testing.T) {
	service := NewService()
	ctx := context.Background()

	tmpDir := t.TempDir()
	testFile := filepath.Join(tmpDir, "test.txt")
	testContent := "Line 1\nLine 2\nLine 3\nLine 4\nLine 5"

	// Write test file
	_, err := service.WriteFile(ctx, WriteLocalFileParams{
		Path:    testFile,
		Content: testContent,
	})
	if err != nil {
		t.Fatalf("WriteFile failed: %v", err)
	}

	// Read lines 2-4
	loc := [2]int{2, 4}
	readResult, err := service.ReadFile(ctx, LocalReadFileParams{
		Path: testFile,
		Loc:  &loc,
	})
	if err != nil {
		t.Fatalf("ReadFile failed: %v", err)
	}

	expectedContent := "Line 2\nLine 3\nLine 4"
	if readResult.Content != expectedContent {
		t.Errorf("Expected content %q, got %q", expectedContent, readResult.Content)
	}
	if readResult.LineCount != 3 {
		t.Errorf("Expected 3 lines, got %d", readResult.LineCount)
	}
	if readResult.TotalLineCount != 5 {
		t.Errorf("Expected total 5 lines, got %d", readResult.TotalLineCount)
	}
}

func TestService_ListFiles(t *testing.T) {
	service := NewService()
	ctx := context.Background()

	tmpDir := t.TempDir()

	// Create test files
	files := []string{"file1.txt", "file2.txt", "file3.txt"}
	for _, f := range files {
		path := filepath.Join(tmpDir, f)
		if err := os.WriteFile(path, []byte("test"), 0644); err != nil {
			t.Fatalf("Failed to create test file: %v", err)
		}
	}

	// Create test directory
	subDir := filepath.Join(tmpDir, "subdir")
	if err := os.Mkdir(subDir, 0755); err != nil {
		t.Fatalf("Failed to create test directory: %v", err)
	}

	// List files
	items, err := service.ListFiles(ctx, ListLocalFileParams{Path: tmpDir})
	if err != nil {
		t.Fatalf("ListFiles failed: %v", err)
	}

	if len(items) != 4 { // 3 files + 1 directory
		t.Errorf("Expected 4 items, got %d", len(items))
	}

	// Check directory item
	foundDir := false
	for _, item := range items {
		if item.Name == "subdir" && item.IsDirectory {
			foundDir = true
			break
		}
	}
	if !foundDir {
		t.Error("Expected to find subdirectory")
	}
}

func TestService_EditFile(t *testing.T) {
	service := NewService()
	ctx := context.Background()

	tmpDir := t.TempDir()
	testFile := filepath.Join(tmpDir, "test.txt")
	originalContent := "Hello World\nHello World\nGoodbye World"

	// Write original file
	_, err := service.WriteFile(ctx, WriteLocalFileParams{
		Path:    testFile,
		Content: originalContent,
	})
	if err != nil {
		t.Fatalf("WriteFile failed: %v", err)
	}

	// Edit file - replace first occurrence
	editResult, err := service.EditFile(ctx, EditLocalFileParams{
		FilePath:   testFile,
		OldString:  "Hello",
		NewString:  "Hi",
		ReplaceAll: false,
	})
	if err != nil {
		t.Fatalf("EditFile failed: %v", err)
	}
	if !editResult.Success {
		t.Fatalf("EditFile returned success=false: %s", editResult.Error)
	}
	if editResult.Replacements != 1 {
		t.Errorf("Expected 1 replacement, got %d", editResult.Replacements)
	}

	// Read and verify
	readResult, err := service.ReadFile(ctx, LocalReadFileParams{Path: testFile})
	if err != nil {
		t.Fatalf("ReadFile failed: %v", err)
	}
	expectedContent := "Hi World\nHello World\nGoodbye World"
	if readResult.Content != expectedContent {
		t.Errorf("Expected content %q, got %q", expectedContent, readResult.Content)
	}
}

func TestService_EditFileReplaceAll(t *testing.T) {
	service := NewService()
	ctx := context.Background()

	tmpDir := t.TempDir()
	testFile := filepath.Join(tmpDir, "test.txt")
	originalContent := "Hello World\nHello World\nHello Universe"

	// Write original file
	_, err := service.WriteFile(ctx, WriteLocalFileParams{
		Path:    testFile,
		Content: originalContent,
	})
	if err != nil {
		t.Fatalf("WriteFile failed: %v", err)
	}

	// Edit file - replace all occurrences
	editResult, err := service.EditFile(ctx, EditLocalFileParams{
		FilePath:   testFile,
		OldString:  "Hello",
		NewString:  "Hi",
		ReplaceAll: true,
	})
	if err != nil {
		t.Fatalf("EditFile failed: %v", err)
	}
	if !editResult.Success {
		t.Fatalf("EditFile returned success=false: %s", editResult.Error)
	}
	if editResult.Replacements != 3 {
		t.Errorf("Expected 3 replacements, got %d", editResult.Replacements)
	}

	// Read and verify
	readResult, err := service.ReadFile(ctx, LocalReadFileParams{Path: testFile})
	if err != nil {
		t.Fatalf("ReadFile failed: %v", err)
	}
	expectedContent := "Hi World\nHi World\nHi Universe"
	if readResult.Content != expectedContent {
		t.Errorf("Expected content %q, got %q", expectedContent, readResult.Content)
	}
}

func TestService_RenameFile(t *testing.T) {
	service := NewService()
	ctx := context.Background()

	tmpDir := t.TempDir()
	oldPath := filepath.Join(tmpDir, "old.txt")
	newName := "new.txt"

	// Create test file
	if err := os.WriteFile(oldPath, []byte("test"), 0644); err != nil {
		t.Fatalf("Failed to create test file: %v", err)
	}

	// Rename file
	result, err := service.RenameFile(ctx, RenameLocalFileParams{
		Path:    oldPath,
		NewName: newName,
	})
	if err != nil {
		t.Fatalf("RenameFile failed: %v", err)
	}
	if !result.Success {
		t.Fatalf("RenameFile returned success=false: %s", result.Error)
	}

	expectedNewPath := filepath.Join(tmpDir, newName)
	if result.NewPath != expectedNewPath {
		t.Errorf("Expected new path %q, got %q", expectedNewPath, result.NewPath)
	}

	// Verify old file doesn't exist
	if _, err := os.Stat(oldPath); !os.IsNotExist(err) {
		t.Error("Old file still exists")
	}

	// Verify new file exists
	if _, err := os.Stat(result.NewPath); err != nil {
		t.Errorf("New file doesn't exist: %v", err)
	}
}

func TestService_MoveFiles(t *testing.T) {
	service := NewService()
	ctx := context.Background()

	tmpDir := t.TempDir()
	srcDir := filepath.Join(tmpDir, "src")
	dstDir := filepath.Join(tmpDir, "dst")

	// Create directories
	if err := os.MkdirAll(srcDir, 0755); err != nil {
		t.Fatalf("Failed to create src directory: %v", err)
	}

	// Create test files
	file1 := filepath.Join(srcDir, "file1.txt")
	file2 := filepath.Join(srcDir, "file2.txt")
	if err := os.WriteFile(file1, []byte("test1"), 0644); err != nil {
		t.Fatalf("Failed to create test file: %v", err)
	}
	if err := os.WriteFile(file2, []byte("test2"), 0644); err != nil {
		t.Fatalf("Failed to create test file: %v", err)
	}

	// Move files
	results, err := service.MoveFiles(ctx, MoveLocalFilesParams{
		Items: []MoveLocalFileParams{
			{OldPath: file1, NewPath: filepath.Join(dstDir, "file1.txt")},
			{OldPath: file2, NewPath: filepath.Join(dstDir, "file2.txt")},
		},
	})
	if err != nil {
		t.Fatalf("MoveFiles failed: %v", err)
	}

	if len(results) != 2 {
		t.Fatalf("Expected 2 results, got %d", len(results))
	}

	for _, result := range results {
		if !result.Success {
			t.Errorf("Move failed for %s: %s", result.SourcePath, result.Error)
		}
	}

	// Verify files moved
	if _, err := os.Stat(filepath.Join(dstDir, "file1.txt")); err != nil {
		t.Errorf("file1.txt not found in destination: %v", err)
	}
	if _, err := os.Stat(filepath.Join(dstDir, "file2.txt")); err != nil {
		t.Errorf("file2.txt not found in destination: %v", err)
	}
}

func TestService_SearchFiles(t *testing.T) {
	service := NewService()
	ctx := context.Background()

	tmpDir := t.TempDir()

	// Create test files
	files := map[string]string{
		"test1.txt":     "content",
		"test2.txt":     "content",
		"example.txt":   "content",
		"test_file.log": "content",
	}
	for name, content := range files {
		path := filepath.Join(tmpDir, name)
		if err := os.WriteFile(path, []byte(content), 0644); err != nil {
			t.Fatalf("Failed to create test file: %v", err)
		}
	}

	// Search for files containing "test"
	results, err := service.SearchFiles(ctx, LocalSearchFilesParams{
		Keywords:  "test",
		Directory: tmpDir,
	})
	if err != nil {
		t.Fatalf("SearchFiles failed: %v", err)
	}

	if len(results) != 3 { // test1.txt, test2.txt, test_file.log
		t.Errorf("Expected 3 results, got %d", len(results))
	}
}

func TestService_RunCommand(t *testing.T) {
	service := NewService()
	ctx := context.Background()

	// Run simple command
	result, err := service.RunCommand(ctx, RunCommandParams{
		Command: "echo 'Hello, World!'",
	})
	if err != nil {
		t.Fatalf("RunCommand failed: %v", err)
	}
	if !result.Success {
		t.Fatalf("RunCommand returned success=false: %s", result.Error)
	}
	if result.ExitCode != 0 {
		t.Errorf("Expected exit code 0, got %d", result.ExitCode)
	}
}

func TestService_GlobFiles(t *testing.T) {
	service := NewService()
	ctx := context.Background()

	tmpDir := t.TempDir()

	// Create test files
	files := []string{"test1.txt", "test2.txt", "example.log", "test3.md"}
	for _, f := range files {
		path := filepath.Join(tmpDir, f)
		if err := os.WriteFile(path, []byte("test"), 0644); err != nil {
			t.Fatalf("Failed to create test file: %v", err)
		}
	}

	// Glob for .txt files
	result, err := service.GlobFiles(ctx, GlobFilesParams{
		Pattern: "*.txt",
		Path:    tmpDir,
	})
	if err != nil {
		t.Fatalf("GlobFiles failed: %v", err)
	}
	if !result.Success {
		t.Fatal("GlobFiles returned success=false")
	}
	if result.TotalFiles != 2 {
		t.Errorf("Expected 2 .txt files, got %d", result.TotalFiles)
	}
}

func TestService_GrepContent(t *testing.T) {
	service := NewService()
	ctx := context.Background()

	tmpDir := t.TempDir()

	// Create test files with content
	files := map[string]string{
		"file1.txt": "Hello World\nGoodbye World",
		"file2.txt": "Hello Universe\nGoodbye Universe",
		"file3.txt": "No match here",
	}
	for name, content := range files {
		path := filepath.Join(tmpDir, name)
		if err := os.WriteFile(path, []byte(content), 0644); err != nil {
			t.Fatalf("Failed to create test file: %v", err)
		}
	}

	// Grep for "Hello"
	result, err := service.GrepContent(ctx, GrepContentParams{
		Pattern: "Hello",
		Path:    tmpDir,
	})
	if err != nil {
		t.Fatalf("GrepContent failed: %v", err)
	}
	if !result.Success {
		t.Fatal("GrepContent returned success=false")
	}
	if result.TotalMatches != 2 {
		t.Errorf("Expected 2 matches, got %d", result.TotalMatches)
	}
}
