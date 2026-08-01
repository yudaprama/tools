package localfs_test

import (
	"context"
	"fmt"
	"log"

	"github.com/yudaprama/tools/localfs"
)

func ExampleService_WriteFile() {
	service := localfs.NewService()
	ctx := context.Background()

	result, err := service.WriteFile(ctx, localfs.WriteLocalFileParams{
		Path:    "/tmp/example.txt",
		Content: "Hello, World!",
	})
	if err != nil {
		log.Fatal(err)
	}

	fmt.Printf("Success: %v\n", result.Success)
	// Output: Success: true
}

func ExampleService_ReadFile() {
	service := localfs.NewService()
	ctx := context.Background()

	// First write a file
	_, _ = service.WriteFile(ctx, localfs.WriteLocalFileParams{
		Path:    "/tmp/example.txt",
		Content: "Line 1\nLine 2\nLine 3",
	})

	// Read the file
	result, err := service.ReadFile(ctx, localfs.LocalReadFileParams{
		Path: "/tmp/example.txt",
	})
	if err != nil {
		log.Fatal(err)
	}

	fmt.Printf("Lines: %d\n", result.LineCount)
	// Output: Lines: 3
}

func ExampleService_ReadFile_withLineRange() {
	service := localfs.NewService()
	ctx := context.Background()

	// First write a file
	_, _ = service.WriteFile(ctx, localfs.WriteLocalFileParams{
		Path:    "/tmp/example.txt",
		Content: "Line 1\nLine 2\nLine 3\nLine 4\nLine 5",
	})

	// Read lines 2-4
	loc := [2]int{2, 4}
	result, err := service.ReadFile(ctx, localfs.LocalReadFileParams{
		Path: "/tmp/example.txt",
		Loc:  &loc,
	})
	if err != nil {
		log.Fatal(err)
	}

	fmt.Printf("Content: %s\n", result.Content)
	fmt.Printf("Line count: %d\n", result.LineCount)
	fmt.Printf("Total lines: %d\n", result.TotalLineCount)
	// Output:
	// Content: Line 2
	// Line 3
	// Line 4
	// Line count: 3
	// Total lines: 5
}

func ExampleService_EditFile() {
	service := localfs.NewService()
	ctx := context.Background()

	// Write a file
	_, _ = service.WriteFile(ctx, localfs.WriteLocalFileParams{
		Path:    "/tmp/example.txt",
		Content: "Hello World\nHello Universe",
	})

	// Edit the file - replace first occurrence
	result, err := service.EditFile(ctx, localfs.EditLocalFileParams{
		FilePath:   "/tmp/example.txt",
		OldString:  "Hello",
		NewString:  "Hi",
		ReplaceAll: false,
	})
	if err != nil {
		log.Fatal(err)
	}

	fmt.Printf("Replacements: %d\n", result.Replacements)
	// Output: Replacements: 1
}

func ExampleService_EditFile_replaceAll() {
	service := localfs.NewService()
	ctx := context.Background()

	// Write a file
	_, _ = service.WriteFile(ctx, localfs.WriteLocalFileParams{
		Path:    "/tmp/example.txt",
		Content: "Hello World\nHello Universe\nHello Galaxy",
	})

	// Edit the file - replace all occurrences
	result, err := service.EditFile(ctx, localfs.EditLocalFileParams{
		FilePath:   "/tmp/example.txt",
		OldString:  "Hello",
		NewString:  "Hi",
		ReplaceAll: true,
	})
	if err != nil {
		log.Fatal(err)
	}

	fmt.Printf("Replacements: %d\n", result.Replacements)
	// Output: Replacements: 3
}

func ExampleService_RunCommand() {
	service := localfs.NewService()
	ctx := context.Background()

	result, err := service.RunCommand(ctx, localfs.RunCommandParams{
		Command: "echo 'Hello from shell'",
	})
	if err != nil {
		log.Fatal(err)
	}

	fmt.Printf("Success: %v\n", result.Success)
	fmt.Printf("Exit code: %d\n", result.ExitCode)
	// Output:
	// Success: true
	// Exit code: 0
}

func ExampleService_SearchFiles() {
	service := localfs.NewService()
	ctx := context.Background()

	// This example assumes files exist in /tmp
	results, err := service.SearchFiles(ctx, localfs.LocalSearchFilesParams{
		Keywords:  "example",
		Directory: "/tmp",
	})
	if err != nil {
		log.Fatal(err)
	}

	fmt.Printf("Found %d files\n", len(results))
}

func ExampleService_GlobFiles() {
	service := localfs.NewService()
	ctx := context.Background()

	result, err := service.GlobFiles(ctx, localfs.GlobFilesParams{
		Pattern: "*.txt",
		Path:    "/tmp",
	})
	if err != nil {
		log.Fatal(err)
	}

	fmt.Printf("Success: %v\n", result.Success)
	fmt.Printf("Found %d files\n", result.TotalFiles)
}

func ExampleService_GrepContent() {
	service := localfs.NewService()
	ctx := context.Background()

	result, err := service.GrepContent(ctx, localfs.GrepContentParams{
		Pattern: "Hello",
		Path:    "/tmp",
		CaseI:   true,
	})
	if err != nil {
		log.Fatal(err)
	}

	fmt.Printf("Success: %v\n", result.Success)
	fmt.Printf("Total matches: %d\n", result.TotalMatches)
}

func ExampleNewService() {
	// Create a new local file service
	service := localfs.NewService()
	ctx := context.Background()

	// Use the service for file operations
	_, err := service.WriteFile(ctx, localfs.WriteLocalFileParams{
		Path:    "/tmp/test.txt",
		Content: "Test content",
	})
	if err != nil {
		log.Fatal(err)
	}

	fmt.Println("Service created and used successfully")
	// Output: Service created and used successfully
}
