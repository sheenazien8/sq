package config

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"

	"github.com/sheenazien8/sq/storage"
	"gopkg.in/yaml.v3"
)

// Config holds the application configuration
type Config struct {
	Theme string `json:"theme"`
}

// SQLSConfig represents the sqls LSP server configuration
type SQLSConfig struct {
	LowercaseKeywords bool                   `yaml:"lowercaseKeywords,omitempty"`
	Connections       []SQLSConnectionConfig `yaml:"connections"`
}

// SQLSConnectionConfig represents a database connection configuration for sqls
type SQLSConnectionConfig struct {
	Alias          string            `yaml:"alias,omitempty"`
	Driver         string            `yaml:"driver"`
	DataSourceName string            `yaml:"dataSourceName,omitempty"`
	Proto          string            `yaml:"proto,omitempty"`
	User           string            `yaml:"user,omitempty"`
	Passwd         string            `yaml:"passwd,omitempty"`
	Host           string            `yaml:"host,omitempty"`
	Port           string            `yaml:"port,omitempty"`
	DbName         string            `yaml:"dbName,omitempty"`
	Params         map[string]string `yaml:"params,omitempty"`
}

// DefaultConfig returns the default configuration
func DefaultConfig() *Config {
	return &Config{
		Theme: "default",
	}
}

// configDir returns the config directory path
func configDir() (string, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(home, ".config", "sq"), nil
}

// configPath returns the config file path
func configPath() (string, error) {
	dir, err := configDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(dir, "config.json"), nil
}

// Load reads the config from disk
func Load() (*Config, error) {
	path, err := configPath()
	if err != nil {
		return DefaultConfig(), err
	}

	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return DefaultConfig(), nil
		}
		return DefaultConfig(), err
	}

	var cfg Config
	if err := json.Unmarshal(data, &cfg); err != nil {
		return DefaultConfig(), err
	}

	return &cfg, nil
}

// Save writes the config to disk
func (c *Config) Save() error {
	dir, err := configDir()
	if err != nil {
		return err
	}

	// Create config directory if it doesn't exist
	if err := os.MkdirAll(dir, 0755); err != nil {
		return err
	}

	path, err := configPath()
	if err != nil {
		return err
	}

	data, err := json.MarshalIndent(c, "", "  ")
	if err != nil {
		return err
	}

	return os.WriteFile(path, data, 0644)
}

// SetTheme updates the theme in config
func (c *Config) SetTheme(themeName string) {
	c.Theme = themeName
}

// GenerateSQLSConfig generates a sqls configuration from saved database connections
func GenerateSQLSConfig() (*SQLSConfig, error) {
	connections, err := storage.GetAllConnections()
	if err != nil {
		return nil, fmt.Errorf("failed to get connections: %w", err)
	}

	fmt.Printf("Found %d database connections for LSP config\n", len(connections))

	config := &SQLSConfig{
		LowercaseKeywords: false,
		Connections:       make([]SQLSConnectionConfig, 0, len(connections)),
	}

	for _, conn := range connections {
		sqlsConn := SQLSConnectionConfig{
			Alias:  conn.Name,
			Driver: conn.Driver,
		}

		// For MySQL, use dataSourceName format
		if conn.Driver == "mysql" {
			sqlsConn.DataSourceName = conn.URL
		} else {
			// For other drivers, we could parse the URL, but for now just use dataSourceName
			sqlsConn.DataSourceName = conn.URL
		}

		config.Connections = append(config.Connections, sqlsConn)
		fmt.Printf("Added connection: %s (%s)\n", conn.Name, conn.Driver)
	}

	if len(config.Connections) == 0 {
		fmt.Println("Warning: No database connections found. LSP completion will not work without connections.")
	}

	return config, nil
}

// SaveSQLSConfig saves the sqls configuration to the config directory
func SaveSQLSConfig() error {
	config, err := GenerateSQLSConfig()
	if err != nil {
		return fmt.Errorf("failed to generate sqls config: %w", err)
	}

	dir, err := configDir()
	if err != nil {
		return err
	}

	path := filepath.Join(dir, "sqls.yml")

	data, err := yaml.Marshal(config)
	if err != nil {
		return fmt.Errorf("failed to marshal sqls config: %w", err)
	}

	return os.WriteFile(path, data, 0644)
}

// GetSQLSConfigPath returns the path to the sqls configuration file
func GetSQLSConfigPath() (string, error) {
	dir, err := configDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(dir, "sqls.yml"), nil
}
