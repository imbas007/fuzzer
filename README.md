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

## Todo:
- Slow down if being blocked [ % ]
- Random user agent [ % ]
- Random wait between requests [ % ]


