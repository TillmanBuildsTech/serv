// Package platform defines the ServiceManager interface that all
// platform-specific service management implementations must satisfy.
package platform

import "github.com/TillmanBuildsTech/serv/pkg/api"

// ServiceStatus represents the current state of a managed service.
type ServiceStatus struct {
	// State is the current running state (e.g. "running", "stopped", "paused").
	State string
	// PID is the process identifier, or 0 if the service is not running.
	PID int
	// ExitCode is the last exit code of the service process.
	ExitCode int
}

// ServiceInfo provides summary information about an installed service.
type ServiceInfo struct {
	// Name is the unique service identifier.
	Name string
	// DisplayName is the human-readable service name.
	DisplayName string
	// State is the current running state.
	State string
	// PID is the process identifier, or 0 if the service is not running.
	PID int
}

// ServiceManager is the interface that platform-specific implementations must
// satisfy to install, control, and query services.
type ServiceManager interface {
	// Install registers a new service from the given configuration.
	Install(cfg *api.ServiceConfig) error
	// Remove unregisters and deletes a service by name.
	Remove(name string) error
	// Start launches a stopped service.
	Start(name string) error
	// Stop halts a running service.
	Stop(name string) error
	// Restart stops and then starts a service.
	Restart(name string) error
	// Status returns the current status of a service.
	Status(name string) (ServiceStatus, error)
	// List returns information about all installed services.
	List() ([]ServiceInfo, error)
	// UpdateConfig applies a new configuration to an existing service.
	UpdateConfig(name string, cfg *api.ServiceConfig) error
}
