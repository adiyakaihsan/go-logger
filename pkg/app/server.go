package app

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/julienschmidt/httprouter"
	"github.com/spf13/cobra"
)

type Server struct {
	server *http.Server
	router *httprouter.Router
	app    *App
}

func NewServer(opts ...Option) *Server {
	router := httprouter.New()

	cfg := applyOptions(defaultConfig, opts...)

	app, err := NewApp(opts...)
	if err != nil {
		log.Fatalf("Cannot instantiate App. Error: %v", err)
	}
	server := &Server{
		router: router,
		server: &http.Server{
			Addr:    fmt.Sprintf(":%s", cfg.Port),
			Handler: router,
		},
		app: app,
	}
	server.registerRoutes()
	return server
}

func (s *Server) registerRoutes() {
	s.router.POST("/api/v1/log/ingest", s.app.ingester)
	s.router.POST("/api/v1/log/search", s.app.search)
}

func (s *Server) Start() error {
	go func() {
		log.Printf("Starting server on: localhost%v", s.server.Addr)
		if err := s.server.ListenAndServe(); err != nil {
			log.Printf("HTTP server error: %v", err)
		}
	}()

	//start App
	s.app.Start()
	return nil
}

func (s *Server) Shutdown(ctx context.Context) error {
	// shutdown App
	s.app.Shutdown()
	return s.server.Shutdown(ctx)
}

func Run(cmd *cobra.Command, args []string) {
	port, _ := cmd.Flags().GetInt("port")
	indexName, _ := cmd.Flags().GetString("index")
	portString := fmt.Sprintf("%d", port)

	opts := []Option{
		IndexName(indexName),
		RetentionDays(12 * 24 * time.Hour),
		ShutdownTimer(5 * time.Second),
		Port(portString),
	}

	server := NewServer(opts...)

	if err := server.Start(); err != nil {
		log.Fatalf("Failed to start app: %v", err)
	}

	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)

	//Catch kill signal for graceful shutdown
	sig := <-sigChan
	log.Printf("Caught signal: %v", sig)

	ctx, cancel := context.WithTimeout(context.Background(), defaultConfig.ShutdownTimer)
	defer cancel()

	if err := server.Shutdown(ctx); err != nil {
		log.Fatalf("Failed to shutdown gracefully: %v", err)
	}

	log.Println("All log processed. Shutting down . . . ")
}
