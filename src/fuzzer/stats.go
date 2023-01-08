package fuzzer

import (
	"time"

	"github.com/dpanic/fuzzer/src/logger"

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

func (f *Fuzzer) GetCurrentStats() {
	duration := time.Since(f.stats.LastPrint)
	seconds := duration.Seconds()
	processed := f.stats.Processed - f.stats.LastProcessed
	reqPerSec := float64(processed) / float64(seconds)

	logger.Log.Info("stats",
		zap.String("url", f.URL),
		zap.String("proxyURL", f.ProxyURL),
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

			if time.Since(f.stats.LastPrint) > interval {
				f.GetCurrentStats()
			}

		case <-time.After(interval):
			f.GetCurrentStats()
		}
	}
}
