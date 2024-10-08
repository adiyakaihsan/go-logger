package app

import (
	"log"
	"os"

	"github.com/adiyakaihsan/go-logger/pkg/types"
	bleve "github.com/blevesearch/bleve/v2"
)

func (app App) indexer(logs types.Log_format) {
	log.Println("Indexing")
	id := logs.Timestamp.Format("20060102150405.000")
	if err := app.index.Index(id, logs); err != nil {
		log.Println("Cannot index data")
	}
	log.Printf("Index ID: %v", id)
}

func checkIndex(indexPath string) (bleve.Index, error) {
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
