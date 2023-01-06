package main

import (
	"flag"
	"fmt"
	"fuzzer/src/fuzzer"
	"fuzzer/src/logger"
	"os"
	"os/signal"
	"syscall"
	"time"
)

// args := []string{
// 	"-timeout", "2",
// 	"-maxtime", fmt.Sprintf("%.0f", maxTime.Seconds()),
// 	"-X", j.Method,
// 	"-c",
// 	"-noninteractive",
// 	// "-rate", "50",
// 	"-w", j.Dictionary,
// 	"-o", j.GetFileLoc("", "raw"),
// 	"-fc", "404",
// 	"-fc", "403",
// 	"-of", "json",
// }

func main() {
	maxTime := flag.Int("maxTime", 3600, "maximum execution time")
	maxReqSec := flag.Int("maxReqSec", 0, "maximum requests per second, default unlimited")
	method := flag.String("X", "GET", "GET, POST, HEAD, OPTIONS, PUT ...")
	filterCodes := flag.String("fc", "", "403,404")
	filterLines := flag.String("fl", "", "123,321")
	filterWords := flag.String("fw", "", "1,2,3")
	filterSize := flag.String("fs", "", "300,200")

	outFile := flag.String("o", "", "/tmp/outFile.json")
	wordList := flag.String("w", "", "wordlists/big.txt")
	url := flag.String("u", "", "https://www.google.com/FUZZ")
	proxyURL := flag.String("p", "", "https://10.106.0.2:1005")

	flag.Parse()

	f, err := fuzzer.New(&fuzzer.Config{
		URL:       *url,
		Method:    *method,
		ProxyURL:  *proxyURL,
		OutFile:   *outFile,
		WordList:  *wordList,
		MaxTime:   time.Duration(*maxTime) * time.Second,
		MaxReqSec: *maxReqSec,
		Filters: fuzzer.Filters{
			StatusCodes: fuzzer.GetUniqueNumbers(*filterCodes, ","),
			Words:       fuzzer.GetUniqueNumbers(*filterWords, ","),
			Lines:       fuzzer.GetUniqueNumbers(*filterLines, ","),
			Size:        fuzzer.GetUniqueNumbers(*filterSize, ","),
		},
	})

	if err != nil {
		flag.PrintDefaults()
		os.Exit(1)
	}

	go f.Start()

	signalChannel := make(chan os.Signal, 1)
	signal.Notify(signalChannel,
		syscall.SIGHUP,
		syscall.SIGINT,
		syscall.SIGTERM,
		syscall.SIGQUIT,
	)

	exitChannel := make(chan string)
	go func() {
		for {
			s := <-signalChannel
			logger.Log.Warn(fmt.Sprintf("received signal %s", s.String()))

			switch s {
			case syscall.SIGHUP:
			case syscall.SIGINT:
				exitChannel <- "SIGINT"
				return
			case syscall.SIGTERM:
				exitChannel <- "SIGTERM"
				return
			case syscall.SIGQUIT:
				exitChannel <- "SIGQUIT"
				return
			}
		}
	}()

	<-exitChannel
	f.Stop()
}
