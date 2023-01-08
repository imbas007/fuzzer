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

type Fuzzer struct {
	Config
}

//go:generate easytags $GOFILE json:camel
type Config struct {
	URL       string        `json:"url"`
	Method    string        `json:"method"`
	WordList  string        `json:"wordList"`
	OutFile   string        `json:"outFile"`
	MaxTime   time.Duration `json:"maxTime"`
	MaxReqSec int           `json:"maxReqSec"`
	Filters   Filters       `json:"filters"`
	ProxyURL  string        `json:"proxyURL"`

	maxWorkers    int
	mutex         *sync.Mutex
	results       chan result
	jobs          chan job
	control       chan bool
	burstyLimiter chan bool
	ExitChannel   chan string

	stats        stats
	statsQueue   chan string
	totalWorkers int

	startedAt time.Time
}

type Filters struct {
	StatusCodes []int `json:"statusCodes"`
	Words       []int `json:"words"`
	Lines       []int `json:"lines"`
	Size        []int `json:"size"`
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
	fuzzer.results = make(chan result, 4096)
	fuzzer.control = make(chan bool, fuzzer.maxWorkers+3)
	fuzzer.startedAt = time.Now()
	fuzzer.mutex = &sync.Mutex{}
	fuzzer.statsQueue = make(chan string, fuzzer.maxWorkers*4)
	fuzzer.burstyLimiter = make(chan bool, 1)
	fuzzer.ExitChannel = make(chan string, 1)

	request.Setup(fuzzer.ProxyURL)

	return
}

type job struct {
	URL string `json:"url"`
}

func (f *Fuzzer) Start() {
	log := logger.Log.WithOptions(zap.Fields(
		zap.String("url", f.URL),
		zap.String("method", f.Method),
		zap.String("wordList", f.WordList),
		zap.String("outFile", f.OutFile),
		zap.Any("filters", f.Filters),
		zap.Duration("maxTime", f.MaxTime),
	))

	if f.MaxTime.Seconds() > 1 {
		logger.Log.Warn("max time is defined. setting countdown",
			zap.Duration("maxTime", f.MaxTime),
		)
		go func() {
			<-time.After(f.MaxTime)
			f.Stop()
			f.ExitChannel <- "timeouted"
		}()
	}

	go func() {
		for {
			time.Sleep(3 * time.Second)

			if f.stats.Total == f.stats.Processed {
				f.Stop()
				f.ExitChannel <- "timeouted"
				return
			}
		}
	}()

	defer func() {
		f.mutex.Lock()
		defer f.mutex.Unlock()

		logger.Log.Debug("shutting down fan in",
			zap.Int("totalWorkers", f.totalWorkers),
		)
		f.totalWorkers--
	}()

	// start workers
	for i := 0; i < f.maxWorkers; i++ {
		go f.Worker(i)
	}
	f.totalWorkers = f.maxWorkers
	f.totalWorkers += 3 // fanin + results worker

	// open wordlist
	fd, err := os.OpenFile(f.WordList, os.O_RDONLY, os.ModePerm)
	if err != nil {
		log.Error("error in opening word list file",
			zap.Error(err),
		)
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
	go f.Stats(3 * time.Second)

	// start results
	go f.Results()

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

		line = strings.Trim(line, "\n")
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

func (f *Fuzzer) waitAllWorkers() {
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

	f.waitAllWorkers()
}
