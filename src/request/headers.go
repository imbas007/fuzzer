package request

import "net/http"

func GetHeaders() (headers http.Header) {
	return http.Header{}
}

func DefaultTransform(jobURL *string, proxyURL *string, headers *http.Header) {
	(*headers)["Target"] = []string{
		*jobURL,
	}

	*jobURL = *proxyURL
}
