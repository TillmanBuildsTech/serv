//go:build !windows && !linux && !darwin

package platform

// NewServiceManager returns a stub manager on unsupported platforms.
func NewServiceManager() ServiceManager {
	return &stubManager{}
}
