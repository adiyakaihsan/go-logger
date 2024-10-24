package app

import (
	"context"
	"fmt"
	"log"
	"net/http"

	"github.com/julienschmidt/httprouter"
)

type Server struct {
	server *http.Server
	router *httprouter.Router
}

func NewServer(port string) *Server {
	router := httprouter.New()
	return &Server{
		router: router,
		server: &http.Server{
			Addr:    fmt.Sprintf(":%s", port),
			Handler: router,
		},
	}
}

func (s *Server) Start() error {
	go func() {
		log.Println("Starting server . . . .")
		if err := s.server.ListenAndServe(); err != nil {
			log.Printf("HTTP server error: %v", err)
		}
	}()
	return nil
}

func (s *Server) Shutdown(ctx context.Context) error {
	return s.server.Shutdown(ctx)
}
