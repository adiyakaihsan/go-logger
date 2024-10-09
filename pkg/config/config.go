package config

import "time"

const (
	RetentionPeriod = time.Duration(22) * time.Hour

)

var NilTime = time.Time{}