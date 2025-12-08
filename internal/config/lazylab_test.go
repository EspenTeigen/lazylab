package config

import (
	"os"
	"path/filepath"
	"testing"
)

func TestLazyLabConfig_SetHostToken(t *testing.T) {
	cfg := &LazyLabConfig{}

	cfg.SetHostToken("gitlab.com", "test-token")

	if cfg.Hosts == nil {
		t.Fatal("expected hosts map to be initialized")
	}

	host := cfg.GetHostConfig("gitlab.com")
	if host == nil {
		t.Fatal("expected host config to exist")
	}

	if host.Token != "test-token" {
		t.Errorf("expected token 'test-token', got '%s'", host.Token)
	}
}

func TestLazyLabConfig_GetDefaultHost(t *testing.T) {
	tests := []struct {
		name     string
		cfg      *LazyLabConfig
		expected string
	}{
		{
			name:     "empty config returns default",
			cfg:      &LazyLabConfig{},
			expected: DefaultHost,
		},
		{
			name: "returns configured default host",
			cfg: &LazyLabConfig{
				DefaultHost: "gitlab.mycompany.com",
			},
			expected: "gitlab.mycompany.com",
		},
		{
			name: "returns first host if no default set",
			cfg: &LazyLabConfig{
				Hosts: map[string]LazyLabHost{
					"gitlab.example.com": {Token: "token"},
				},
			},
			expected: "gitlab.example.com",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := tt.cfg.GetDefaultHost()
			if result != tt.expected {
				t.Errorf("expected '%s', got '%s'", tt.expected, result)
			}
		})
	}
}

func TestLazyLabConfig_GetHostConfig(t *testing.T) {
	cfg := &LazyLabConfig{
		Hosts: map[string]LazyLabHost{
			"gitlab.com": {Token: "token1"},
		},
	}

	// Existing host
	host := cfg.GetHostConfig("gitlab.com")
	if host == nil {
		t.Fatal("expected host config for gitlab.com")
	}
	if host.Token != "token1" {
		t.Errorf("expected token 'token1', got '%s'", host.Token)
	}

	// Non-existing host
	host = cfg.GetHostConfig("other.com")
	if host != nil {
		t.Error("expected nil for non-existing host")
	}

	// Nil hosts map
	cfg2 := &LazyLabConfig{}
	host = cfg2.GetHostConfig("gitlab.com")
	if host != nil {
		t.Error("expected nil for nil hosts map")
	}
}

func TestSaveAndLoadLazyLabConfig(t *testing.T) {
	// Create temp directory
	tmpDir, err := os.MkdirTemp("", "lazylab-test")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	// Override config path for test
	origHome := os.Getenv("HOME")
	os.Setenv("HOME", tmpDir)
	defer os.Setenv("HOME", origHome)

	// Create and save config
	cfg := &LazyLabConfig{
		DefaultHost: "gitlab.example.com",
	}
	cfg.SetHostToken("gitlab.example.com", "my-secret-token")

	err = SaveLazyLabConfig(cfg)
	if err != nil {
		t.Fatalf("failed to save config: %v", err)
	}

	// Verify file exists
	configPath := filepath.Join(tmpDir, ".config", "lazylab", "config.yaml")
	if _, err := os.Stat(configPath); os.IsNotExist(err) {
		t.Fatal("config file was not created")
	}

	// Load and verify
	loaded, err := LoadLazyLabConfig()
	if err != nil {
		t.Fatalf("failed to load config: %v", err)
	}

	if loaded.DefaultHost != "gitlab.example.com" {
		t.Errorf("expected default host 'gitlab.example.com', got '%s'", loaded.DefaultHost)
	}

	host := loaded.GetHostConfig("gitlab.example.com")
	if host == nil {
		t.Fatal("expected host config to exist after load")
	}
	if host.Token != "my-secret-token" {
		t.Errorf("expected token 'my-secret-token', got '%s'", host.Token)
	}
}

func TestLoadLazyLabConfig_NotExists(t *testing.T) {
	// Create temp directory with no config
	tmpDir, err := os.MkdirTemp("", "lazylab-test")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	origHome := os.Getenv("HOME")
	os.Setenv("HOME", tmpDir)
	defer os.Setenv("HOME", origHome)

	_, err = LoadLazyLabConfig()
	if err == nil {
		t.Error("expected error when config doesn't exist")
	}
}
