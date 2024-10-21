package app

import (
	"fmt"
	"log"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/adiyakaihsan/go-logger/pkg/types"
	bleve "github.com/blevesearch/bleve/v2"
	gocron "github.com/go-co-op/gocron/v2"
)

type IndexLifecycleManager struct {
	index         bleve.Index
	indexSearch   bleve.IndexAlias
	scheduler     gocron.Scheduler
	baseIndexName string
}

func NewIndexLifecycleManager(baseIndexName string) (*IndexLifecycleManager, error) {
	//Init Index
	// baseIndexName := "index"

	//Index Alias used by search
	indexAlias := bleve.NewIndexAlias()

	ilm := &IndexLifecycleManager{
		indexSearch:   indexAlias,
		baseIndexName: baseIndexName,
	}

	index, err := ilm.getActiveIndex()
	if err != nil {
		return nil, fmt.Errorf("failed to get active index: %w", err)
	}

	ilm.index = index

	ilm.indexSearch.Add(index)
	log.Printf("Added index: %v to Index Alias", index.Name())

	// go startHourlyIndexRollover(&app, "index")
	scheduler, err := gocron.NewScheduler()
	if err != nil {
		log.Fatal("Cannot create scheduler for ILM")
	}
	ilm.scheduler = scheduler

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

func (ilm *IndexLifecycleManager) hourlyIndexName() string {
	currentHour := time.Now().Format("2006-01-02-15") // Year-Month-Day-Hour format
	return fmt.Sprintf("%s-%s.log", ilm.baseIndexName, currentHour)
}

func (ilm *IndexLifecycleManager) getActiveIndex() (bleve.Index, error) {
	var index bleve.Index

	indexPath := ilm.hourlyIndexName()

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

func (ilm *IndexLifecycleManager) StartScheduler() error {
	_, err := ilm.scheduler.NewJob(
		gocron.CronJob("0 * * * *", false),
		gocron.NewTask(
			ilm.indexRollover,
			ilm.baseIndexName,
		),
	)
	if err != nil {
		return fmt.Errorf("error scheduling job: %w", err)
	}
	ilm.scheduler.Start()
	log.Printf("Started ILM scheduler.")
	return nil
}

func (ilm *IndexLifecycleManager) StopScheduler() {
	ilm.scheduler.Shutdown()
}

func (ilm *IndexLifecycleManager) indexRollover(baseIndexName string) {
	// Close the old index and create a new one for the new hour
	if err := ilm.index.Close(); err != nil {
		log.Printf("Cannot close index. Error: %v", err)
	}

	newIndex, err := ilm.getActiveIndex()
	if err != nil {
		log.Printf("Cannot create new Index. Error: %v", err)
	}
	ilm.index = newIndex
	//update indexAlias for search
	ilm.indexSearch.Add(newIndex)

	log.Printf("Rolled over to new index: %s", newIndex.Name())

}

func (ilm *IndexLifecycleManager) findAllIndexes() []string {
	var indexList []string
	matches, err := filepath.Glob(fmt.Sprintf("%v*.log", ilm.baseIndexName))
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
	indexList := ilm.findAllIndexes()

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
