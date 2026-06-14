package process

import (
	"context"
	"errors"
	stdexec "os/exec"
	"strings"
	"testing"
	"time"
)

func TestExecRunnerCapturesCombinedOutput(t *testing.T) {
	t.Parallel()
	out, err := ExecRunner{}.Run(context.Background(), "echo", "hi")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got := strings.TrimSpace(string(out)); got != "hi" {
		t.Errorf("output = %q, want %q", got, "hi")
	}
}

func TestExecRunnerMissingBinaryReturnsNotFound(t *testing.T) {
	t.Parallel()
	_, err := ExecRunner{}.Run(context.Background(), "definitely-not-a-real-binary-xyz")
	if err == nil {
		t.Fatal("expected error for missing binary, got nil")
	}
	if !errors.Is(err, stdexec.ErrNotFound) {
		t.Errorf("error = %v, want errors.Is(err, exec.ErrNotFound)", err)
	}
}

func TestExecRunnerHonorsContextCancellation(t *testing.T) {
	t.Parallel()
	ctx, cancel := context.WithTimeout(context.Background(), 50*time.Millisecond)
	defer cancel()

	_, err := ExecRunner{}.Run(ctx, "sleep", "5")
	if err == nil {
		t.Fatal("expected error from cancelled context, got nil")
	}
}
