package config

import (
	"log"
	"os"
	"path/filepath"

	"github.com/BurntSushi/toml"
)

func defaultDBPath() string {
	configDir, err := os.UserConfigDir()
	if err != nil {
		configDir = "."
	}
	return filepath.Join(configDir, "holetab", "holetab.db")
}

func defaultConfigTOML() string {
	return "[server]\nport = \"3654\"\n\n[database]\npath = \"" + defaultDBPath() + "\"\n"
}

// Config holds all runtime configuration values.
type Config struct {
	Server   ServerConfig   `toml:"server"`
	Database DatabaseConfig `toml:"database"`
}

type ServerConfig struct {
	Port string `toml:"port"`
}

type DatabaseConfig struct {
	Path string `toml:"path"`
}

// LoadConfig reads config.toml from ~/.config/holetab/config.toml.
// If the file does not exist, it creates the directory and writes a default config.
func LoadConfig() (*Config, error) {
	configDir, err := os.UserConfigDir()
	if err != nil {
		return nil, err
	}
	dir := filepath.Join(configDir, "holetab")
	filename := filepath.Join(dir, "config.toml")

	if _, err := os.Stat(filename); os.IsNotExist(err) {
		if err := os.MkdirAll(dir, 0755); err != nil {
			return nil, err
		}
		log.Printf("config.toml not found — writing default config to %s", filename)
		if err := os.WriteFile(filename, []byte(defaultConfigTOML()), 0644); err != nil {
			return nil, err
		}
	}

	var cfg Config
	if _, err := toml.DecodeFile(filename, &cfg); err != nil {
		return nil, err
	}
	if cfg.Database.Path == "" {
		cfg.Database.Path = defaultDBPath()
	}
	return &cfg, nil
}
