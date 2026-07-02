package platform

import (
	"errors"
	"testing"

	"github.com/TillmanBuildsTech/serv/pkg/api"
)

// Compile-time check: MockManager implements ServiceManager.
var _ ServiceManager = (*MockManager)(nil)

// Compile-time check: stubManager implements ServiceManager.
var _ ServiceManager = (*stubManager)(nil)

func TestNewServiceManagerReturnsNonNil(t *testing.T) {
	mgr := NewServiceManager()
	if mgr == nil {
		t.Fatal("NewServiceManager() returned nil")
	}
}

func TestMockManagerDefaults(t *testing.T) {
	m := &MockManager{}

	if err := m.Install(&api.ServiceConfig{Name: "test"}); err != nil {
		t.Fatalf("Install: unexpected error: %v", err)
	}
	if err := m.Remove("test"); err != nil {
		t.Fatalf("Remove: unexpected error: %v", err)
	}
	if err := m.Start("test"); err != nil {
		t.Fatalf("Start: unexpected error: %v", err)
	}
	if err := m.Stop("test"); err != nil {
		t.Fatalf("Stop: unexpected error: %v", err)
	}
	if err := m.Restart("test"); err != nil {
		t.Fatalf("Restart: unexpected error: %v", err)
	}

	status, err := m.Status("test")
	if err != nil {
		t.Fatalf("Status: unexpected error: %v", err)
	}
	if status.State != "stopped" {
		t.Fatalf("Status: expected state 'stopped', got %q", status.State)
	}

	list, err := m.List()
	if err != nil {
		t.Fatalf("List: unexpected error: %v", err)
	}
	if list != nil {
		t.Fatalf("List: expected nil, got %v", list)
	}

	if err := m.UpdateConfig("test", &api.ServiceConfig{}); err != nil {
		t.Fatalf("UpdateConfig: unexpected error: %v", err)
	}
}

func TestMockManagerCustomFuncs(t *testing.T) {
	errTest := errors.New("test error")
	m := &MockManager{
		InstallFunc: func(_ *api.ServiceConfig) error { return errTest },
		StatusFunc: func(name string) (ServiceStatus, error) {
			return ServiceStatus{State: "running", PID: 1234}, nil
		},
		ListFunc: func() ([]ServiceInfo, error) {
			return []ServiceInfo{{Name: "svc1", State: "running"}}, nil
		},
	}

	if err := m.Install(&api.ServiceConfig{}); !errors.Is(err, errTest) {
		t.Fatalf("Install: expected errTest, got %v", err)
	}

	status, err := m.Status("svc1")
	if err != nil {
		t.Fatalf("Status: unexpected error: %v", err)
	}
	if status.State != "running" || status.PID != 1234 {
		t.Fatalf("Status: unexpected result: %+v", status)
	}

	list, err := m.List()
	if err != nil {
		t.Fatalf("List: unexpected error: %v", err)
	}
	if len(list) != 1 || list[0].Name != "svc1" {
		t.Fatalf("List: unexpected result: %+v", list)
	}
}

func TestStubManagerReturnsUnsupported(t *testing.T) {
	s := &stubManager{}

	if err := s.Install(&api.ServiceConfig{}); err == nil {
		t.Fatal("Install: expected error, got nil")
	}
	if err := s.Remove("x"); err == nil {
		t.Fatal("Remove: expected error, got nil")
	}
	if err := s.Start("x"); err == nil {
		t.Fatal("Start: expected error, got nil")
	}
	if err := s.Stop("x"); err == nil {
		t.Fatal("Stop: expected error, got nil")
	}
	if err := s.Restart("x"); err == nil {
		t.Fatal("Restart: expected error, got nil")
	}
	if _, err := s.Status("x"); err == nil {
		t.Fatal("Status: expected error, got nil")
	}
	if _, err := s.List(); err == nil {
		t.Fatal("List: expected error, got nil")
	}
	if err := s.UpdateConfig("x", &api.ServiceConfig{}); err == nil {
		t.Fatal("UpdateConfig: expected error, got nil")
	}
}
