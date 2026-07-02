//go:build linux

package platform

// NewServiceManager returns the ServiceManager for Linux.
func NewServiceManager() ServiceManager {
	return &linuxManager{}
}
