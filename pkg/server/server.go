package server

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/adiyakaihsan/go-logger/pkg/app"
	"github.com/julienschmidt/httprouter"
	"github.com/spf13/cobra"
)

type Server struct {
	server *http.Server
	router *httprouter.Router
	app    *app.App
}

func NewServer(serverOpts ...ServerOption) *Server {


	router := httprouter.New()

	cfg := applyOptions(defaultConfig, serverOpts...)

	indexName := cfg.IndexName

	appOpts := []app.AppOption{
		app.IndexName(indexName),
		app.RetentionDays(12 * 24 * time.Hour),
		app.ShutdownTimer(5 * time.Second),
	}

	app, err := app.NewApp(appOpts...)
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
	s.router.POST("/api/v1/log/ingest", s.app.Ingester)
	s.router.POST("/api/v1/log/search", s.app.Search)
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

	opts := []ServerOption{
		ShutdownTimer(5 * time.Second),
		Port(portString),
		IndexName(indexName),
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
