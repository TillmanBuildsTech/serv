//go:build windows

package platform

// NewServiceManager returns the ServiceManager for Windows.
func NewServiceManager() ServiceManager {
	return &windowsManager{}
}
