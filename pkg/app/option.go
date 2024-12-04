package app

import "time"

type Config struct {
	IndexName     string
	RetentionDays time.Duration
	ShutdownTimer time.Duration
	Port          string
}

type Option func(*Config)

var defaultConfig = &Config{
	IndexName:     "index-storage/index",
	RetentionDays: 12 * 24 * time.Hour,
	ShutdownTimer: 5 * time.Second,
	Port:          "8080",
}

func IndexName(indexName string) Option {
	return func(cfg *Config) {
		cfg.IndexName = indexName
	}
}

func RetentionDays(retDays time.Duration) Option {
	return func(cfg *Config) {
		cfg.RetentionDays = retDays
	}
}

func ShutdownTimer(duration time.Duration) Option {
	return func(cfg *Config) {
		cfg.ShutdownTimer = duration
	}
}

func Port(port string) Option {
	return func(cfg *Config) {
		cfg.Port = port
	}
}

func applyOptions(cfg *Config, opts ...Option) *Config {
	// Create a copy of the default config to avoid modifying the original
	appliedCfg := *cfg
	for _, opt := range opts {
		opt(&appliedCfg)
	}
	return &appliedCfg
}
