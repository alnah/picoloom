//go:build integration && !windows

package picoloom

import (
	"context"
	"errors"
	"syscall"
	"testing"
	"time"
)

func TestRodRenderer_CloseTerminatesBrowserProcess(t *testing.T) {
	renderer := newRodRenderer(testTimeout)

	if err := renderer.ensureBrowser(context.Background()); err != nil {
		t.Fatalf("ensureBrowser() unexpected error: %v", err)
	}
	if renderer.launcher == nil {
		t.Fatal("renderer.launcher = nil, want launcher after browser start")
	}

	pid := renderer.launcher.PID()
	if pid <= 0 {
		t.Fatalf("launcher.PID() = %d, want positive browser process PID", pid)
	}

	alive, err := unixProcessExists(pid)
	if err != nil {
		t.Fatalf("checking browser process %d before Close(): %v", pid, err)
	}
	if !alive {
		t.Fatalf("browser process %d is not alive before Close()", pid)
	}

	if err := renderer.Close(); err != nil {
		t.Fatalf("Close() unexpected error: %v", err)
	}

	assertUnixProcessExits(t, pid, 5*time.Second)
}

func assertUnixProcessExits(t *testing.T, pid int, timeout time.Duration) {
	t.Helper()

	deadline := time.Now().Add(timeout)
	for {
		alive, err := unixProcessExists(pid)
		if err != nil {
			t.Fatalf("checking browser process %d after Close(): %v", pid, err)
		}
		if !alive {
			return
		}
		if time.Now().After(deadline) {
			t.Fatalf("browser process %d still exists after Close()", pid)
		}
		time.Sleep(50 * time.Millisecond)
	}
}

func unixProcessExists(pid int) (bool, error) {
	err := syscall.Kill(pid, 0)
	if err == nil {
		return true, nil
	}
	if errors.Is(err, syscall.ESRCH) {
		return false, nil
	}
	if errors.Is(err, syscall.EPERM) {
		return true, nil
	}
	return false, err
}
