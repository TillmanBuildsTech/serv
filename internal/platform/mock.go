package platform

import "github.com/TillmanBuildsTech/serv/pkg/api"

// MockManager is a test-friendly ServiceManager that records calls and returns
// configurable responses. Use it in unit tests to verify interactions with the
// ServiceManager interface without depending on a real platform.
type MockManager struct {
	InstallFunc      func(cfg *api.ServiceConfig) error
	RemoveFunc       func(name string) error
	StartFunc        func(name string) error
	StopFunc         func(name string) error
	RestartFunc      func(name string) error
	StatusFunc       func(name string) (ServiceStatus, error)
	ListFunc         func() ([]ServiceInfo, error)
	UpdateConfigFunc func(name string, cfg *api.ServiceConfig) error
}

func (m *MockManager) Install(cfg *api.ServiceConfig) error {
	if m.InstallFunc != nil {
		return m.InstallFunc(cfg)
	}
	return nil
}

func (m *MockManager) Remove(name string) error {
	if m.RemoveFunc != nil {
		return m.RemoveFunc(name)
	}
	return nil
}

func (m *MockManager) Start(name string) error {
	if m.StartFunc != nil {
		return m.StartFunc(name)
	}
	return nil
}

func (m *MockManager) Stop(name string) error {
	if m.StopFunc != nil {
		return m.StopFunc(name)
	}
	return nil
}

func (m *MockManager) Restart(name string) error {
	if m.RestartFunc != nil {
		return m.RestartFunc(name)
	}
	return nil
}

func (m *MockManager) Status(name string) (ServiceStatus, error) {
	if m.StatusFunc != nil {
		return m.StatusFunc(name)
	}
	return ServiceStatus{State: "stopped"}, nil
}

func (m *MockManager) List() ([]ServiceInfo, error) {
	if m.ListFunc != nil {
		return m.ListFunc()
	}
	return nil, nil
}

func (m *MockManager) UpdateConfig(name string, cfg *api.ServiceConfig) error {
	if m.UpdateConfigFunc != nil {
		return m.UpdateConfigFunc(name, cfg)
	}
	return nil
}
