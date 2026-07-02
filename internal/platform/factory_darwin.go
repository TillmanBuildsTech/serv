//go:build darwin

package platform

// NewServiceManager returns the ServiceManager for macOS.
func NewServiceManager() ServiceManager {
	return &darwinManager{}
}
