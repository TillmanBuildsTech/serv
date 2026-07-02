//go:build windows

package process

import (
	"os"
	"testing"
	"time"

	"github.com/TillmanBuildsTech/serv/pkg/api"
)

func TestStartTimeReturnsRecentTime(t *testing.T) {
	cfg := &api.ServiceConfig{
		Name:       "helper",
		Executable: os.Args[0],
		Environment: map[string]string{
			"GO_HELPER_MODE": "ignore_signals",
		},
	}

	before := time.Now().Add(-2 * time.Second)
	mp, err := StartProcess(cfg)
	if err != nil {
		t.Fatalf("StartProcess: unexpected error: %v", err)
	}
	after := time.Now().Add(2 * time.Second)

	got, ok := StartTime(mp.PID)
	if !ok {
		t.Fatal("StartTime: expected ok=true for a live process")
	}
	if got.Before(before) || got.After(after) {
		t.Errorf("StartTime = %v, want between %v and %v", got, before, after)
	}

	_ = KillTree(mp.PID)
}
