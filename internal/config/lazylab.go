package config

import (
	"os"
	"path/filepath"

	"gopkg.in/yaml.v3"
)

// LazyLabConfig represents the lazylab configuration
type LazyLabConfig struct {
	DefaultHost string                    `yaml:"default_host,omitempty"`
	Hosts       map[string]LazyLabHost    `yaml:"hosts,omitempty"`
}

// LazyLabHost represents a GitLab host configuration
type LazyLabHost struct {
	Token string `yaml:"token"`
}

// GetConfigDir returns the lazylab config directory path
func GetConfigDir() (string, error) {
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(homeDir, ".config", "lazylab"), nil
}

// GetConfigPath returns the lazylab config file path
func GetConfigPath() (string, error) {
	configDir, err := GetConfigDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(configDir, "config.yaml"), nil
}

// LoadLazyLabConfig reads the lazylab configuration
func LoadLazyLabConfig() (*LazyLabConfig, error) {
	configPath, err := GetConfigPath()
	if err != nil {
		return nil, err
	}

	data, err := os.ReadFile(configPath)
	if err != nil {
		return nil, err
	}

	var config LazyLabConfig
	if err := yaml.Unmarshal(data, &config); err != nil {
		return nil, err
	}

	return &config, nil
}

// SaveLazyLabConfig writes the lazylab configuration
func SaveLazyLabConfig(cfg *LazyLabConfig) error {
	configDir, err := GetConfigDir()
	if err != nil {
		return err
	}

	// Create config directory if it doesn't exist
	if err := os.MkdirAll(configDir, 0700); err != nil {
		return err
	}

	configPath, err := GetConfigPath()
	if err != nil {
		return err
	}

	data, err := yaml.Marshal(cfg)
	if err != nil {
		return err
	}

	return os.WriteFile(configPath, data, 0600)
}

// GetHostConfig returns the configuration for a specific host
func (c *LazyLabConfig) GetHostConfig(host string) *LazyLabHost {
	if c.Hosts == nil {
		return nil
	}
	if hostConfig, ok := c.Hosts[host]; ok {
		return &hostConfig
	}
	return nil
}

// SetHostToken sets the token for a specific host
func (c *LazyLabConfig) SetHostToken(host, token string) {
	if c.Hosts == nil {
		c.Hosts = make(map[string]LazyLabHost)
	}
	c.Hosts[host] = LazyLabHost{Token: token}
}

// GetDefaultHost returns the default host
func (c *LazyLabConfig) GetDefaultHost() string {
	if c.DefaultHost != "" {
		return c.DefaultHost
	}
	// Return first host if no default set
	for host := range c.Hosts {
		return host
	}
	return DefaultHost
}
