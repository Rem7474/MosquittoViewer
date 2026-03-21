package config

import (
	"fmt"
	"os"
	"time"

	"gopkg.in/yaml.v3"
)

type Config struct {
	Server ServerConfig      `yaml:"server"`
	Logs   []LogSourceConfig `yaml:"logs"`
	Auth   AuthConfig        `yaml:"auth"`
	SQLite SQLiteConfig      `yaml:"sqlite"`
	Debug  bool              `yaml:"debug"`
}

type ServerConfig struct {
	Port int    `yaml:"port"`
	Host string `yaml:"host"`
}

// LogSourceConfig defines a single log file source.
type LogSourceConfig struct {
	Name                string `yaml:"name"`
	Path                string `yaml:"path"`
	Format              string `yaml:"format"`
	CustomRegex         string `yaml:"custom_regex"`
	BufferSize          int    `yaml:"buffer_size"`
	ReadExistingOnStart bool   `yaml:"read_existing_on_start"`
}

type AuthConfig struct {
	Users []UserConfig `yaml:"users"`
	JWT   JWTConfig    `yaml:"jwt"`
}

type UserConfig struct {
	Username     string `yaml:"username"`
	PasswordHash string `yaml:"password_hash"`
}

type JWTConfig struct {
	PrivateKeyPath  string `yaml:"private_key_path"`
	PublicKeyPath   string `yaml:"public_key_path"`
	AccessTokenTTL  string `yaml:"access_token_ttl"`
	RefreshTokenTTL string `yaml:"refresh_token_ttl"`
}

type SQLiteConfig struct {
	Enabled bool   `yaml:"enabled"`
	Path    string `yaml:"path"`
}

func Load(path string) (Config, error) {
	b, err := os.ReadFile(path)
	if err != nil {
		return Config{}, fmt.Errorf("read config: %w", err)
	}
	var cfg Config
	if err := yaml.Unmarshal(b, &cfg); err != nil {
		return Config{}, fmt.Errorf("parse config: %w", err)
	}
	if cfg.Server.Host == "" {
		cfg.Server.Host = "127.0.0.1"
	}
	if cfg.Server.Port == 0 {
		cfg.Server.Port = 8080
	}
	for i := range cfg.Logs {
		if cfg.Logs[i].Name == "" {
			cfg.Logs[i].Name = fmt.Sprintf("source%d", i+1)
		}
		if cfg.Logs[i].BufferSize <= 0 {
			cfg.Logs[i].BufferSize = 500
		}
		if cfg.Logs[i].Format == "" {
			cfg.Logs[i].Format = "mosquitto_standard"
		}
	}
	if cfg.Auth.JWT.AccessTokenTTL == "" {
		cfg.Auth.JWT.AccessTokenTTL = "15m"
	}
	if cfg.Auth.JWT.RefreshTokenTTL == "" {
		cfg.Auth.JWT.RefreshTokenTTL = (7 * 24 * time.Hour).String()
	}
	return cfg, nil
}
