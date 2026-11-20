package config

import (
	"errors"
	"fmt"
	"os"
	"strconv"

	"github.com/spf13/viper"
)

type Config struct {
	HTTPAddr string
	Postgres Postgres
	DeepSeek DeepSeek
}

type Postgres struct {
	Host     string
	Port     int
	Database string
	User     string
	Password string
	SSLMode  string
}

type DeepSeek struct {
	APIKey  string `json:"-"`
	BaseURL string
	Model   string
}

func Load() (Config, error) {
	v := viper.New()
	path := os.Getenv("APP_CONFIG_FILE")
	v.SetConfigFile("configs/config.yaml")
	if err := v.ReadInConfig(); err != nil && (path == "" || !errors.Is(err, os.ErrNotExist)) {
		return Config{}, fmt.Errorf("read default config: %w", err)
	}
	if path != "" {
		v.SetConfigFile(path)
		if err := v.MergeInConfig(); err != nil {
			return Config{}, fmt.Errorf("read APP_CONFIG_FILE: %w", err)
		}
	}

	userBinding := []string{"postgres.user", "PG_USER"}
	// Shells normally set USER. Prefer the YAML user unless legacy credentials are supplied.
	if os.Getenv("PASSWROD") != "" || v.GetString("postgres.user") == "" {
		userBinding = append(userBinding, "USER")
	}

	for _, binding := range [][]string{
		{"http.addr", "HTTP_ADDR"},
		{"postgres.host", "PG_IP"},
		{"postgres.port", "PG_PORT"},
		{"postgres.database", "PG_DATABASE"},
		userBinding,
		{"postgres.password", "PG_PASSWORD", "PASSWROD"},
		{"postgres.sslmode", "PG_SSLMODE"},
		{"deepseek.api_key", "DS_API_KEY"},
		{"deepseek.base_url", "DS_BASE_URL"},
		{"deepseek.model", "DS_MODEL"},
	} {
		if err := v.BindEnv(binding...); err != nil {
			return Config{}, fmt.Errorf("bind %s: %w", binding[0], err)
		}
	}

	port, err := strconv.Atoi(v.GetString("postgres.port"))
	if err != nil || port < 1 || port > 65535 {
		return Config{}, fmt.Errorf("postgres.port (PG_PORT) must be between 1 and 65535")
	}

	cfg := Config{
		HTTPAddr: v.GetString("http.addr"),
		Postgres: Postgres{
			Host:     v.GetString("postgres.host"),
			Port:     port,
			Database: v.GetString("postgres.database"),
			User:     v.GetString("postgres.user"),
			Password: v.GetString("postgres.password"),
			SSLMode:  v.GetString("postgres.sslmode"),
		},
		DeepSeek: DeepSeek{
			APIKey:  v.GetString("deepseek.api_key"),
			BaseURL: v.GetString("deepseek.base_url"),
			Model:   v.GetString("deepseek.model"),
		},
	}

	if cfg.HTTPAddr == "" {
		return Config{}, fmt.Errorf("http.addr (HTTP_ADDR) must be set")
	}
	if cfg.Postgres.Host == "" || cfg.Postgres.Database == "" || cfg.Postgres.User == "" {
		return Config{}, fmt.Errorf("postgres.host, postgres.database and postgres.user must be set")
	}
	return cfg, nil
}
