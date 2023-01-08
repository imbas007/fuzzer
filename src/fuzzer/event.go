package fuzzer

type Event struct {
	Type  string
	Value string
}

const (
	EventTypeProgress   = "progress"
	EventTypeThroughput = "throughput"
	EventTypeError      = "error"
)
