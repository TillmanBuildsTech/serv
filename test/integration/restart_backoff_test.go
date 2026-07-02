//go:build integration

package integration

import (
	"testing"
	"time"

	"github.com/TillmanBuildsTech/serv/internal/process"
	"github.com/TillmanBuildsTech/serv/pkg/api"
)

// TestAutomaticRestartWithBackoff drives a small supervision loop around
// the helper fixture (configured to exit immediately with a failure code)
// using process.StartProcess and process.Throttle directly — the same
// primitives internal/service's runtime uses — and confirms each restart's
// backoff delay escalates.
func TestAutomaticRestartWithBackoff(t *testing.T) {
	exe := helperBinary(t)
	cfg := baseConfig("restartbackoff", exe, "-exit-code=1", "-exit-after=10ms")
	cfg.Restart.Delay = api.Duration(50 * time.Millisecond)
	cfg.Restart.ThrottleCap = api.Duration(2 * time.Second)

	throttle := process.NewThrottle(cfg.Restart)

	var delays []time.Duration
	for i := 0; i < 3; i++ {
		mp, err := process.StartProcess(cfg)
		if err != nil {
			t.Fatalf("StartProcess (attempt %d): %v", i, err)
		}
		throttle.RecordStart(mp.StartTime)

		select {
		case <-mp.Done():
		case <-time.After(5 * time.Second):
			t.Fatalf("helper did not exit within timeout on attempt %d", i)
		}

		code, _ := mp.Wait()
		if code != 1 {
			t.Fatalf("attempt %d: exit code = %d, want 1", i, code)
		}

		delay := throttle.NextDelay(time.Now())
		delays = append(delays, delay)
		time.Sleep(delay)
	}

	for i := 1; i < len(delays); i++ {
		if delays[i] <= delays[i-1] {
			t.Errorf("backoff did not escalate: delays = %v", delays)
			break
		}
	}
}
