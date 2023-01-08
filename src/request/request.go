package request

import (
	"bytes"
	"context"
	"crypto/tls"
	"errors"
	"io"
	"net"
	"net/http"
	"net/url"
	"time"

	"github.com/dpanic/fuzzer/src/logger"

	"go.uber.org/zap"
)

var (
	client  *http.Client
	timeout = 20 * time.Second
)

func Setup(proxyURL string) {
	var proxy func(*http.Request) (*url.URL, error)

	if proxyURL != "" {
		fixedURL, _ := url.Parse(proxyURL)
		proxy = http.ProxyURL(fixedURL)
		logger.Log.Debug("proxy url is set",
			zap.String("proxyURL", proxyURL),
		)
	}

	client = &http.Client{
		Timeout: timeout,
		Transport: &http.Transport{
			Proxy: proxy,
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

			DisableKeepAlives:   true,
			MaxIdleConns:        1000,
			MaxIdleConnsPerHost: 500,
			MaxConnsPerHost:     500,
		},
	}
}

const (
	maxReadSize = 5 << 20
)

func Do(address, method string, body []byte, customLogger *zap.Logger) (result []byte, statusCode int, location string, err error) {
	var (
		status string
		log    *zap.Logger
	)

	log = customLogger.WithOptions(zap.Fields(
		zap.String("address", address),
		zap.String("method", method),
		zap.Int("lenBody", len(body)),
	),
	)

	ctx, cancel := context.WithTimeout(context.Background(), timeout+time.Duration(2)*time.Second)
	defer cancel()

	httpRequest, err := http.NewRequestWithContext(ctx, method, address, bytes.NewBuffer(body))

	if err != nil {
		// avoid stack trace
		log.Warn("error in creating request",
			zap.Error(err),
		)
		return
	}

	resp, err := client.Do(httpRequest)
	if err != nil {
		log.Warn("error in client",
			zap.Error(err),
		)
		return
	}
	defer func() {
		if resp != nil {
			resp.Body.Close()
		}
	}()

	if resp == nil {
		return
	}

	result = make([]byte, maxReadSize)
	n, err := io.ReadFull(resp.Body, result)

	if n == maxReadSize {
		err = errors.New("file limit has reached")
	}

	if resp != nil {
		location = resp.Request.URL.String()
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
