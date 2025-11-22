package config

import (
	"fmt"
	"os"

	"gopkg.in/yaml.v3"
)

// Config represents the application configuration
type Config struct {
	Server struct {
		Domain    string `yaml:"domain"`
		BaseURL   string `yaml:"base_url"`
		SSHPort   string `yaml:"ssh_port"`
		HTTPPort  string `yaml:"http_port"`
		HTTPSPort string `yaml:"https_port"`
		TLS       struct {
			CertFile string `yaml:"cert_file"`
			KeyFile  string `yaml:"key_file"`
			AutoCert bool   `yaml:"auto_cert"`
		} `yaml:"tls"`
	} `yaml:"server"`

	Database struct {
		Postgres struct {
			Host           string `yaml:"host"`
			Port           int    `yaml:"port"`
			User           string `yaml:"user"`
			Password       string `yaml:"password"`
			Database       string `yaml:"database"`
			SSLMode        string `yaml:"sslmode"`
			MaxConnections int    `yaml:"max_connections"`
		} `yaml:"postgres"`
		Redis struct {
			Host     string `yaml:"host"`
			Port     int    `yaml:"port"`
			Password string `yaml:"password"`
			DB       int    `yaml:"db"`
		} `yaml:"redis"`
	} `yaml:"database"`

	OAuth struct {
		DeviceCodeExpiry int    `yaml:"device_code_expiry"`
		PollInterval     int    `yaml:"poll_interval"`
		CallbackURL      string `yaml:"callback_url"`
	} `yaml:"oauth"`

	ActivityPub struct {
		Enabled          bool   `yaml:"enabled"`
		UserAgent        string `yaml:"user_agent"`
		MaxInboxSize     int    `yaml:"max_inbox_size"`
		DeliveryWorkers  int    `yaml:"delivery_workers"`
		InboxWorkers     int    `yaml:"inbox_workers"`
		RetryMaxAttempts int    `yaml:"retry_max_attempts"`
		RetryBaseDelay   int    `yaml:"retry_base_delay"`
	} `yaml:"activitypub"`

	Features struct {
		ChatRoulette struct {
			Enabled      bool `yaml:"enabled"`
			QueueTimeout int  `yaml:"queue_timeout"`
		} `yaml:"chatroulette"`
		AnonymousPosting struct {
			Enabled   bool `yaml:"enabled"`
			RateLimit int  `yaml:"rate_limit"`
		} `yaml:"anonymous_posting"`
		Registration struct {
			Enabled       bool `yaml:"enabled"`
			RequireInvite bool `yaml:"require_invite"`
		} `yaml:"registration"`
	} `yaml:"features"`

	Security struct {
		RateLimiting struct {
			Enabled           bool `yaml:"enabled"`
			RequestsPerMinute int  `yaml:"requests_per_minute"`
		} `yaml:"rate_limiting"`
		BlockedInstances []string `yaml:"blocked_instances"`
	} `yaml:"security"`

	Logging struct {
		Level  string `yaml:"level"`
		Format string `yaml:"format"`
		Output string `yaml:"output"`
	} `yaml:"logging"`
}

// Load reads and parses the configuration file
func Load(path string) (*Config, error) {
	// Expand environment variables in path
	path = os.ExpandEnv(path)

	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("failed to read config file: %w", err)
	}

	// Expand environment variables in config content
	expanded := os.ExpandEnv(string(data))

	var cfg Config
	if err := yaml.Unmarshal([]byte(expanded), &cfg); err != nil {
		return nil, fmt.Errorf("failed to parse config file: %w", err)
	}

	return &cfg, nil
}

// LoadOrDefault loads config from path, or returns default if file doesn't exist
func LoadOrDefault(path string) *Config {
	cfg, err := Load(path)
	if err != nil {
		return DefaultConfig()
	}
	return cfg
}

// DefaultConfig returns a configuration with sensible defaults
func DefaultConfig() *Config {
	cfg := &Config{}

	// Server defaults
	cfg.Server.Domain = "localhost"
	cfg.Server.BaseURL = "http://localhost:8080"
	cfg.Server.SSHPort = "22"
	cfg.Server.HTTPPort = "8080"
	cfg.Server.HTTPSPort = "443"

	// Database defaults
	cfg.Database.Postgres.Host = "localhost"
	cfg.Database.Postgres.Port = 5432
	cfg.Database.Postgres.User = "terminalpub"
	cfg.Database.Postgres.Password = "terminalpub_dev_password"
	cfg.Database.Postgres.Database = "terminalpub"
	cfg.Database.Postgres.SSLMode = "disable"
	cfg.Database.Postgres.MaxConnections = 25

	cfg.Database.Redis.Host = "localhost"
	cfg.Database.Redis.Port = 6379
	cfg.Database.Redis.DB = 0

	// OAuth defaults
	cfg.OAuth.DeviceCodeExpiry = 600
	cfg.OAuth.PollInterval = 5
	cfg.OAuth.CallbackURL = "http://localhost:8080/oauth/callback"

	// ActivityPub defaults
	cfg.ActivityPub.Enabled = true
	cfg.ActivityPub.UserAgent = "terminalpub/0.1.0"
	cfg.ActivityPub.MaxInboxSize = 1000
	cfg.ActivityPub.DeliveryWorkers = 10
	cfg.ActivityPub.InboxWorkers = 5
	cfg.ActivityPub.RetryMaxAttempts = 5
	cfg.ActivityPub.RetryBaseDelay = 30

	// Features defaults
	cfg.Features.ChatRoulette.Enabled = true
	cfg.Features.ChatRoulette.QueueTimeout = 300
	cfg.Features.AnonymousPosting.Enabled = true
	cfg.Features.AnonymousPosting.RateLimit = 10
	cfg.Features.Registration.Enabled = true
	cfg.Features.Registration.RequireInvite = false

	// Security defaults
	cfg.Security.RateLimiting.Enabled = true
	cfg.Security.RateLimiting.RequestsPerMinute = 60
	cfg.Security.BlockedInstances = []string{}

	// Logging defaults
	cfg.Logging.Level = "info"
	cfg.Logging.Format = "json"
	cfg.Logging.Output = "stdout"

	return cfg
}
