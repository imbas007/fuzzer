package request

import (
	"bytes"
	"context"
	"crypto/tls"
	"errors"
	"fuzzer/src/logger"
	"io"
	"net"
	"net/http"
	"time"

	"go.uber.org/zap"
)

var (
	ctx    context.Context
	client *http.Client
)

func init() {
	timeout := 20 * time.Second

	ctx, _ = context.WithTimeout(context.Background(), timeout+time.Duration(2)*time.Second)

	client = &http.Client{
		Timeout: timeout,
		Transport: &http.Transport{
			DialContext: (&net.Dialer{
				Timeout: 10 * time.Second,
			}).DialContext,

			Dial: (&net.Dialer{
				Timeout:   10 * time.Second,
				KeepAlive: 5 * time.Second,
			}).Dial,

			TLSHandshakeTimeout: 5 * time.Second,
			TLSClientConfig: &tls.Config{
				InsecureSkipVerify: false,
			},

			MaxIdleConns:        1000,
			MaxIdleConnsPerHost: 500,
			MaxConnsPerHost:     500,
		},
	}
}

const (
	maxReadSize = 1 << 20
)

func Do(address, method string, body []byte) (result []byte, statusCode int, location string, err error) {
	var (
		status string
	)

	log := logger.Log.WithOptions(zap.Fields(
		zap.String("address", address),
		zap.String("method", method),
		zap.Int("lenBody", len(body)),
	))

	httpRequest, err := http.NewRequestWithContext(ctx, method, address, bytes.NewBuffer(body))

	if err != nil {
		// avoid stack trace
		log.Warn("error in creating request",
			zap.Error(err),
		)
	}

	resp, err := client.Do(httpRequest)
	if err != nil {
		// avoid stack trace
		log.Warn("error in client do",
			zap.Error(err),
		)
	}
	defer resp.Body.Close()

	result = make([]byte, maxReadSize)
	n, err := io.ReadFull(resp.Body, result)

	if n == maxReadSize {
		err = errors.New("file limit has reached")
	}

	if resp != nil {
		loc, err := resp.Location()
		if err == nil {
			location = loc.String()
		}
		status = resp.Status
		statusCode = resp.StatusCode
	}

	switch err {
	case nil:
		//msg := fmt.Sprintf("Response larger than %dMB", maxSize)
		//log.Error(msg)

	case io.ErrUnexpectedEOF:
		err = nil
		result = result[:n]

	default:
		if n == 0 {
			err = nil
		} else {
			log.Error("error in processing response",
				zap.Error(err),
				zap.Int("downloadedBytes", n),
				zap.Int("statusCode", statusCode),
				zap.String("status", status),
			)
		}
	}

	return
}
