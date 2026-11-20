package config

import (
	"os"
	"path/filepath"
	"testing"
)

func TestLoadYAMLAndOverrides(t *testing.T) {
	t.Chdir("../..")
	t.Setenv("APP_CONFIG_FILE", "")
	t.Setenv("USER", "shell_user")
	t.Setenv("PASSWROD", "")
	t.Setenv("PG_USER", "")
	t.Setenv("PG_IP", "")
	t.Setenv("PG_PORT", "")

	cfg, err := Load()
	if err != nil {
		t.Fatal(err)
	}
	if cfg.Postgres.User != "ai_user" || cfg.Postgres.Port != 55432 {
		t.Fatal("expected database user and port from the default YAML")
	}

	path := filepath.Join(t.TempDir(), "override.yaml")
	if err := os.WriteFile(path, []byte("postgres:\n  user: yaml_user\n  port: 5433\n"), 0600); err != nil {
		t.Fatal(err)
	}
	t.Setenv("APP_CONFIG_FILE", path)
	t.Setenv("PG_PORT", "6543")

	cfg, err = Load()
	if err != nil {
		t.Fatal(err)
	}
	if cfg.Postgres.User != "yaml_user" || cfg.Postgres.Port != 6543 || cfg.Postgres.Host != "127.0.0.1" {
		t.Fatal("expected settings from the environment, override YAML, and default YAML")
	}
}

func TestLoadDatabaseCredentialAliases(t *testing.T) {
	t.Chdir("../..")
	t.Setenv("APP_CONFIG_FILE", "")
	t.Setenv("PG_PORT", "55432")
	t.Setenv("PG_USER", "")
	t.Setenv("PG_PASSWORD", "")
	t.Setenv("USER", "legacy_user")
	t.Setenv("PASSWROD", "legacy_password")

	t.Run("fallback", func(t *testing.T) {
		cfg, err := Load()
		if err != nil {
			t.Fatal(err)
		}
		if cfg.Postgres.User != "legacy_user" || cfg.Postgres.Password != "legacy_password" {
			t.Fatal("expected USER and PASSWROD credentials")
		}
	})

	t.Run("preferred variables", func(t *testing.T) {
		t.Setenv("PG_USER", "preferred_user")
		t.Setenv("PG_PASSWORD", "preferred_password")
		cfg, err := Load()
		if err != nil {
			t.Fatal(err)
		}
		if cfg.Postgres.User != "preferred_user" || cfg.Postgres.Password != "preferred_password" {
			t.Fatal("expected PG_USER and PG_PASSWORD to take precedence")
		}
	})
}
