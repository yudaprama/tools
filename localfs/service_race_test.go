package localfs

import (
	"context"
	"runtime"
	"testing"
)

// Test concurrent access to shellCommands map to ensure no data race
func TestService_ShellCommandsRace(t *testing.T) {
	service := NewService()
	ctx := context.Background()

	// Start a simple background command
	cmdStr := "sleep 0.1"
	if runtime.GOOS == "windows" {
		cmdStr = "ping -n 2 127.0.0.1 > NUL"
	}

	result, err := service.RunCommand(ctx, RunCommandParams{
		Command:         cmdStr,
		RunInBackground: true,
	})
	if err != nil {
		t.Fatalf("RunCommand background failed: %v", err)
	}
	shellID := result.ShellID

	done := make(chan struct{})

	go func() {
		defer close(done)
		for i := 0; i < 1000; i++ {
			_, _ = service.GetCommandOutput(ctx, GetCommandOutputParams{ShellID: shellID})
		}
	}()

	for i := 0; i < 1000; i++ {
		_, _ = service.GetCommandOutput(ctx, GetCommandOutputParams{ShellID: shellID})
	}

	<-done

	_, _ = service.KillCommand(ctx, KillCommandParams{ShellID: shellID})
}
