package app

import (
	"fmt"
	"log"

	"github.com/adiyakaihsan/go-logger/pkg/queue"
)

type App struct {
	queue     queue.Queue
	ilm       *IndexLifecycleManager
	processor *LogProcessor
}

func NewApp(opts ...Option) (*App, error) {
	logQueue, err := queue.NewNatsQueue("nats://localhost:4222", "log", "logQueue", true)
	if err != nil {
		log.Fatalf("Failed to initiate channel. Error: %v", err)
	}

	cfg := applyOptions(defaultConfig, opts...)

	for _, opt := range opts {
		opt(cfg)
	}

	ilm, err := NewIndexLifecycleManager(cfg.IndexName, cfg.RetentionDays)
	if err != nil {
		log.Fatalf("Failed to initiate index. Error: %v", err)
	}

	processor := NewLogProcessor(logQueue, ilm)

	app := &App{
		queue:     logQueue,
		ilm:       ilm,
		processor: processor,
	}

	return app, nil
}

func (a *App) Start() error {
	if err := a.ilm.StartScheduler(); err != nil {
		log.Printf("Error starting ILM Scheduler. Error: %v", err)
	}

	if err := a.processor.Start(); err != nil {
		return fmt.Errorf("failed to start log processor: %w", err)
	}

	return nil
}

func (a *App) Shutdown() error {
	a.ilm.StopScheduler()
	a.queue.Close()
	if err := a.processor.Shutdown(); err != nil {
		return fmt.Errorf("processor shutdown failed: %w", err)
	}

	return nil
}

// func getEnvDefault(key, defaultValue string) string {
// 	if value, exists := os.LookupEnv(key); exists {
// 		return value
// 	}
// 	return defaultValue
// }
