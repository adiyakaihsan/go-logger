package app

import (
	"log"
	"time"

	"github.com/adiyakaihsan/go-logger/pkg/types"
	bleve "github.com/blevesearch/bleve/v2"
	"github.com/blevesearch/bleve/v2/search"
)

func (app App) searchWithRange(startTime time.Time, endTime time.Time) (*bleve.SearchResult, error) {
	rangeQuery := bleve.NewDateRangeQuery(startTime, endTime)
	rangeQuery.SetField("timestamp")

	searchRequest := bleve.NewSearchRequest(rangeQuery)
	searchRequest.Fields = []string{"timestamp", "level", "message"}
	searchResults, err := app.index.Search(searchRequest)
	if err != nil {
		return nil, err
	}
	return searchResults, nil

}
// TO DO: can be deleted once we implement retention with index removal
func (app App) searchWithQuery(searchQuery types.SearchFormat) (search.DocumentMatchCollection, error) {
	query := bleve.NewQueryStringQuery(searchQuery.Query)
	searchRequest := bleve.NewSearchRequest(query)

	searchRequest.Fields = []string{"timestamp", "level", "message"}

	//search Index Alias
	indexAliasSearchResults, err := app.indexSearch.Search(searchRequest)
	if err != nil {
		return nil, err
	}
	// search current active index
	activeIndexSearchResults, err := app.index.Search(searchRequest)
	if err != nil {
		return nil, err
	}
	combinedHits := append(activeIndexSearchResults.Hits, indexAliasSearchResults.Hits...)
	log.Printf("Found %v document match!", combinedHits.Len())
	return combinedHits, err
}
