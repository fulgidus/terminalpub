package config

import (
	"os"
	"testing"
)

func TestDefaultConfig(t *testing.T) {
	cfg := DefaultConfig()

	if cfg.Server.SSHPort != "22" {
		t.Errorf("Expected SSH port 22, got %s", cfg.Server.SSHPort)
	}

	if cfg.Database.Postgres.Host != "localhost" {
		t.Errorf("Expected postgres host localhost, got %s", cfg.Database.Postgres.Host)
	}

	if cfg.Database.Redis.Port != 6379 {
		t.Errorf("Expected redis port 6379, got %d", cfg.Database.Redis.Port)
	}
}

func TestLoadConfig(t *testing.T) {
	// Create a temporary config file
	tmpfile, err := os.CreateTemp("", "config-*.yaml")
	if err != nil {
		t.Fatal(err)
	}
	defer os.Remove(tmpfile.Name())

	content := `
server:
  domain: test.example.com
  ssh_port: "2222"
database:
  postgres:
    host: db.example.com
    port: 5433
`
	if _, err := tmpfile.Write([]byte(content)); err != nil {
		t.Fatal(err)
	}
	if err := tmpfile.Close(); err != nil {
		t.Fatal(err)
	}

	cfg, err := Load(tmpfile.Name())
	if err != nil {
		t.Fatalf("Failed to load config: %v", err)
	}

	if cfg.Server.Domain != "test.example.com" {
		t.Errorf("Expected domain test.example.com, got %s", cfg.Server.Domain)
	}

	if cfg.Server.SSHPort != "2222" {
		t.Errorf("Expected SSH port 2222, got %s", cfg.Server.SSHPort)
	}

	if cfg.Database.Postgres.Host != "db.example.com" {
		t.Errorf("Expected postgres host db.example.com, got %s", cfg.Database.Postgres.Host)
	}

	if cfg.Database.Postgres.Port != 5433 {
		t.Errorf("Expected postgres port 5433, got %d", cfg.Database.Postgres.Port)
	}
}

func TestLoadOrDefault(t *testing.T) {
	// Test with non-existent file
	cfg := LoadOrDefault("/non/existent/file.yaml")
	if cfg.Server.SSHPort != "22" {
		t.Error("LoadOrDefault should return default config for non-existent file")
	}
}

func TestEnvironmentVariableExpansion(t *testing.T) {
	os.Setenv("TEST_PASSWORD", "secret123")
	defer os.Unsetenv("TEST_PASSWORD")

	tmpfile, err := os.CreateTemp("", "config-*.yaml")
	if err != nil {
		t.Fatal(err)
	}
	defer os.Remove(tmpfile.Name())

	content := `
database:
  postgres:
    password: ${TEST_PASSWORD}
`
	if _, err := tmpfile.Write([]byte(content)); err != nil {
		t.Fatal(err)
	}
	if err := tmpfile.Close(); err != nil {
		t.Fatal(err)
	}

	cfg, err := Load(tmpfile.Name())
	if err != nil {
		t.Fatalf("Failed to load config: %v", err)
	}

	if cfg.Database.Postgres.Password != "secret123" {
		t.Errorf("Expected password 'secret123', got '%s'", cfg.Database.Postgres.Password)
	}
}
