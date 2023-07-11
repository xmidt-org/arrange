package arrangehttp

import (
	"net/http"
	"time"

	"github.com/xmidt-org/arrange/arrangetls"
	"github.com/xmidt-org/arrange/internal/arrangereflect"
	"github.com/xmidt-org/httpaux"
	"github.com/xmidt-org/httpaux/roundtrip"
)

// ClientFactory is the interface implemented by unmarshaled configuration objects
// that produces an http.Client.  The default implementation of this interface is ClientConfig.
type ClientFactory interface {
	NewClient() (*http.Client, error)
}

// TransportConfig holds the unmarshalable configuration options for building an http.Transport.
// Fields in this struct correspond almost entirely with those in http.Transport.
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

// NewTransport creates an http.Transport using this unmarshaled configuration
// together with TLS information
func (tc TransportConfig) NewTransport(c *arrangetls.Config) (transport *http.Transport, err error) {
	transport = &http.Transport{
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

	transport.TLSClientConfig, err = c.New()
	return
}

// ClientConfig holds unmarshaled client configuration options.  It is the
// built-in ClientFactory implementation in this package.
type ClientConfig struct {
	Timeout   time.Duration
	Transport TransportConfig
	Header    http.Header
	TLS       *arrangetls.Config
}

// NewClient produces an http.Client given these unmarshaled configuration options
func (cc ClientConfig) NewClient() (client *http.Client, err error) {
	client = &http.Client{
		Timeout: cc.Timeout,
	}

	client.Transport, err = cc.Transport.NewTransport(cc.TLS)
	return
}

// Apply allows a ClientConfig to be used as an Option[http.Client].  This method
// decorates the client's transport so that the configured headers are supplied
// with every request.
func (cc ClientConfig) Apply(c *http.Client) error {
	if len(cc.Header) > 0 {
		header := httpaux.NewHeader(cc.Header)
		c.Transport = roundtrip.Header(header.SetTo)(
			arrangereflect.Safe(c.Transport, http.DefaultTransport),
		)
	}

	return nil
}
