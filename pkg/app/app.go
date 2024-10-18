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
	gocron "github.com/go-co-op/gocron/v2"
	"github.com/julienschmidt/httprouter"
)

type App struct {
	queue queue.ChannelQueue
	ilm   *IndexLifecycleManager
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

	ilm, err := NewIndexLifecycleManager()
	if err != nil {
		log.Fatalf("Failed to initiate index. Error: %v", err)
	}

	app := App{
		queue: *logQueue,
		ilm:   ilm,
	}

	//Index rollover goroutine
	// go startHourlyIndexRollover(&app, "index")
	schedule, err := gocron.NewScheduler()
	if err != nil {
		log.Fatal("Cannot create scheduler for ILM")
	}

	_, err = schedule.NewJob(
		gocron.CronJob("0 * * * *", false),
		gocron.NewTask(
			app.ilm.indexRollover,
			&app,
			"index",
		),
	)
	if err != nil {
		log.Printf("Error scheduling job. Error: %v", err)
	}
	schedule.Start()
	log.Printf("Started ILM scheduler.")

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
				app.ilm.indexer(logItem)
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

	if err := schedule.Shutdown(); err != nil {
		log.Fatalf("Failed shutting down scheduler")
	}

	logQueue.Close()

	wg.Wait()

	log.Println("All log processed. Shutting down . . . ")

}
