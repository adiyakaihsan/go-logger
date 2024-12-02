package app

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"time"

	"github.com/julienschmidt/httprouter"
)

type Server struct {
	server *http.Server
	router *httprouter.Router
	app    *App
}

func NewServer(cfg Config) *Server {
	router := httprouter.New()

	opts := []Option{
		IndexName("index-storage/index"),
		RetentionDays(12 * 24 * time.Hour),
		ShutdownTimer(5 * time.Second),
		Port(cfg.Port),
	}

	app, err := NewApp(cfg, opts...)
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
