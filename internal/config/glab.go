package config

import (
	"os"
	"path/filepath"

	"gopkg.in/yaml.v3"
)

// GlabConfig represents the glab CLI configuration
type GlabConfig struct {
	Hosts map[string]GlabHost `yaml:"hosts"`
	Host  string              `yaml:"host"` // default host
}

// GlabHost represents a GitLab host configuration
type GlabHost struct {
	Token       string `yaml:"token"`
	APIHost     string `yaml:"api_host"`
	APIProtocol string `yaml:"api_protocol"`
	User        string `yaml:"user"`
}

// LoadGlabConfig reads the glab CLI configuration
func LoadGlabConfig() (*GlabConfig, error) {
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return nil, err
	}

	configPath := filepath.Join(homeDir, ".config", "glab-cli", "config.yml")
	data, err := os.ReadFile(configPath)
	if err != nil {
		return nil, err
	}

	var config GlabConfig
	if err := yaml.Unmarshal(data, &config); err != nil {
		return nil, err
	}

	return &config, nil
}

// GetHostConfig returns the configuration for a specific host
func (c *GlabConfig) GetHostConfig(host string) *GlabHost {
	if hostConfig, ok := c.Hosts[host]; ok {
		return &hostConfig
	}
	return nil
}

// GetDefaultHost returns the default host from glab config
func (c *GlabConfig) GetDefaultHost() string {
	if c.Host != "" {
		return c.Host
	}
	// Return first host if no default set
	for host := range c.Hosts {
		return host
	}
	return "gitlab.com"
}
