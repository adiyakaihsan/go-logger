package queue

import "github.com/adiyakaihsan/go-logger/pkg/types"

type Queue interface {
	Enqueue(log types.LogFormat) error
	Dequeue() (types.LogFormat, error)
	Close()
}