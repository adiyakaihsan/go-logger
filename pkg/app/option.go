package app

import "time"

type AppConfig struct {
	IndexName     string
	RetentionDays time.Duration
	ShutdownTimer time.Duration
}

type AppOption func(*AppConfig)

var defaultConfig = &AppConfig{
	IndexName:     "index-storage/index",
	RetentionDays: 12 * 24 * time.Hour,
	ShutdownTimer: 5 * time.Second,
}

func IndexName(indexName string) AppOption {
	return func(cfg *AppConfig) {
		cfg.IndexName = indexName
	}
}

func RetentionDays(retDays time.Duration) AppOption {
	return func(cfg *AppConfig) {
		cfg.RetentionDays = retDays
	}
}

func ShutdownTimer(duration time.Duration) AppOption {
	return func(cfg *AppConfig) {
		cfg.ShutdownTimer = duration
	}
}


func applyOptions(cfg *AppConfig, opts ...AppOption) *AppConfig {
	// Create a copy of the default config to avoid modifying the original
	appliedCfg := *cfg
	for _, opt := range opts {
		opt(&appliedCfg)
	}
	return &appliedCfg
}
