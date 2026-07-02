//go:build linux || darwin

package service

import (
	"net"
	"os"
	"path/filepath"
	"syscall"
	"testing"
	"time"
)

func TestWaitBackoffUnixElapsesFully(t *testing.T) {
	sig := make(chan os.Signal, 1)

	start := time.Now()
	stopped := waitBackoffUnix(sig, 20*time.Millisecond)
	elapsed := time.Since(start)

	if stopped {
		t.Error("waitBackoffUnix: expected false when no stop signal arrives")
	}
	if elapsed < 20*time.Millisecond {
		t.Errorf("waitBackoffUnix returned after %v, want >= 20ms", elapsed)
	}
}

func TestWaitBackoffUnixInterruptedBySIGTERM(t *testing.T) {
	sig := make(chan os.Signal, 1)
	sig <- syscall.SIGTERM

	start := time.Now()
	stopped := waitBackoffUnix(sig, 5*time.Second)
	elapsed := time.Since(start)

	if !stopped {
		t.Error("waitBackoffUnix: expected true when SIGTERM arrives")
	}
	if elapsed > 500*time.Millisecond {
		t.Errorf("waitBackoffUnix took %v to react to SIGTERM, want fast", elapsed)
	}
}

func TestWaitBackoffUnixInterruptedBySIGINT(t *testing.T) {
	sig := make(chan os.Signal, 1)
	sig <- syscall.SIGINT

	if !waitBackoffUnix(sig, 5*time.Second) {
		t.Error("waitBackoffUnix: expected true when SIGINT arrives")
	}
}

func TestWaitBackoffUnixIgnoresSIGHUP(t *testing.T) {
	sig := make(chan os.Signal, 1)
	sig <- syscall.SIGHUP

	start := time.Now()
	stopped := waitBackoffUnix(sig, 30*time.Millisecond)
	elapsed := time.Since(start)

	if stopped {
		t.Error("waitBackoffUnix: SIGHUP should not be treated as a stop request")
	}
	if elapsed < 30*time.Millisecond {
		t.Errorf("waitBackoffUnix returned after %v despite SIGHUP, want it to still wait out the delay", elapsed)
	}
}

func TestNotifyReadyNoopWithoutSocket(t *testing.T) {
	t.Setenv("NOTIFY_SOCKET", "")
	// Must not panic or block when NOTIFY_SOCKET is unset.
	notifyReady()
}

func TestNotifyReadySendsReadyMessage(t *testing.T) {
	socketPath := filepath.Join(t.TempDir(), "notify.sock")

	conn, err := net.ListenUnixgram("unixgram", &net.UnixAddr{Name: socketPath, Net: "unixgram"})
	if err != nil {
		t.Fatalf("ListenUnixgram: %v", err)
	}
	defer conn.Close()

	t.Setenv("NOTIFY_SOCKET", socketPath)
	notifyReady()

	buf := make([]byte, 64)
	conn.SetReadDeadline(time.Now().Add(2 * time.Second))
	n, err := conn.Read(buf)
	if err != nil {
		t.Fatalf("reading notify message: %v", err)
	}
	if got := string(buf[:n]); got != "READY=1" {
		t.Errorf("notify message = %q, want %q", got, "READY=1")
	}
}
