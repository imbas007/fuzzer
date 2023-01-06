package fuzzer

import (
	"fuzzer/src/logger"
	"fuzzer/src/request"
	"strings"
	"time"

	"go.uber.org/zap"
)

func (f *Fuzzer) Worker(id int) {
	defer func() {
		f.mutex.Lock()
		defer f.mutex.Unlock()

		logger.Log.Warn("shutting down worker",
			zap.Int("id", id),
			zap.Int("totalWorkers", f.totalWorkers),
		)

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

		var (
			j job
		)

		select {
		case j = <-f.jobs:
		case <-time.After(3 * time.Second):
			continue
		}

		// logger.Log.Debug("received new job!",
		// 	zap.String("url", j.URL),
		// )

		res, statusCode, location, err := request.Do(j.URL, f.Method, nil)
		if err != nil {
			f.mutexStats.Lock()
			f.stats.Errors += 1
			f.mutexStats.Unlock()
		}

		lines := strings.Count(string(res), "\n")
		words := strings.Count(string(res), " ")

		redirectLocation := ""
		if location != j.URL {
			redirectLocation = location
		}

		size := len(res)

		f.mutexStats.Lock()
		f.stats.Processed += 1
		f.mutexStats.Unlock()

		if !f.filterResult(lines, words, size, statusCode) {
			continue
		}

		f.mutexStats.Lock()
		f.stats.Saved += 1
		f.mutexStats.Unlock()

		f.results <- result{
			RedirectLocation: redirectLocation,
			URL:              j.URL,
			Size:             size,
			Lines:            lines,
			StatusCode:       statusCode,
			Words:            words,
		}
	}
}
