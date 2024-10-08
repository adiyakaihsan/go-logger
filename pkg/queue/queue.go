package queue

import "github.com/adiyakaihsan/go-logger/pkg/types"

type Queue interface {
	Enqueue(log types.Log_format) error
	Dequeue() (types.Log_format, error)
	Close()
}