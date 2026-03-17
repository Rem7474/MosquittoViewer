package config

import (
	"fmt"
	"os"
	"time"

	"gopkg.in/yaml.v3"
)

type Config struct {
	Server ServerConfig `yaml:"server"`
	Log    LogConfig    `yaml:"log"`
	Auth   AuthConfig   `yaml:"auth"`
	SQLite SQLiteConfig `yaml:"sqlite"`
	Debug  bool         `yaml:"debug"`
}

type ServerConfig struct {
	Port int    `yaml:"port"`
	Host string `yaml:"host"`
}

type LogConfig struct {
	Path        string `yaml:"path"`
	Format      string `yaml:"format"`
	CustomRegex string `yaml:"custom_regex"`
	BufferSize  int    `yaml:"buffer_size"`
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
	if cfg.Log.BufferSize <= 0 {
		cfg.Log.BufferSize = 1000
	}
	if cfg.Auth.JWT.AccessTokenTTL == "" {
		cfg.Auth.JWT.AccessTokenTTL = "15m"
	}
	if cfg.Auth.JWT.RefreshTokenTTL == "" {
		cfg.Auth.JWT.RefreshTokenTTL = (7 * 24 * time.Hour).String()
	}
	return cfg, nil
}
