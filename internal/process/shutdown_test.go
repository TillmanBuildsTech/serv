package process

import (
	"context"
	"errors"
	"os"
	"testing"
	"time"

	"github.com/TillmanBuildsTech/serv/pkg/api"
)

func TestShutdownExitsEarlyIfAlreadyDone(t *testing.T) {
	done := make(chan struct{})
	close(done) // process already exited before Shutdown even runs a stage

	if err := Shutdown(context.Background(), 1234, done, api.StopConfig{}); err != nil {
		t.Fatalf("Shutdown: unexpected error: %v", err)
	}
}

func TestRunStagesEscalatesThenExits(t *testing.T) {
	done := make(chan struct{})

	var ran []string
	stages := []shutdownStage{
		{name: "one", timeout: 20 * time.Millisecond, run: func() error { ran = append(ran, "one"); return nil }},
		{name: "two", timeout: 20 * time.Millisecond, run: func() error {
			ran = append(ran, "two")
			close(done) // simulate the process exiting during this stage
			return nil
		}},
		{name: "three", timeout: 20 * time.Millisecond, run: func() error { ran = append(ran, "three"); return nil }},
	}

	if err := runStages(context.Background(), stages, done, 1234); err != nil {
		t.Fatalf("runStages: unexpected error: %v", err)
	}
	if len(ran) != 2 || ran[0] != "one" || ran[1] != "two" {
		t.Errorf("expected stages [one two] to run, got %v", ran)
	}
}

func TestRunStagesAllTimeOut(t *testing.T) {
	done := make(chan struct{}) // never closes

	var ran []string
	stages := []shutdownStage{
		{name: "one", timeout: 5 * time.Millisecond, run: func() error { ran = append(ran, "one"); return nil }},
		{name: "two", timeout: 5 * time.Millisecond, run: func() error { ran = append(ran, "two"); return nil }},
	}

	err := runStages(context.Background(), stages, done, 1234)
	if err == nil {
		t.Fatal("runStages: expected error when all stages time out")
	}
	if len(ran) != 2 {
		t.Errorf("expected both stages to run, got %v", ran)
	}
}

func TestRunStagesContextCancelled(t *testing.T) {
	done := make(chan struct{}) // never closes
	ctx, cancel := context.WithCancel(context.Background())

	var ran []string
	stages := []shutdownStage{
		{name: "one", timeout: time.Second, run: func() error {
			ran = append(ran, "one")
			cancel() // cancel mid-stage
			return nil
		}},
		{name: "two", timeout: time.Second, run: func() error { ran = append(ran, "two"); return nil }},
	}

	err := runStages(ctx, stages, done, 1234)
	if !errors.Is(err, context.Canceled) {
		t.Fatalf("runStages: expected context.Canceled, got %v", err)
	}
	if len(ran) != 1 || ran[0] != "one" {
		t.Errorf("expected only stage 'one' to run, got %v", ran)
	}
}

func TestRunStagesIgnoresStageRunError(t *testing.T) {
	done := make(chan struct{})
	close(done)

	stages := []shutdownStage{
		{name: "one", timeout: time.Second, run: func() error { return errors.New("boom") }},
	}

	if err := runStages(context.Background(), stages, done, 1234); err != nil {
		t.Fatalf("runStages: unexpected error: %v", err)
	}
}

func TestDurationOrDefault(t *testing.T) {
	def := 5 * time.Second
	if got := durationOrDefault(api.Duration(0), def); got != def {
		t.Errorf("durationOrDefault(0) = %v, want %v", got, def)
	}
	set := api.Duration(3 * time.Second)
	if got := durationOrDefault(set, def); got != 3*time.Second {
		t.Errorf("durationOrDefault(set) = %v, want 3s", got)
	}
}

func TestMethodEnabled(t *testing.T) {
	if !methodEnabled(nil, api.StopMethodConsole) {
		t.Error("methodEnabled: empty methods should enable everything")
	}
	methods := []api.StopMethod{api.StopMethodTerminate}
	if methodEnabled(methods, api.StopMethodConsole) {
		t.Error("methodEnabled: console should be disabled when not listed")
	}
	if !methodEnabled(methods, api.StopMethodTerminate) {
		t.Error("methodEnabled: terminate should be enabled when listed")
	}
}

// TestShutdownTerminatesUnresponsiveProcess launches a real helper process
// that ignores termination signals, then runs the full platform shutdown
// escalation against it with short timeouts and confirms it is eventually
// force-killed.
func TestShutdownTerminatesUnresponsiveProcess(t *testing.T) {
	cfg := &api.ServiceConfig{
		Name:       "helper",
		Executable: os.Args[0],
		Environment: map[string]string{
			"GO_HELPER_MODE": "ignore_signals",
		},
	}

	mp, err := StartProcess(cfg)
	if err != nil {
		t.Fatalf("StartProcess: unexpected error: %v", err)
	}

	stopCfg := api.StopConfig{
		ConsoleTimeout:   api.Duration(50 * time.Millisecond),
		WindowTimeout:    api.Duration(50 * time.Millisecond),
		ThreadsTimeout:   api.Duration(50 * time.Millisecond),
		TerminateTimeout: api.Duration(2 * time.Second),
	}

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	if err := Shutdown(ctx, mp.PID, mp.Done(), stopCfg); err != nil {
		t.Fatalf("Shutdown: unexpected error: %v", err)
	}

	select {
	case <-mp.Done():
	case <-time.After(time.Second):
		t.Fatal("process did not report exit after Shutdown returned")
	}
}
