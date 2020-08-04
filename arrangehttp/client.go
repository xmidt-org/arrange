package arrangehttp

import (
	"net/http"
	"time"
)

type ClientFactory interface {
	NewClient() (*http.Client, error)
}

type TransportConfig struct {
	TLSHandshakeTimeout    time.Duration
	DisableKeepAlives      bool
	DisableCompression     bool
	MaxIdleConns           int
	MaxIdleConnsPerHost    int
	MaxConnsPerHost        int
	IdleConnTimeout        time.Duration
	ResponseHeaderTimeout  time.Duration
	ExpectContinueTimeout  time.Duration
	ProxyConnectHeader     http.Header
	MaxResponseHeaderBytes int64
	WriteBufferSize        int
	ReadBufferSize         int
	ForceAttemptHTTP2      bool
}

func (tc TransportConfig) NewTransport() *http.Transport {
	return &http.Transport{
		TLSHandshakeTimeout:    tc.TLSHandshakeTimeout,
		DisableKeepAlives:      tc.DisableKeepAlives,
		DisableCompression:     tc.DisableCompression,
		MaxIdleConns:           tc.MaxIdleConns,
		MaxIdleConnsPerHost:    tc.MaxIdleConnsPerHost,
		MaxConnsPerHost:        tc.MaxConnsPerHost,
		IdleConnTimeout:        tc.IdleConnTimeout,
		ResponseHeaderTimeout:  tc.ResponseHeaderTimeout,
		ExpectContinueTimeout:  tc.ExpectContinueTimeout,
		ProxyConnectHeader:     tc.ProxyConnectHeader,
		MaxResponseHeaderBytes: tc.MaxResponseHeaderBytes,
		WriteBufferSize:        tc.WriteBufferSize,
		ReadBufferSize:         tc.ReadBufferSize,
		ForceAttemptHTTP2:      tc.ForceAttemptHTTP2,
	}
}

type ClientConfig struct {
	Timeout   time.Duration
	Transport TransportConfig
}

func (cc ClientConfig) NewClient() (*http.Client, error) {
	client := &http.Client{
		Timeout:   cc.Timeout,
		Transport: cc.Transport.NewTransport(),
	}

	// TODO: implement client TLS
	return client, nil
}

type ClientOption func(*http.Client) error

type C struct {
	co        []ClientOption
	prototype ClientFactory
}

func Client(opts ...ClientOption) *C {
	c := new(C)
	if len(opts) > 0 {
		// safe copy
		c.co = append([]ClientOption{}, opts...)
	}

	return c.ClientFactory(ClientConfig{})
}

func (c *C) ClientFactory(prototype ClientFactory) *C {
	c.prototype = prototype
	return c
}
