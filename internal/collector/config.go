package collector

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"

	"gopkg.in/yaml.v3"
)

// CollectorConfig represents a collector configuration
type CollectorConfig struct {
	Name         string   `json:"name" yaml:"name"`
	APIKey       string   `json:"api_key" yaml:"api_key"`
	AccessToken  string   `json:"access_token" yaml:"access_token"`
	AutoStart    bool     `json:"auto_start" yaml:"auto_start"`
	Symbols      []string `json:"symbols" yaml:"symbols"`
	Watchlists   []string `json:"watchlists" yaml:"watchlists"`
	Mode         string   `json:"mode" yaml:"mode"` // ltp, quote, full
}

// AutoStartConfig represents auto-start configuration file
type AutoStartConfig struct {
	Collectors []CollectorConfig `json:"collectors" yaml:"collectors"`
}

// LoadConfigFromFile loads collector configuration from a file
func LoadConfigFromFile(filePath string) (*AutoStartConfig, error) {
	data, err := os.ReadFile(filePath)
	if err != nil {
		return nil, fmt.Errorf("failed to read config file: %w", err)
	}

	var config AutoStartConfig

	// Determine file type by extension
	ext := filepath.Ext(filePath)
	switch ext {
	case ".json":
		if err := json.Unmarshal(data, &config); err != nil {
			return nil, fmt.Errorf("failed to parse JSON config: %w", err)
		}
	case ".yaml", ".yml":
		if err := yaml.Unmarshal(data, &config); err != nil {
			return nil, fmt.Errorf("failed to parse YAML config: %w", err)
		}
	default:
		return nil, fmt.Errorf("unsupported config file format: %s (use .json or .yaml)", ext)
	}

	return &config, nil
}

// SaveConfigToFile saves collector configuration to a file
func SaveConfigToFile(filePath string, config *AutoStartConfig) error {
	ext := filepath.Ext(filePath)

	var data []byte
	var err error

	switch ext {
	case ".json":
		data, err = json.MarshalIndent(config, "", "  ")
	case ".yaml", ".yml":
		data, err = yaml.Marshal(config)
	default:
		return fmt.Errorf("unsupported config file format: %s (use .json or .yaml)", ext)
	}

	if err != nil {
		return fmt.Errorf("failed to marshal config: %w", err)
	}

	if err := os.WriteFile(filePath, data, 0644); err != nil {
		return fmt.Errorf("failed to write config file: %w", err)
	}

	return nil
}

// GetDefaultConfigPath returns the default config file path
func GetDefaultConfigPath() string {
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return "./collectors.yaml"
	}
	return filepath.Join(homeDir, ".market-bridge", "collectors.yaml")
}

// CreateDefaultConfig creates a default configuration file
func CreateDefaultConfig(filePath string) error {
	config := &AutoStartConfig{
		Collectors: []CollectorConfig{
			{
				Name:        "default",
				APIKey:      "${ZERODHA_API_KEY}",
				AccessToken: "${ZERODHA_ACCESS_TOKEN}",
				AutoStart:   false,
				Watchlists:  []string{"NIFTY50", "BANKNIFTY"},
				Mode:        "full",
			},
		},
	}

	// Create directory if it doesn't exist
	dir := filepath.Dir(filePath)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return fmt.Errorf("failed to create config directory: %w", err)
	}

	return SaveConfigToFile(filePath, config)
}

// ExpandEnvVars expands environment variables in config
func ExpandEnvVars(config *AutoStartConfig) {
	for i := range config.Collectors {
		config.Collectors[i].APIKey = os.ExpandEnv(config.Collectors[i].APIKey)
		config.Collectors[i].AccessToken = os.ExpandEnv(config.Collectors[i].AccessToken)
	}
}
