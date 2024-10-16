package config

import "time"

const (
	RetentionPeriod = time.Duration(220) * time.Hour //temporary adjustment, until implement index based retention
	// BaseIndexName = "index-2024-10-15-21.log"
	BaseIndexName = "index*.log"	

)

var NilTime = time.Time{}