//go:build !windows && !linux && !darwin

package service

import "fmt"

// Run is implemented per-OS: control.go for Windows, run_unix.go for
// Linux/macOS. Other platforms are unsupported.
func Run(name string) error {
	return fmt.Errorf("service runtime is not supported on this platform")
}
