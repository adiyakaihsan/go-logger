package app

import (
	"errors"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"regexp"
	"strconv"
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
	searchManager *SearchManager
	retentionDays time.Duration
}

type SearchManager struct {
	alias   bleve.IndexAlias
	indices map[string]bleve.Index
}

func NewIndexLifecycleManager(baseIndexName string, retentionDays time.Duration) (*IndexLifecycleManager, error) {
	//Index Alias used by search
	indexAlias := bleve.NewIndexAlias()

	sm := &SearchManager{
		alias:   indexAlias,
		indices: map[string]bleve.Index{},
	}
	ilm := &IndexLifecycleManager{
		indexSearch:   indexAlias,
		baseIndexName: baseIndexName,
		searchManager: sm,
		retentionDays: retentionDays,
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

func (ilm *IndexLifecycleManager) indexWithRetry(logs types.LogFormat) {
	var maxRetries = 3
	var retryInterval = 5 * time.Second

	log.Println("Indexing")
	id := logs.Timestamp.Format("20060102150405.000")
	for attempt := 1; attempt <= maxRetries; attempt++ {
		err := ilm.index.Index(id, logs)
		if err == nil {
			log.Printf("Index ID: %v", id)
			return
		}
		log.Printf("Cannot index data. Error: %v", err)
		time.Sleep(retryInterval)

	}
}

func (ilm *IndexLifecycleManager) getHourlyIndexName() string {
	currentHour := time.Now().Format("2006-01-02-15") // Year-Month-Day-Hour format
	return fmt.Sprintf("%s-%s.log", ilm.baseIndexName, currentHour)
}

func (ilm *IndexLifecycleManager) getActiveIndex() (bleve.Index, error) {
	var index bleve.Index

	indexPath := ilm.getHourlyIndexName()

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
	// Job for index rollover
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
	// Job for index retention lifecycle
	_, err = ilm.scheduler.NewJob(
		gocron.CronJob("0 * * * *", false),
		gocron.NewTask(ilm.indexCleanUp),
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
	ilm.searchManager.indices[newIndex.Name()] = newIndex

	log.Printf("Rolled over to new index: %s", newIndex.Name())

}

func isOlderThan(filename string, age time.Duration) (bool, error) {
	// Regular expression to extract date components
	re := regexp.MustCompile(`index-(\d{4})-(\d{2})-(\d{2})-\d{2}`)
	matches := re.FindStringSubmatch(filename)

	if len(matches) != 4 {
		return false, errors.New("invalid filename format")
	}

	// Parse year, month, and day from matches
	year, _ := strconv.Atoi(matches[1])
	month, _ := strconv.Atoi(matches[2])
	day, _ := strconv.Atoi(matches[3])

	// Create time.Time object for the extracted date
	fileDate := time.Date(year, time.Month(month), day, 0, 0, 0, 0, time.UTC)

	// Get current time
	now := time.Now().UTC()

	// Calculate the difference
	difference := now.Sub(fileDate)

	return difference > age, nil
}

func (ilm *IndexLifecycleManager) indexCleanUp() error {
	for _, index := range ilm.searchManager.indices {
		delete, err := isOlderThan(index.Name(), ilm.retentionDays)
		if err != nil {
			log.Printf("Cannot compare %v age. Error: %v", index.Name(), err)
			return err
		}
		if delete {
			log.Printf("Removing %v from index alias.", index.Name())
			ilm.indexSearch.Remove(index)

			log.Printf("Closing index %v", index.Name())
			index.Close()

			log.Printf("Removing %v index file from system.", index.Name())
			if err := os.RemoveAll(index.Name()); err != nil {
				log.Printf("Cannot remove delete %v. Error: %v", index.Name(), err)
				return err
			}
		}
	}
	return nil
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
		ilm.searchManager.indices[id.Name()] = id
		log.Printf("Added index: %v to Index Alias", id.Name())
	}
	return nil
}
