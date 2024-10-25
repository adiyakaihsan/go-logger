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
)

type App struct {
	queue     queue.ChannelQueue
	ilm       *IndexLifecycleManager
	processor *LogProcessor
}

type Config struct {
	IndexName     string
	ShutdownTimer time.Duration
	Port          string
}

func NewApp(cfg Config) (*App, error) {
	logQueue := queue.NewChannelQueue()

	ilm, err := NewIndexLifecycleManager(cfg.IndexName)
	if err != nil {
		log.Fatalf("Failed to initiate index. Error: %v", err)
	}

	processor := NewLogProcessor(*logQueue, ilm)

	app := &App{
		queue:     *logQueue,
		ilm:       ilm,
		processor: processor,
	}

	return app, nil
}

func (a *App) Start() error {
	a.ilm.StartScheduler()

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

func Run() {
	cfg := Config{
		IndexName:     "index",
		ShutdownTimer: 5 * time.Second,
		Port:          "8081",
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
