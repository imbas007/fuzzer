package fuzzer

type Event struct {
	Type  string
	Value string
}

const (
	TypeProgress   = "progress"
	TypeThroughput = "throughput"
)
