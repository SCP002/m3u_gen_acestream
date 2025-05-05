package network

import (
	"net/http"
	"time"
)

// NewHTTPClient returns new HTTP client with a time limit for requests set to `timeout`.
func NewHTTPClient(timeout time.Duration) *http.Client {
	return &http.Client{
		Timeout: timeout,
		Transport: &http.Transport{
			Proxy: http.ProxyFromEnvironment,
		},
	}
}
