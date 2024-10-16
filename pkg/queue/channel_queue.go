package queue

import (
	"errors"

	"github.com/adiyakaihsan/go-logger/pkg/types"
)

type ChannelQueue struct {
	logStream chan types.LogFormat
}


func NewChannelQueue() *ChannelQueue {
	return &ChannelQueue{
		logStream: make(chan types.LogFormat),
	}
}

func (cq *ChannelQueue) Enqueue(log types.LogFormat) error {
	cq.logStream <- log
	return nil
}

func (cq *ChannelQueue) Dequeue() (types.LogFormat, error) {
	log, ok := <- cq.logStream
	if !ok {
		return log, errors.New("Channel is closed")
	}
	return log, nil
}

func (cq *ChannelQueue) Close() {
	close(cq.logStream)
}