package metrics

import "time"

type Result uint8

const (
	Allowed Result = iota
	Denied
	Errored
	NumResults
)

type Recorder interface {
	ObserveDecision(r Result, elapsed time.Duration)
}
