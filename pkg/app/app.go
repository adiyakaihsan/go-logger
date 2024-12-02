package app

import (
	"context"
	"fmt"
	"log"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/adiyakaihsan/go-logger/pkg/queue"
	"github.com/spf13/cobra"
)

type App struct {
	queue     queue.Queue
	ilm       *IndexLifecycleManager
	processor *LogProcessor
}

type Config struct {
	IndexName     string
	RetentionDays time.Duration
	ShutdownTimer time.Duration
	Port          string
}

type Option func(Config)

func IndexName (indexName string) Option {
	return func(cfg Config) {
		cfg.IndexName = indexName
	}
}

func RetentionDays (retDays time.Duration) Option {
	return func(cfg Config) {
		cfg.RetentionDays = retDays
	}
}

func ShutdownTimer (duration time.Duration) Option {
	return func(cfg Config) {
		cfg.ShutdownTimer = duration
	}
}

func Port (port string) Option {
	return func(cfg Config) {
		cfg.Port = port
	}
}
func NewApp(cfg Config, opts ...Option) (*App, error) {
	logQueue, err := queue.NewNatsQueue("nats://localhost:4222", "log", "logQueue", true)
	if err != nil {
		log.Fatalf("Failed to initiate channel. Error: %v", err)
	}

	ilm, err := NewIndexLifecycleManager(cfg.IndexName, cfg.RetentionDays)
	if err != nil {
		log.Fatalf("Failed to initiate index. Error: %v", err)
	}

	processor := NewLogProcessor(logQueue, ilm)

	cfg = Config{
		IndexName:     "INDEX_PREFIX",
		RetentionDays: 12 * 24 * time.Hour,
		ShutdownTimer: 5 * time.Second,
		Port:          "8080",
	}

	for _, opt := range opts {
		opt(cfg)
	}

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

func getEnvDefault(key, defaultValue string) string {
	if value, exists := os.LookupEnv(key); exists {
		return value
	}
	return defaultValue
}

func Run(cmd *cobra.Command, args []string) {
	port, _ := cmd.Flags().GetInt("port")
	portString := fmt.Sprintf("%d",port)

	cfg := Config{
		IndexName:     getEnvDefault("INDEX_PREFIX", "index-storage/index"),
		RetentionDays: 12 * 24 * time.Hour,
		ShutdownTimer: 5 * time.Second,
		Port:          portString,
	}

	server := NewServer(cfg)

	if err := server.Start(); err != nil {
		log.Fatalf("Failed to start app: %v", err)
	}

	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)

	//Catch kill signal for graceful shutdown
	sig := <-sigChan
	log.Printf("Caught signal: %v", sig)

	ctx, cancel := context.WithTimeout(context.Background(), cfg.ShutdownTimer)
	defer cancel()

	if err := server.Shutdown(ctx); err != nil {
		log.Fatalf("Failed to shutdown gracefully: %v", err)
	}

	log.Println("All log processed. Shutting down . . . ")
}
