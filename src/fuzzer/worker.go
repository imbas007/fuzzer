package fuzzer

import (
	"fmt"
	"strings"
	"time"

	"github.com/dpanic/fuzzer/src/request"

	"go.uber.org/zap"
)

func (f *Fuzzer) Worker(id int) {
	defer func() {
		f.mutex.Lock()
		defer f.mutex.Unlock()

		if !f.IsSilent {
			f.Log.Warn("shutting down worker",
				zap.Int("id", id),
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

		var (
			j job
		)

		select {
		case j = <-f.jobs:
		case <-time.After(3 * time.Second):
			continue
		}

		// if !f.IsSilent {
		// 	logger.Log.Debug("received new job!",
		// 		zap.String("url", j.URL),
		// 	)
		// }

		var (
			res        []byte
			statusCode int
			location   string
			err        error
			url        string
		)

		url = j.URL
		headers := request.GetHeaders()

		if f.PreExecuteRequestTransform != nil {
			fmt.Println(111)
			(f.PreExecuteRequestTransform)(&url, &f.ProxyURL, &headers)
		}

		fmt.Println(url, headers)

		res, statusCode, location, err = request.Do(url, f.Method, nil, headers, f.Log)

		if err != nil {
			f.statsQueue <- "error"
			f.statsQueue <- "processed"

			event := Event{
				Type:  EventTypeError,
				Value: err.Error(),
			}
			select {
			case f.Events <- event:
			case <-time.After(10 * time.Millisecond):
			}

			continue
		}

		lines := strings.Count(string(res), "\n")
		words := strings.Count(string(res), " ")

		redirectLocation := ""
		if location != url {
			redirectLocation = location
		}

		size := len(res)

		f.statsQueue <- "processed"

		if !f.filterResult(lines, words, size, statusCode) {
			continue
		}

		f.statsQueue <- "saved"

		f.results <- Result{
			RedirectLocation: redirectLocation,
			URL:              j.URL,
			Size:             size,
			Lines:            lines,
			StatusCode:       statusCode,
			Words:            words,
		}
	}
}
