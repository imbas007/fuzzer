package request

import "net/http"

const (
	userAgent = "github.com/dpanic/fuzzer"
)

func GetHeaders() (headers http.Header) {
	headers = http.Header{
		"user-agent": []string{
			userAgent,
		},
	}

	return
}

func DefaultTransform(jobURL *string, proxyURL *string, headers *http.Header) {
	(*headers)["Target"] = []string{
		*jobURL,
	}

	*jobURL = *proxyURL
}
