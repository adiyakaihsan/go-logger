package types

import "time"

type Log_format struct {
	Timestamp time.Time `json:"timestamp"`
	Level     string    `json:"level"`
	Message   string    `json:"message"`
}

type Search_format struct {
	Query string `json:"query"`
}
