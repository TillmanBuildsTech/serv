package config

import (
	"fmt"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/TillmanBuildsTech/serv/pkg/api"
)

const validYAMLTemplate = `
name: testapp
display_name: Test Application
description: A test service
executable: %s
arguments:
  - --port
  - "9090"
working_directory: %s
start_type: auto
stop_method:
  methods: [console, terminate]
  console_timeout: 2s
  terminate_timeout: 3s
restart:
  enabled: true
  delay: 500ms
  throttle_cap: 2m
exit_actions:
  0: exit
  1: restart
stdout: %s/stdout.log
stderr: %s/stderr.log
log_rotation:
  enabled: true
  max_bytes: 5242880
  max_age: 72h
  online_rotation: true
environment:
  APP_ENV: test
kill_process_tree: true
priority: high
hooks:
  pre_start: %s/hook.sh
dependencies:
  - redis
`

func writeTestConfig(t *testing.T, content string) string {
	t.Helper()
	dir := t.TempDir()
	path := filepath.Join(dir, "config.yaml")
	if err := os.WriteFile(path, []byte(content), 0644); err != nil {
		t.Fatalf("writing test config: %v", err)
	}
	return path
}

func tempExecutable(t *testing.T) string {
	t.Helper()
	dir := t.TempDir()
	exe := filepath.Join(dir, "app.exe")
	if err := os.WriteFile(exe, []byte("#!/bin/sh\n"), 0755); err != nil {
		t.Fatalf("creating temp executable: %v", err)
	}
	return exe
}

func TestParse_FullConfig(t *testing.T) {
	dir := t.TempDir()
	exe := filepath.Join(dir, "app.exe")
	os.WriteFile(exe, []byte{}, 0755)

	yaml := fmt.Sprintf(validYAMLTemplate, exe, dir, dir, dir, dir)
	cfg, err := Parse([]byte(yaml))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if cfg.Name != "testapp" {
		t.Errorf("expected name testapp, got %s", cfg.Name)
	}
	if cfg.DisplayName != "Test Application" {
		t.Errorf("expected display_name 'Test Application', got %s", cfg.DisplayName)
	}
	if cfg.Executable != exe {
		t.Errorf("expected executable %s, got %s", exe, cfg.Executable)
	}
	if len(cfg.Arguments) != 2 || cfg.Arguments[1] != "9090" {
		t.Errorf("unexpected arguments: %v", cfg.Arguments)
	}
	if cfg.StartType != api.StartTypeAuto {
		t.Errorf("expected start_type auto, got %s", cfg.StartType)
	}
	if len(cfg.StopMethod.Methods) != 2 {
		t.Errorf("expected 2 stop methods, got %d", len(cfg.StopMethod.Methods))
	}
	if cfg.StopMethod.ConsoleTimeout != api.Duration(2*time.Second) {
		t.Errorf("expected console_timeout 2s, got %v", cfg.StopMethod.ConsoleTimeout)
	}
	if cfg.StopMethod.TerminateTimeout != api.Duration(3*time.Second) {
		t.Errorf("expected terminate_timeout 3s, got %v", cfg.StopMethod.TerminateTimeout)
	}
	// window_timeout was not set, so defaults should have been applied
	if cfg.StopMethod.WindowTimeout != api.Duration(1500*time.Millisecond) {
		t.Errorf("expected window_timeout default 1500ms, got %v", cfg.StopMethod.WindowTimeout)
	}
	if cfg.Restart.Enabled == nil || !*cfg.Restart.Enabled {
		t.Error("expected restart.enabled true")
	}
	if cfg.Restart.Delay != api.Duration(500*time.Millisecond) {
		t.Errorf("expected restart.delay 500ms, got %v", cfg.Restart.Delay)
	}
	if cfg.ExitActions[0] != api.ExitActionExit {
		t.Errorf("expected exit_actions[0]=exit, got %s", cfg.ExitActions[0])
	}
	if cfg.ExitActions[1] != api.ExitActionRestart {
		t.Errorf("expected exit_actions[1]=restart, got %s", cfg.ExitActions[1])
	}
	if !cfg.LogRotation.Enabled {
		t.Error("expected log_rotation.enabled true")
	}
	if cfg.LogRotation.MaxBytes != 5242880 {
		t.Errorf("expected max_bytes 5242880, got %d", cfg.LogRotation.MaxBytes)
	}
	if cfg.Environment["APP_ENV"] != "test" {
		t.Errorf("unexpected environment: %v", cfg.Environment)
	}
	if cfg.KillProcessTree == nil || !*cfg.KillProcessTree {
		t.Error("expected kill_process_tree true")
	}
	if cfg.Priority != "high" {
		t.Errorf("expected priority high, got %s", cfg.Priority)
	}
	if len(cfg.Dependencies) != 1 || cfg.Dependencies[0] != "redis" {
		t.Errorf("unexpected dependencies: %v", cfg.Dependencies)
	}
}

func TestParse_MinimalConfig(t *testing.T) {
	exe := tempExecutable(t)
	yaml := fmt.Sprintf("name: minimal\nexecutable: %s\n", exe)
	cfg, err := Parse([]byte(yaml))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if cfg.Name != "minimal" {
		t.Errorf("expected name minimal, got %s", cfg.Name)
	}
	// Defaults should be applied
	if cfg.StartType != api.StartTypeAuto {
		t.Errorf("expected default start_type auto, got %s", cfg.StartType)
	}
	if cfg.Restart.Enabled == nil || !*cfg.Restart.Enabled {
		t.Error("expected restart.enabled defaulted to true")
	}
	if cfg.KillProcessTree == nil || !*cfg.KillProcessTree {
		t.Error("expected kill_process_tree defaulted to true")
	}
}

func TestParse_InvalidYAML(t *testing.T) {
	_, err := Parse([]byte("{{{invalid"))
	if err == nil {
		t.Fatal("expected error for invalid YAML")
	}
}

func TestLoad_FromFile(t *testing.T) {
	exe := tempExecutable(t)
	yaml := fmt.Sprintf("name: filetest\nexecutable: %s\n", exe)
	path := writeTestConfig(t, yaml)

	cfg, err := Load(path)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if cfg.Name != "filetest" {
		t.Errorf("expected name filetest, got %s", cfg.Name)
	}
}

func TestLoad_FileNotFound(t *testing.T) {
	_, err := Load("/nonexistent/config.yaml")
	if err == nil {
		t.Fatal("expected error for missing file")
	}
}

func TestDefaultConfigPath(t *testing.T) {
	path := DefaultConfigPath("myservice")
	if path == "" {
		t.Fatal("expected non-empty default config path")
	}
	if !filepath.IsAbs(path) {
		t.Errorf("expected absolute path, got %s", path)
	}
}
