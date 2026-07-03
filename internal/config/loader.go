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
	if dir := os.Getenv("SERV_CONFIG_DIR"); dir != "" {
		return filepath.Join(dir, serviceName, "config.yaml")
	}
	switch runtime.GOOS {
	case "windows":
		programData := os.Getenv("PROGRAMDATA")
		if programData == "" {
			programData = `C:\ProgramData`
		}
		return filepath.Join(programData, "serv", serviceName, "config.yaml")
	case "darwin":
		home, _ := os.UserHomeDir()
		return filepath.Join(home, "Library", "Application Support", "serv", serviceName, "config.yaml")
	default:
		return filepath.Join("/etc", "serv", serviceName, "config.yaml")
	}
}
