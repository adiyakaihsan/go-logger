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
	server    *Server
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

	server := NewServer(cfg.Port)

	processor := NewLogProcessor(*logQueue, ilm)

	app := &App{
		queue:     *logQueue,
		ilm:       ilm,
		processor: processor,
		server:    server,
	}
	app.registerRoutes()

	return app, nil
}

func (a *App) registerRoutes() {
	a.server.router.POST("/api/v1/log/ingest", a.ingester)
	a.server.router.POST("/api/v1/log/search", a.search)
	a.server.router.DELETE("/api/v1/log/delete", a.delete)
}

func (a *App) Start() error {
	a.ilm.StartScheduler()

	if err := a.processor.Start(); err != nil {
		return fmt.Errorf("failed to start log processor: %w", err)
	}

	if err := a.server.Start(); err != nil {
		return fmt.Errorf("failed to start server: %w", err)
	}
	return nil
}

func (a *App) Shutdown(ctx context.Context) error {
	if err := a.server.Shutdown(ctx); err != nil {
		return fmt.Errorf("server shutdown failed: %w", err)
	}

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

	application, err := NewApp(cfg)
	if err != nil {
		log.Fatalf("Failed to initialize app: %v", err)
	}

	if err := application.Start(); err != nil {
		log.Fatalf("Failed to start app: %v", err)
	}

	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)

	//Catch kill signal for graceful shutdown
	sig := <-sigChan
	log.Printf("Caught signal: %v", sig)

	ctx, cancel := context.WithTimeout(context.Background(), cfg.ShutdownTimer)
	defer cancel()

	if err := application.Shutdown(ctx); err != nil {
		log.Fatalf("Failed to shutdown gracefully: %v", err)
	}

	log.Println("All log processed. Shutting down . . . ")

}
