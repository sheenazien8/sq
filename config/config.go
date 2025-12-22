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
	Theme  string            `json:"theme"`
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

	config := &SQLSConfig{
		LowercaseKeywords: false,
		Connections:       make([]SQLSConnectionConfig, 0, len(connections)),
	}

	for _, conn := range connections {
		sqlsConn := SQLSConnectionConfig{
			Alias:  conn.Name,
			Driver: conn.Driver,
		}

		// Convert URL to proper DSN format for sqls
		// sqls expects: user:password@tcp(host:port)/dbname for MySQL
		dsn, err := convertURLToDSN(conn.URL, conn.Driver)
		if err != nil {
			// If conversion fails, try using the URL as-is
			sqlsConn.DataSourceName = conn.URL
		} else {
			sqlsConn.DataSourceName = dsn
		}

		config.Connections = append(config.Connections, sqlsConn)
	}

	return config, nil
}

// convertURLToDSN converts a database URL to DSN format for sqls
func convertURLToDSN(url string, driver string) (string, error) {
	// Parse URL format: mysql://user:password@host:port/dbname
	// Convert to DSN: user:password@tcp(host:port)/dbname

	if driver == "mysql" {
		// Remove mysql:// prefix
		dsn := url
		if len(dsn) > 8 && dsn[:8] == "mysql://" {
			dsn = dsn[8:]
		}

		// Find @ separator
		atIdx := -1
		for i := 0; i < len(dsn); i++ {
			if dsn[i] == '@' {
				atIdx = i
				break
			}
		}

		if atIdx == -1 {
			return url, nil // No credentials, return as-is
		}

		userPass := dsn[:atIdx]
		hostPath := dsn[atIdx+1:]

		// Find / separator for host:port/dbname
		slashIdx := -1
		for i := 0; i < len(hostPath); i++ {
			if hostPath[i] == '/' {
				slashIdx = i
				break
			}
		}

		var host, dbName string
		if slashIdx == -1 {
			host = hostPath
			dbName = ""
		} else {
			host = hostPath[:slashIdx]
			dbName = hostPath[slashIdx+1:]
		}

		// Build DSN: user:password@tcp(host:port)/dbname
		return fmt.Sprintf("%s@tcp(%s)/%s", userPass, host, dbName), nil
	}

	// For other drivers, return as-is
	return url, nil
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
