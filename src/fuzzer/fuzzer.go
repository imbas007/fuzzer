package fuzzer

import (
	"bufio"
	"errors"
	"io"
	"net/url"
	"os"
	"runtime"
	"strings"
	"sync"
	"time"

	"github.com/dpanic/fuzzer/src/logger"
	"github.com/dpanic/fuzzer/src/request"

	"go.uber.org/zap"
)

//go:generate easytags $GOFILE json:camel
type Fuzzer struct {
	Config
}

type Config struct {
	// URL defines target for fuzzing
	URL string `json:"url"`

	// Method defines which HTTP method should be used
	Method string `json:"method"`

	// WordList defines which word list should be used in fuzzing
	WordList string `json:"wordList"`

	// OutFile defines output of fuzzing process
	OutFile string `json:"outFile"`

	// MaxTime defines maximum runtime of fuzzer, if not set then indefinite
	MaxTime time.Duration `json:"maxTime"`

	// MaxReqSec defines maximum requests per second for fuzzer
	// if not set than no limits are applied
	MaxReqSec int `json:"maxReqSec"`

	// Filters perform filtering out of results per words, lines, size of body etc
	Filters Filters `json:"filters"`

	// ProxyURL defines HTTP forwarding proxy if set
	ProxyURL string `json:"proxyURL"`

	// IsSilent defines should fuzzer perform detailed logging or not
	IsSilent bool `json:"isSilent"`

	// err is external error which will be returned by .Wait() method
	err error

	// Done channel is used for close initialized by command line
	Done chan string `json:"exit"`

	// Events sends events by fuzzer, which can be parsed by third party
	Events chan Event `json:"events"`

	// maxWorkers is used to determine maximum number of go routines
	maxWorkers int
	mutex      *sync.Mutex

	// results is channel for streaming results, which is picked up by go routine and saved
	// into external file
	results chan Result

	// jobs is channel for streaming new jobs red from wordlist
	jobs chan job

	// control channel is used for shutting down started go routines such as
	// workers, fan in, stats, results
	control chan bool

	// burstyLimiter is channel used for creating ticks and by it controlling limit rate
	// of fuzzer
	burstyLimiter chan bool

	// stats defines stats, total, processed, errors etc.
	stats stats

	// statsQueue is used for sending results to stats structure
	statsQueue chan string

	// totalWorkers is used to determine how many go routines are up and running
	totalWorkers int

	// startedAt defines at which time fuzzer is started
	startedAt time.Time

	// Log you can define custom logger
	Log *zap.Logger
}

// Validate validates input params
func (f *Fuzzer) validate() (err error) {
	// set max worktime
	if f.MaxTime < 10 {
		f.MaxTime = 3600
	}

	// set where to save output
	if f.OutFile == "" {
		f.OutFile = "tmp/out.json"
		os.MkdirAll("tmp", 0755)
	}

	if f.WordList == "" {
		err = errors.New("word list must be defined")
		return
	}

	// set proxy url
	_, err = url.Parse(f.ProxyURL)
	if err != nil {
		f.ProxyURL = ""
	}
	err = nil

	// set target url
	if f.URL == "" {
		err = errors.New("target URL must be defined")
		return
	}

	_, err = url.Parse(f.URL)
	if err != nil {
		err = errors.New("error in parsing target url")
		return
	}

	return
}

// New generates basic new instance of Fuzzer
func New(config *Config) (fuzzer *Fuzzer, err error) {
	fuzzer = &Fuzzer{
		*config,
	}

	err = fuzzer.validate()
	if err != nil {
		return
	}

	fuzzer.maxWorkers = runtime.NumCPU()

	fuzzer.jobs = make(chan job, 4096)
	fuzzer.results = make(chan Result, 4096)
	fuzzer.control = make(chan bool, fuzzer.maxWorkers+3)
	fuzzer.startedAt = time.Now()
	fuzzer.mutex = &sync.Mutex{}
	fuzzer.statsQueue = make(chan string, fuzzer.maxWorkers*4)
	fuzzer.burstyLimiter = make(chan bool, 1)
	fuzzer.Done = make(chan string, 1)
	fuzzer.Events = make(chan Event, 4096)

	if fuzzer.Log == nil {
		fuzzer.Log = logger.Log
	}

	request.Setup(fuzzer.ProxyURL)

	fuzzer.totalWorkers = fuzzer.maxWorkers
	fuzzer.totalWorkers += 3 // fanin + results worker

	return
}

type job struct {
	URL string `json:"url"`
}

func (f *Fuzzer) Start() {
	log := f.Log.WithOptions(zap.Fields(
		zap.String("url", f.URL),
		zap.String("method", f.Method),
		zap.String("wordList", f.WordList),
		zap.String("outFile", f.OutFile),
		zap.Any("filters", f.Filters),
		zap.Duration("maxTime", f.MaxTime),
	))

	if f.MaxTime.Seconds() > 1 {
		log.Warn("max time is defined. setting countdown",
			zap.Duration("maxTime", f.MaxTime),
		)
		go func() {
			<-time.After(f.MaxTime)
			f.Stop()
			f.Done <- "timeouted"
			f.err = ErrMaxRuntime
		}()
	}

	go func() {
		for {
			time.Sleep(3 * time.Second)

			if f.stats.Total == f.stats.Processed {
				f.Stop()
				f.Done <- "done"
				return
			}
		}
	}()

	defer func() {
		f.mutex.Lock()
		defer f.mutex.Unlock()

		if !f.IsSilent {
			log.Debug("shutting down fan in",
				zap.Int("totalWorkers", f.totalWorkers),
			)
		}
		f.totalWorkers--
	}()

	main := strings.ReplaceAll(f.URL, "FUZZ", "")
	_, _, _, err := request.Do(main, f.Method, nil, f.Log)

	if err != nil {
		log.Warn("error in connecting to main url of server")
		return
	}

	// start workers
	for i := 0; i < f.maxWorkers; i++ {
		go f.Worker(i)
	}

	// open wordlist
	fd, err := os.OpenFile(f.WordList, os.O_RDONLY, os.ModePerm)
	if err != nil {
		log.Error("error in opening word list file",
			zap.Error(err),
		)
		f.err = err
		return
	}
	defer fd.Close()

	rd := bufio.NewReader(fd)

	// count lines
	for {
		_, err := rd.ReadString('\n')

		if err != nil {
			if err == io.EOF {
				break
			}
			break
		}

		f.stats.Total++
	}
	fd.Seek(0, io.SeekStart)

	var (
		shouldWork = true
	)

	// monitoring for control exit
	go func() {
		<-f.control

		f.mutex.Lock()
		defer f.mutex.Unlock()

		shouldWork = false

		if len(f.burstyLimiter) == 0 {
			f.burstyLimiter <- true
		}

		if len(f.jobs) > 0 {
			<-f.jobs
		}
	}()

	// print stats
	go f.printStats(3 * time.Second)

	// start results
	go f.saveResults()

	// enable limiter
	if f.MaxReqSec > 0 {
		go func() {
			interval := 1000 / f.MaxReqSec

			for {
				if !shouldWork {
					return
				}

				f.burstyLimiter <- true
				time.Sleep(time.Duration(interval) * time.Millisecond)
			}
		}()
	}

	for {
		f.mutex.Lock()
		if !shouldWork {
			f.mutex.Unlock()
			return
		}
		f.mutex.Unlock()

		line, err := rd.ReadString('\n')
		if err != nil {
			if err == io.EOF {
				shouldWork = false
				return
			}

			log.Error("error in reading line from file",
				zap.Error(err),
			)
			shouldWork = false
			return
		}

		line = strings.ReplaceAll(line, "\r", "")
		line = strings.ReplaceAll(line, "\n", "")
		line = strings.ReplaceAll(line, "\t", "")
		line = strings.Trim(line, " ")

		u := strings.ReplaceAll(f.URL, "FUZZ", line)

		// rate limit requests
		if f.MaxReqSec > 0 {
			<-f.burstyLimiter
		}

		f.jobs <- job{
			URL: u,
		}
	}
}

func (f *Fuzzer) Wait() (err error) {
	defer func() {
		err = f.err
	}()

	for {
		f.mutex.Lock()
		if f.totalWorkers == 0 {
			f.mutex.Unlock()
			return
		}
		f.mutex.Unlock()

		time.Sleep(1 * time.Second)
	}

}

// Stop sends intent to all workers to stop
func (f *Fuzzer) Stop() {
	for i := 0; i < f.maxWorkers; i++ {
		f.control <- true
	}

	// fan in
	f.control <- true

	// results worker
	f.control <- true

	// results stats
	f.control <- true

	f.Wait()
}
