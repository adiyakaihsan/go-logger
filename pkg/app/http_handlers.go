package app

import (
	"encoding/json"
	"log"
	"net/http"
	"time"

	"github.com/adiyakaihsan/go-logger/pkg/config"
	"github.com/adiyakaihsan/go-logger/pkg/types"
	"github.com/julienschmidt/httprouter"
)

func (app App) ingester(w http.ResponseWriter, r *http.Request, _ httprouter.Params) {
	var logs types.LogFormat

	if err := json.NewDecoder(r.Body).Decode(&logs); err != nil {
		log.Printf("Cannot decode log. Error: %v", err)
	}
	if err := app.queue.Enqueue(logs); err != nil {
		log.Printf("Cannot enqueue logs. Error: %v", err)
	}

	w.Write([]byte("OK"))
}

func (app App) search(w http.ResponseWriter, r *http.Request, _ httprouter.Params) {
	var search_query types.SearchFormat

	if err := json.NewDecoder(r.Body).Decode(&search_query); err != nil {
		log.Printf("Cannot decode log. Error: %v", err)
	}
	searchResults, err := app.searchWithQuery(search_query)
	if err != nil {
		log.Printf("Cannot search with Query: %v, Error: %v", search_query.Query, err)
		return
	}

	resultJSON, err := json.Marshal(searchResults)
	if err != nil {
		http.Error(w, "Failed to marshal search results", http.StatusInternalServerError)
		return
	}

	// log.Print(searchResults)

	w.Header().Set("Content-Type", "application/json")

	w.Write(resultJSON)
}

// TO DO: can be deleted once we implement retention with index removal
func (app App) delete(w http.ResponseWriter, r *http.Request, _ httprouter.Params) {
	retentionPeriod := time.Now().Add(-1 * config.RetentionPeriod)

	searchResults, err := app.searchWithRange(config.NilTime, retentionPeriod)
	if err != nil {
		return
	}
	if searchResults.Hits.Len() == 0 {
		log.Printf("No document found with specified criteria")
		return
	}
	for _, hit := range searchResults.Hits {
		if err := app.ilm.index.Delete(hit.ID); err != nil {
			log.Printf("Error deleting document ID: %v. Error: %v", hit.ID, err)
			return
		}
		log.Printf("Successfully delete document ID: %v", hit.ID)
	}

}
