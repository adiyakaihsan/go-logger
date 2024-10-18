package app

import (
	"fmt"
	"log"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/adiyakaihsan/go-logger/pkg/config"
	"github.com/adiyakaihsan/go-logger/pkg/types"
	bleve "github.com/blevesearch/bleve/v2"
)

type IndexLifecycleManager struct {
	index       bleve.Index
	indexSearch bleve.IndexAlias
}

func NewIndexLifecycleManager() (*IndexLifecycleManager, error) {
	//Init Index
	baseIndexName := "index"
	indexPath := hourlyIndexName(baseIndexName)

	index, err := (&IndexLifecycleManager{}).getActiveIndex(indexPath)
	if err != nil {
		return nil, fmt.Errorf("failed to get active index: %w", err)
	}
	//Index Alias used by search
	indexAlias := bleve.NewIndexAlias()

	indexAlias.Add(index)
	log.Printf("Added index: %v to Index Alias", index.Name())

	ilm := &IndexLifecycleManager{
		index:       index,
		indexSearch: indexAlias,
	}

	if err := ilm.getIndexAlias(); err != nil {
		log.Printf("failed to get index alias: %v", err)
	}

	return ilm, nil

}

func (ilm *IndexLifecycleManager) indexer(logs types.LogFormat) {
	log.Println("Indexing")
	id := logs.Timestamp.Format("20060102150405.000")
	if err := ilm.index.Index(id, logs); err != nil {
		log.Println("Cannot index data")
	}
	log.Printf("Index ID: %v", id)
}

func hourlyIndexName(baseName string) string {
	currentHour := time.Now().Format("2006-01-02-15") // Year-Month-Day-Hour format
	return fmt.Sprintf("%s-%s.log", baseName, currentHour)
}

func (ilm *IndexLifecycleManager) getActiveIndex(indexPath string) (bleve.Index, error) {
	var index bleve.Index
	// Check if the index already exists
	if _, err := os.Stat(indexPath); os.IsNotExist(err) {
		// Index doesn't exist, so create a new one
		log.Println("Index does not exist, creating new index...")
		mapping := bleve.NewIndexMapping()
		index, err = bleve.New(indexPath, mapping)
		if err != nil {
			log.Printf("Cannot create new index: %v", err)
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
	log.Printf("Current active index: %v", index.Name())
	return index, nil
}

func (ilm *IndexLifecycleManager) indexRollover(app *App, baseIndexName string) {
	// Close the old index and create a new one for the new hour
	if err := app.ilm.index.Close(); err != nil {
		log.Printf("Cannot close index. Error: %v", err)
	}

	newIndexName := hourlyIndexName(baseIndexName)
	newIndex, err := app.ilm.getActiveIndex(newIndexName)
	if err != nil {
		log.Printf("Cannot create new Index for %s", newIndexName)
	}
	app.ilm.index = newIndex
	//update indexAlias for search
	ilm.indexSearch.Add(newIndex)

	log.Printf("Rolled over to new index: %s", newIndexName)

}

func findAllIndexes() []string {
	var indexList []string
	matches, err := filepath.Glob(config.BaseIndexName)
	if err != nil {
		log.Fatal(err)
		return indexList
	}
	indexList = append(indexList, matches...)

	return indexList
}

func openIndexWithTimeout(indexPath string, timeout time.Duration) (bleve.Index, error) {
	var index bleve.Index
	var err error
	done := make(chan bool)

	go func() {
		index, err = bleve.Open(indexPath)
		close(done)
	}()

	select {
	case <-done:
		if err != nil && strings.Contains(err.Error(), "index is already open") {
			return nil, fmt.Errorf("index is busy: %v", err)
		}
		return index, err
	case <-time.After(timeout):
		return nil, fmt.Errorf("time out opening index. index could be already opened")
	}
}

func (ilm IndexLifecycleManager) getIndexAlias() error {
	indexList := findAllIndexes()

	for _, index := range indexList {
		id, err := openIndexWithTimeout(index, 5*time.Second)
		// log.Printf("var %v", id)
		if err != nil {
			log.Printf("Cannot open index. Error: %v", err)
			continue
		}
		ilm.indexSearch.Add(id)
		log.Printf("Added index: %v to Index Alias", id.Name())
	}
	return nil
}
