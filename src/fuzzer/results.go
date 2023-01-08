package fuzzer

import (
	"encoding/json"
	"os"
	"strconv"
	"strings"
	"time"

	"go.uber.org/zap"
)

type Result struct {
	RedirectLocation string `json:"redirectLocation"`
	URL              string `json:"url"`
	Size             int    `json:"size"`
	Lines            int    `json:"lines"`
	StatusCode       int    `json:"statusCode"`
	Words            int    `json:"words"`
}

// saveResults is worker which saves results one by one in jsonl format
func (f *Fuzzer) saveResults() {
	fd, err := os.OpenFile(f.OutFile, os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		f.Log.Error("error in opening out file",
			zap.Error(err),
		)
		return
	}
	defer fd.Close()

	defer func() {
		f.mutex.Lock()
		defer f.mutex.Unlock()

		if !f.IsSilent {
			f.Log.Debug("shutting down results worker",
				zap.Int("totalWorkers", f.totalWorkers),
			)
		}

		f.totalWorkers--
	}()

	shouldWork := true

	// monitoring for control exit
	go func() {
		<-f.control

		f.mutex.Lock()
		defer f.mutex.Unlock()

		shouldWork = false
	}()

	for {
		f.mutex.Lock()
		if !shouldWork {
			f.mutex.Unlock()
			return
		}
		f.mutex.Unlock()

		var r Result
		select {
		case r = <-f.results:
		case <-time.After(3 * time.Second):
			continue
		}

		raw, _ := json.Marshal(r)
		fd.WriteString(string(raw) + "\n")
		fd.Sync()
	}
}

func GetUniqueNumbers(input, delimiter string) (res []int) {
	res = make([]int, 0)

	tmp := strings.Split(input, delimiter)
	unq := make(map[int]bool, 0)
	for _, t := range tmp {
		val, err := strconv.Atoi(t)
		if err != nil {
			continue
		}

		unq[val] = true
	}

	for key := range unq {
		res = append(res, key)
	}

	return
}
