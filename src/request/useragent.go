package request

import (
	"fmt"
	"time"
)

const (
	userAgent = "github.com/dpanic/fuzzer"
)

func GetUserAgent(ua string, pseudoRandom bool) (res string) {
	if ua == "" {
		res = userAgent
	} else {
		res = ua
	}

	if pseudoRandom {
		res = fmt.Sprintf("%s-%v", res, time.Now())
	}

	return
}
