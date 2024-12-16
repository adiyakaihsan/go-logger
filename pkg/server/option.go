package server

import "time"

type ServerConfig struct {
	ShutdownTimer time.Duration
	Port          string
	IndexName     string
}

type ServerOption func(*ServerConfig)

var defaultConfig = &ServerConfig{
	ShutdownTimer: 5 * time.Second,
	Port:          "8080",
}

func ShutdownTimer(duration time.Duration) ServerOption {
	return func(cfg *ServerConfig) {
		cfg.ShutdownTimer = duration
	}
}

func Port(port string) ServerOption {
	return func(cfg *ServerConfig) {
		cfg.Port = port
	}
}

func IndexName(indexName string) ServerOption {
	return func(cfg *ServerConfig) {
		cfg.IndexName = indexName
	}
}

func applyOptions(cfg *ServerConfig, opts ...ServerOption) *ServerConfig {
	// Create a copy of the default config to avoid modifying the original
	appliedCfg := *cfg
	for _, opt := range opts {
		opt(&appliedCfg)
	}
	return &appliedCfg
}
