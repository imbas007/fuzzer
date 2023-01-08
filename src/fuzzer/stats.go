package fuzzer

import (
	"fmt"
	"time"

	"github.com/dpanic/fuzzer/src/logger"

	"go.uber.org/zap"
)

//go:generate easytags $GOFILE json:camel

type stats struct {
	LastCalculated time.Time `json:"lastCalculated"`
	LastProcessed  int       `json:"lastProcessed"`
	ReqPerSec      float64   `json:"reqPerSec"`
	Total          int       `json:"total"`
	Processed      int       `json:"processed"`
	Errors         int       `json:"errors"`
	Saved          int       `json:"saved"`
}

func (f *Fuzzer) calculateStats() {
	duration := time.Since(f.stats.LastCalculated)
	seconds := duration.Seconds()
	processed := f.stats.Processed - f.stats.LastProcessed
	reqPerSec := float64(processed) / float64(seconds)

	// add throughput
	event := Event{
		Type:  TypeThroughput,
		Value: fmt.Sprintf("%.2f / sec", reqPerSec),
	}
	select {
	case f.Events <- event:
	case <-time.After(10 * time.Millisecond):
	}

	// add progress
	event = Event{
		Type:  TypeProgress,
		Value: fmt.Sprintf("%d / %d", f.stats.Processed, f.stats.Total),
	}
	select {
	case f.Events <- event:
	case <-time.After(10 * time.Millisecond):
	}

	f.stats.ReqPerSec = reqPerSec
	f.stats.LastCalculated = time.Now()
	f.stats.LastProcessed = f.stats.Processed
}

func (f *Fuzzer) PrintStats() {
	if !f.IsSilent {
		logger.Log.Info("stats",
			zap.String("url", f.URL),
			zap.String("proxyURL", f.ProxyURL),
			zap.Int("total", f.stats.Total),
			zap.Int("processed", f.stats.Processed),
			zap.Int("left", f.stats.Total-f.stats.Processed),
			zap.Int("saved", f.stats.Saved),
			zap.Int("errors", f.stats.Errors),
			zap.Float64("req/s", f.stats.ReqPerSec),
			zap.Duration("runtime", time.Since(f.startedAt)),
		)
	}
}

func (f *Fuzzer) printStats(interval time.Duration) {
	for {
		select {
		case <-f.control:
			f.mutex.Lock()
			defer f.mutex.Unlock()

			logger.Log.Debug("shutting down stats",
				zap.Int("totalWorkers", f.totalWorkers),
			)

			f.totalWorkers--
			return

		case s := <-f.statsQueue:
			switch s {
			case "processed":
				f.stats.Processed += 1

			case "error":
				f.stats.Errors += 1

			case "saved":
				f.stats.Saved += 1
			}

			if time.Since(f.stats.LastCalculated) > interval {
				f.calculateStats()
				f.PrintStats()
			}

		case <-time.After(interval):
			f.calculateStats()
			f.PrintStats()
		}
	}
}
