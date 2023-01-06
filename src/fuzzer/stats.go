package fuzzer

import (
	"fuzzer/src/logger"
	"time"

	"go.uber.org/zap"
)

//go:generate easytags $GOFILE json:camel

type stats struct {
	LastPrint     time.Time `json:"lastPrint"`
	LastProcessed int       `json:"lastProcessed"`
	Total         int       `json:"total"`
	Processed     int       `json:"processed"`
	Errors        int       `json:"errors"`
	Saved         int       `json:"saved"`
}

func (f *Fuzzer) PrintStats() {
	f.mutexStats.Lock()
	defer f.mutexStats.Unlock()

	duration := time.Since(f.stats.LastPrint)
	seconds := duration.Seconds()
	processed := f.stats.Processed - f.stats.LastProcessed
	reqPerSec := float64(processed) / float64(seconds)

	logger.Log.Info("stats",
		zap.Int("total", f.stats.Total),
		zap.Int("processed", f.stats.Processed),
		zap.Int("left", f.stats.Total-f.stats.Processed),
		zap.Int("saved", f.stats.Saved),
		zap.Int("errors", f.stats.Errors),
		zap.Float64("req/s", reqPerSec),
		zap.Duration("runtime", time.Since(f.startedAt)),
	)

	f.stats.LastPrint = time.Now()
	f.stats.LastProcessed = f.stats.Processed
}

func (f *Fuzzer) Stats(interval time.Duration) {
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

		case <-time.After(interval):
			f.PrintStats()
		}
	}
}
