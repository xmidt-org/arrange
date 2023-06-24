package arrangehttp

import (
	"net/http"

	"github.com/xmidt-org/arrange/internal/arrangereflect"
)

// ClientMiddleware returns a ClientOption that applies the given middleware
// to the Transport (http.RoundTripper).
func ClientMiddleware[M Middleware[http.RoundTripper]](fns ...M) Option[http.Client] {
	return AsOption[http.Client](func(c *http.Client) {
		c.Transport = ApplyMiddleware(
			arrangereflect.Safe[http.RoundTripper](c.Transport, http.DefaultTransport),
			fns...,
		)
	})
}
