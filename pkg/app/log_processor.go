package app

import (
	"log"
	"sync"

	"github.com/adiyakaihsan/go-logger/pkg/queue"
)

type LogProcessor struct {
	queue queue.Queue
	ilm   *IndexLifecycleManager
	wg    sync.WaitGroup
}

func NewLogProcessor(queue queue.Queue, ilm *IndexLifecycleManager) *LogProcessor {
	return &LogProcessor{
		queue: queue,
		ilm:   ilm,
	}
}

func (lp *LogProcessor) Start() error {
	go lp.processLogs()
	log.Println("Log Processor Started.")
	return nil
}

func (lp *LogProcessor) Shutdown() error {

	lp.wg.Wait()
	log.Println("Log Processor Shutdown.")

	return nil
}

func (lp *LogProcessor) processLogs() {
	go func() {
		for {
			logItem, err := lp.queue.Dequeue()
			if err != nil {
				log.Printf("Stopped retrieving from queue. Info: %v", err)
				return
			}

			lp.wg.Add(1)
			go func() {
				defer lp.wg.Done()
				lp.ilm.indexWithRetry(logItem)
			}()
		}
	}()
}
