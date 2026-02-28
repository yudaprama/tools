// Package localfs provides local file system operations service.
//
// This package offers a comprehensive service for interacting with the local file system,
// including file operations, shell commands, and search functionality.
//
// Example usage:
//
//	service := localfs.NewService()
//
//	// List files in a directory
//	files, err := service.ListFiles(ctx, "/path/to/dir")
//	if err != nil {
//	    log.Fatal(err)
//	}
//
//	// Read a file
//	content, err := service.ReadFile(ctx, localfs.ReadFileParams{
//	    Path: "/path/to/file.txt",
//	})
//	if err != nil {
//	    log.Fatal(err)
//	}
//
//	// Write a file
//	err = service.WriteFile(ctx, localfs.WriteFileParams{
//	    Path:    "/path/to/file.txt",
//	    Content: "Hello, World!",
//	})
//
//	// Run a command
//	result, err := service.RunCommand(ctx, localfs.RunCommandParams{
//	    Command: "ls -la",
//	})
package localfs
