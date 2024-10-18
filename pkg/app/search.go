package app

import (
	"log"
	"time"

	"github.com/adiyakaihsan/go-logger/pkg/types"
	bleve "github.com/blevesearch/bleve/v2"
)

func (app App) searchWithRange(startTime time.Time, endTime time.Time) (*bleve.SearchResult, error) {
	rangeQuery := bleve.NewDateRangeQuery(startTime, endTime)
	rangeQuery.SetField("timestamp")

	searchRequest := bleve.NewSearchRequest(rangeQuery)
	searchRequest.Fields = []string{"timestamp", "level", "message"}
	searchResults, err := app.ilm.indexSearch.Search(searchRequest)
	if err != nil {
		return nil, err
	}
	return searchResults, nil

}

func (app App) searchWithQuery(searchQuery types.SearchFormat) (*bleve.SearchResult, error) {
	query := bleve.NewQueryStringQuery(searchQuery.Query)
	searchRequest := bleve.NewSearchRequest(query)

	searchRequest.Fields = []string{"timestamp", "level", "message"}

	//search Index Alias indexSearch
	searchResults, err := app.ilm.indexSearch.Search(searchRequest)
	if err != nil {
		return nil, err
	}

	// combinedHits := append(activeIndexSearchResults.Hits, indexAliasSearchResults.Hits...)
	log.Printf("Found %v document match!", searchResults.Hits.Len())
	return searchResults, err
}
