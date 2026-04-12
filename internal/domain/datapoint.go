package domain

import "time"

// Datapoint is the normalized unit of data produced by a Prometheus query.
type Datapoint struct {
	Labels    map[string]string
	Value     float64
	Timestamp time.Time
}
