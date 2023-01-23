package fuzzer

type Event struct {
	Type        string
	Description string
	Value       interface{}
}

const (
	EventTypeProgress   = "progress"
	EventTypeThroughput = "throughput"
	EventTypeError      = "error"
)
