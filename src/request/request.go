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
			ForceAttemptHTTP2: true,
			Proxy:             proxy,
			DialContext: (&net.Dialer{
				Timeout: 10 * time.Second,
			}).DialContext,

			Dial: (&net.Dialer{
				Timeout:   10 * time.Second,
				KeepAlive: 5 * time.Second,
			}).Dial,

			TLSHandshakeTimeout: 5 * time.Second,
			TLSClientConfig: &tls.Config{
				InsecureSkipVerify: true,
			},

			DisableKeepAlives:   false,
			MaxIdleConns:        1000,
			MaxIdleConnsPerHost: 500,
			MaxConnsPerHost:     500,
		},
	}
}

const (
	maxReadSize = 1 << 20
)

func Do(address, method string, body []byte, headers http.Header, customLogger *zap.Logger) (result []byte, statusCode int, location string, err error) {
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

	// tsStart := time.Now()
	// defer func() {
	// 	log.Debug("finished do request",
	// 		zap.Duration("duration", time.Since(tsStart)),
	// 	)
	// }()

	ctx, cancel := context.WithTimeout(context.Background(), timeout+time.Duration(2)*time.Second)
	defer cancel()

	httpRequest, err := http.NewRequestWithContext(ctx, method, address, bytes.NewBuffer(body))

	if headers != nil {
		httpRequest.Header = headers
	}

	if err != nil {
		// avoid stack trace
		log.Warn("error in creating request",
			zap.Error(err),
		)
		return
	}

	resp, err := client.Do(httpRequest)
	if err != nil {
		if resp != nil && resp.StatusCode != http.StatusNotFound {
			log.Warn("error in client",
				zap.Error(err),
			)
		}
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

	// initial size of result
	result = make([]byte, 0, 256<<10)

	// buffer size
	buf := make([]byte, 1024)
	var n int

	// Read only maxSize
	for {
		n, err = resp.Body.Read(buf)

		if n > 0 {
			buf = buf[:n]
			result = append(result, buf...)

			if len(result) > maxReadSize {
				result = result[:maxReadSize]
				err = errors.New("file limit has reached")
				break
			}
		}

		if err != nil && err != io.EOF {
			break
		}

		if err == io.EOF {
			break
		}
	}

	if err == io.ErrUnexpectedEOF || err == io.EOF {
		err = nil
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
