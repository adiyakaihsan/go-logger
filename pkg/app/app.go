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

	"github.com/adiyakaihsan/go-logger/pkg/queue"
	bleve "github.com/blevesearch/bleve/v2"
	"github.com/julienschmidt/httprouter"
)

type App struct {
	index bleve.Index
	queue queue.ChannelQueue
}

func Run() {
	var wg sync.WaitGroup

	logQueue := queue.NewChannelQueue()

	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)

	router := httprouter.New()

	server := &http.Server{
		Addr: ":8081", Handler: router,
	}

	index, err := checkIndex("index.log")
	if err != nil {
		log.Fatal("Cannot create new Index")
	}

	app := App{}

	app.index = index
	app.queue = *logQueue
	defer app.index.Close()

	router.POST("/api/v1/log/ingest", app.ingester)
	router.POST("/api/v1/log/search", app.search)
	router.DELETE("/api/v1/log/delete", app.delete)

	log.Println("Starting server on port 8081. . . .")

	go func() {
		if err := server.ListenAndServe(); err != nil {
			// log.Fatalf("Cannot start http server. Error: %v", err)
			return
		}
	}()

	go func() {
		for {
			logItem, err := logQueue.Dequeue()
			if err != nil {
				log.Printf("Stopped retrieving from queue. Info: %v", err)
				return
			}
			wg.Add(1)
			go func() {
				defer wg.Done()
				app.indexer(logItem)
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

	logQueue.Close()

	wg.Wait()

	log.Println("All log processed. Shutting down . . . ")

}
