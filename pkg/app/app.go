package app

import (
	"context"
	"log"
	"net/http"
	"os"
	"os/signal"
	"sync"
	"syscall"
	"time"

	"github.com/adiyakaihsan/go-logger/pkg/types"
	bleve "github.com/blevesearch/bleve/v2"
	"github.com/julienschmidt/httprouter"
)

var log_stream = make(chan types.Log_format)

type App struct {
	index bleve.Index
}

func Run() {
	var wg sync.WaitGroup

	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)

	router := httprouter.New()

	server := &http.Server{
		Addr: ":8081", Handler: router,
	}

	index, err := isIndexExists("index.log")
	if err != nil {
		log.Fatal("Cannot create new Index")
	}

	app := App{}

	app.index = index
	defer app.index.Close()

	router.POST("/api/v1/log/ingest", app.ingester)
	router.POST("/api/v1/log/search", app.search)

	log.Println("Starting server on port 8081. . . .")

	go func() {
		if err := server.ListenAndServe(); err != nil {
			// log.Fatalf("Cannot start http server. Error: %v", err)
			return
		}
	}()

	go func() {
		for log := range log_stream {
			wg.Add(1)
			go func() {
				defer wg.Done()
				app.indexer(log)
			}()
		}
	}()

	sig := <-sigChan
	log.Printf("Caught signal: %v", sig)

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := server.Shutdown(ctx); err != nil {
		log.Fatalf("Failed shutting down HTTP Server. Error: %v", err)
	}

	close(log_stream)

	wg.Wait()

	log.Println("All log processed. Shutting down . . . ")

}
