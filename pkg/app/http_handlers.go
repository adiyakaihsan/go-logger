package app

import (
	"encoding/json"
	"log"
	"net/http"

	"github.com/adiyakaihsan/go-logger/pkg/types"
	"github.com/julienschmidt/httprouter"
)

func (app App) Ingester(w http.ResponseWriter, r *http.Request, _ httprouter.Params) {
	var logs types.LogFormat

	if err := json.NewDecoder(r.Body).Decode(&logs); err != nil {
		log.Printf("Cannot decode log. Error: %v", err)
	}
	if err := app.queue.Enqueue(logs); err != nil {
		log.Printf("Cannot enqueue logs. Error: %v", err)
	}

	w.Write([]byte("OK"))
}

func (app App) Search(w http.ResponseWriter, r *http.Request, _ httprouter.Params) {
	var searchQuery types.SearchFormat

	if err := json.NewDecoder(r.Body).Decode(&searchQuery); err != nil {
		log.Printf("Cannot decode log. Error: %v", err)
	}
	searchResults, err := app.searchWithQuery(searchQuery)
	if err != nil {
		log.Printf("Cannot search with Query: %v, Error: %v", searchQuery.Query, err)
		return
	}

	resultJSON, err := json.Marshal(searchResults)
	if err != nil {
		http.Error(w, "Failed to marshal search results", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")

	w.Write(resultJSON)
}
