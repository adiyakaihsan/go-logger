package app

import (
	"time"

	"github.com/adiyakaihsan/go-logger/pkg/types"
	bleve "github.com/blevesearch/bleve/v2"
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

func (app App) searchWithQuery(searchQuery types.Search_format) (*bleve.SearchResult, error) {
	query := bleve.NewQueryStringQuery(searchQuery.Query)
	searchRequest := bleve.NewSearchRequest(query)

	searchRequest.Fields = []string{"timestamp", "level", "message"}

	searchResults, err := app.index.Search(searchRequest)
	if err != nil {
		return nil, err
	}

	return searchResults, err
}
