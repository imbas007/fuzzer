# README
[![Go Report Card](https://goreportcard.com/badge/github.com/dpanic/fuzzer)](https://goreportcard.com/report/github.com/dpanic/fuzzer)

Micro Web Fuzzer written in Go Lang.

## Features:
- Multi threaded ✅
- Filters:
    http codes ✅
    words ✅
    lines ✅
    size of body ✅
- Gracefoul shutdown ✅
- Reuse HTTP connection, don't create every request new TCP connection ✅
- Shuts down after maximum worktime ✅
- Limit requests per second ✅
- Low memory footprint ✅
- Save output in JSONL ✅


## Use:

Command line:
```
go run main.go \
    -maxReqSec 17 \
    -w wordlists/big.txt \
    -u https://google.com/FUZZ \
    -fc 403,404 \
    -maxTime 120 \
    -o tmp/test.json \
    -X GET
```

As a lib:
```
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
```

## Todo:
- Slow down if being blocked [ % ]
- Random user agent [ % ]
- Random wait between requests [ % ]


