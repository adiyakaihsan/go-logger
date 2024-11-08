package app

import (
	"context"
	"fmt"
	"log"
	"os"
	"os/signal"
	"sync"
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

func NewApp(cfg Config) (*App, error) {
	logQueue, err := queue.NewNatsQueue("nats://localhost:4222", "log", "logQueue", cfg.Port, true)
	if err != nil {
		log.Fatalf("Failed to initiate channel. Error: %v", err)
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

func Run(cmd *cobra.Command, args []string) {
	port, _ := cmd.Flags().GetInt("port")
	serverCount, _ := cmd.Flags().GetInt("server")

	indexPrefix, _ := cmd.Flags().GetString("index-prefix")

	var wg sync.WaitGroup

	// Channel to coordinate shutdown across all servers
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)

	// Create servers
	servers := make([]*Server, serverCount)

	for i := 0; i < serverCount; i++ {
		wg.Add(1)
		go func(index int) {
			defer wg.Done()
			log.Printf("increment: %v", i)
			incrementPort := port + index
			portString := fmt.Sprintf("%d", incrementPort)
			indexName := fmt.Sprintf("%s-%d/index", indexPrefix, index+1)

			cfg := Config{
				IndexName:     indexName,
				RetentionDays: 12 * 24 * time.Hour,
				ShutdownTimer: 5 * time.Second,
				Port:          portString,
			}

			server := NewServer(cfg)
			servers[index] = server

			if err := server.Start(); err != nil {
				log.Fatalf("Failed to start server %d: %v", index, err)
			}

		}(i)
	}
	//Catch kill signal for graceful shutdown
	sig := <-sigChan
	log.Printf("Caught signal: %v", sig)

	for i, server := range servers {
		if server != nil {
			ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
			defer cancel()

			if err := server.Shutdown(ctx); err != nil {
				log.Fatalf("Failed to shutdown server %d gracefully: %v", i, err)
			}
			cancel()
		}
	}

	wg.Wait()
	log.Println("All log processed. Shutting down . . . ")
}
