package platform

import (
	"fmt"

	"github.com/TillmanBuildsTech/serv/pkg/api"
)

// stubManager is a no-op ServiceManager returned on unsupported platforms
// and as a placeholder until real platform implementations are built.
type stubManager struct{}

var errUnsupported = fmt.Errorf("service management is not supported on this platform")

func (s *stubManager) Install(_ *api.ServiceConfig) error            { return errUnsupported }
func (s *stubManager) Remove(_ string) error                         { return errUnsupported }
func (s *stubManager) Start(_ string) error                          { return errUnsupported }
func (s *stubManager) Stop(_ string) error                           { return errUnsupported }
func (s *stubManager) Restart(_ string) error                        { return errUnsupported }
func (s *stubManager) Status(_ string) (ServiceStatus, error)        { return ServiceStatus{}, errUnsupported }
func (s *stubManager) List() ([]ServiceInfo, error)                  { return nil, errUnsupported }
func (s *stubManager) UpdateConfig(_ string, _ *api.ServiceConfig) error { return errUnsupported }
