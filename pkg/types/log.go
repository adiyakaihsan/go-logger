package types

import (
	"time"
)

type LogFormat struct {
	Timestamp time.Time `json:"timestamp"`
	Level     string    `json:"level"`
	Message   string    `json:"message"`
}

type SearchFormat struct {
	Query string `json:"query"`
}
