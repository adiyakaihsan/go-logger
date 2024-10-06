package app

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"

	"github.com/adiyakaihsan/go-logger/pkg/types"
	bleve "github.com/blevesearch/bleve/v2"
	"github.com/julienschmidt/httprouter"
)

func (app App) ingester(w http.ResponseWriter, r *http.Request, _ httprouter.Params) {
	var logs types.Log_format

	if err := json.NewDecoder(r.Body).Decode(&logs); err != nil {
		log.Printf("Cannot decode log. Error: %v", err)
	}

	log_stream <- logs

	w.Write([]byte("OK"))
}

func (app App) search(w http.ResponseWriter, r *http.Request, _ httprouter.Params) {
	var search_query types.Search_format

	if err := json.NewDecoder(r.Body).Decode(&search_query); err != nil {
		log.Printf("Cannot decode log. Error: %v", err)
	}

	query := bleve.NewQueryStringQuery(search_query.Query)
	searchRequest := bleve.NewSearchRequest(query)

	searchRequest.Fields = []string{"timestamp", "level", "message"}

	searchResults, err := app.index.Search(searchRequest)
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
