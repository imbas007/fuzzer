package fuzzer

import "errors"

var (
	ErrMaxRuntime = errors.New("command reached maximum runtime")
)
