package config

import (
	"fmt"
	"os"
	"path/filepath"
	"runtime"

	"github.com/TillmanBuildsTech/serv/pkg/api"
	"gopkg.in/yaml.v3"
)

func Load(path string) (*api.ServiceConfig, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("reading config file: %w", err)
	}
	return Parse(data)
}

func Parse(data []byte) (*api.ServiceConfig, error) {
	var cfg api.ServiceConfig
	if err := yaml.Unmarshal(data, &cfg); err != nil {
		return nil, fmt.Errorf("parsing config: %w", err)
	}
	ApplyDefaults(&cfg)
	return &cfg, nil
}

func DefaultConfigPath(serviceName string) string {
	return filepath.Join(ConfigRoot(), serviceName, "config.yaml")
}

// ConfigRoot returns the per-OS directory holding one subdirectory per
// serv-managed service (overridable with SERV_CONFIG_DIR).
func ConfigRoot() string {
	if dir := os.Getenv("SERV_CONFIG_DIR"); dir != "" {
		return dir
	}
	switch runtime.GOOS {
	case "windows":
		programData := os.Getenv("PROGRAMDATA")
		if programData == "" {
			programData = `C:\ProgramData`
		}
		return filepath.Join(programData, "serv")
	case "darwin":
		home, _ := os.UserHomeDir()
		return filepath.Join(home, "Library", "Application Support", "serv")
	default:
		return filepath.Join("/etc", "serv")
	}
}
