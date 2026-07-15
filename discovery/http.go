package discovery

import (
	"net"
	"net/http"
	"time"
)

// NewHTTPClient returns a new http.Client with custom settings for timeouts and connection pooling.
func NewHTTPClient() *http.Client {
	return &http.Client{
		Transport: newTransport(),
		Timeout:   15 * time.Second,
	}
}

// newTransport returns a new http.Transport with custom settings for timeouts and connection pooling.
func newTransport() *http.Transport {
	dialer := net.Dialer{
		Timeout: 7 * time.Second,

		// Enable newer TCP keep-alive features on supported platforms.
		KeepAliveConfig: net.KeepAliveConfig{
			Enable:   true,
			Idle:     15 * time.Second,
			Interval: 15 * time.Second,
			Count:    3,
		},
	}

	t := &http.Transport{
		Proxy:                 http.ProxyFromEnvironment,
		ForceAttemptHTTP2:     false,
		DialContext:           dialer.DialContext,
		MaxIdleConns:          30, // default: unlimited
		MaxIdleConnsPerHost:   3,  // default: 2
		IdleConnTimeout:       90 * time.Second,
		TLSHandshakeTimeout:   5 * time.Second,
		ResponseHeaderTimeout: 15 * time.Second,
		ExpectContinueTimeout: 1 * time.Second,
		WriteBufferSize:       4 * 1024, // default: 4 kB
		ReadBufferSize:        4 * 1024, // default: 4 kB
	}
	return t
}
