package main

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"time"

	bleve "github.com/blevesearch/bleve/v2"
	"github.com/julienschmidt/httprouter"
)

type log_format struct {
	Timestamp time.Time `json:"timestamp"`
	Level     string    `json:"level"`
	Message   string    `json:"message"`
}

type search_format struct {
	Query string `json:"query"`
}
type App struct {
	index bleve.Index
}

func main() {
	router := httprouter.New()

	server := &http.Server{
		Addr: ":8081", Handler: router,
	}

	// mapping := bleve.NewIndexMapping()

	index, err := isIndexExists("index.bleve2")
	if err != nil {
		log.Fatal("Cannot create new Index")
	}

	app := App{}

	app.index = index
	defer app.index.Close()
	// index some data

	router.POST("/v1/api/log/ingest", app.ingester)
	router.POST("/v1/api/log/search", app.search)

	log.Println("Starting server. . . .")

	if err := server.ListenAndServe(); err != nil {
		log.Fatalf("Cannot start http server. Error: %v", err)
	}

}

func (app App) ingester(w http.ResponseWriter, r *http.Request, _ httprouter.Params) {
	var logs log_format

	if err := json.NewDecoder(r.Body).Decode(&logs); err != nil {
		log.Printf("Cannot decode log. Error: %v", err)
	}
	if err := app.index.Index("id", logs); err != nil {
		log.Println("Cannot index data")
	}

	w.Write([]byte("OK"))
}

func (app App) search(w http.ResponseWriter, r *http.Request, _ httprouter.Params) {
	var search_query search_format

	if err := json.NewDecoder(r.Body).Decode(&search_query); err != nil {
		log.Printf("Cannot decode log. Error: %v", err)
	}

	query := bleve.NewMatchQuery(search_query.Query)
	search := bleve.NewSearchRequest(query)
	searchResults, err := app.index.Search(search)
	if err != nil {
		fmt.Println(err)
		return
	}

	resultJSON, err := json.Marshal(searchResults)
	if err != nil {
		http.Error(w, "Failed to marshal search results", http.StatusInternalServerError)
		return
	}

	log.Print(searchResults)

	w.Header().Set("Content-Type", "application/json")

	w.Write(resultJSON)
}

func isIndexExists(indexPath string) (bleve.Index, error) {
	var index bleve.Index
	// Check if the index already exists
	if _, err := os.Stat(indexPath); os.IsNotExist(err) {
		// Index doesn't exist, so create a new one
		log.Println("Index does not exist, creating new index...")
		mapping := bleve.NewIndexMapping()
		index, err = bleve.New(indexPath, mapping)
		if err != nil {
			log.Fatalf("Cannot create new index: %v", err)
			return nil, err
		}
		// defer index.Close()
		log.Println("Index created successfully.")
	} else {
		// Index exists, open it
		log.Println("Index exists, opening...")
		index, err = bleve.Open(indexPath)
		if err != nil {
			log.Fatalf("Cannot open existing index: %v", err)
			return nil, err
		}
		// defer index.Close()
		log.Println("Index opened successfully.")
	}
	return index, nil
}
