package queue

import (
	"errors"

	"github.com/adiyakaihsan/go-logger/pkg/types"
)

type ChannelQueue struct {
	logStream chan types.Log_format
}


func NewChannelQueue() *ChannelQueue {
	return &ChannelQueue{
		logStream: make(chan types.Log_format),
	}
}

func (cq *ChannelQueue) Enqueue(log types.Log_format) error {
	cq.logStream <- log
	return nil
}

func (cq *ChannelQueue) Dequeue() (types.Log_format, error) {
	log, ok := <- cq.logStream
	if !ok {
		return log, errors.New("Channel is closed")
	}
	return log, nil
}

func (cq *ChannelQueue) Close() {
	close(cq.logStream)
}