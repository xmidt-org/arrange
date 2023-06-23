package arrangehttp

import (
	"net/http"

	"github.com/xmidt-org/arrange/internal/arrangereflect"
)

// ClientMiddlewareFunc is the underlying type for round tripper decoration.
type ClientMiddlewareFunc interface {
	~func(http.RoundTripper) http.RoundTripper
}

// ClientMiddleware returns a ClientOption that applies the given middleware
// to the Transport (http.RoundTripper).
func ClientMiddleware[M ClientMiddlewareFunc](fns ...M) Option[http.Client] {
	return AsOption[http.Client](func(c *http.Client) {
		c.Transport = arrangereflect.Decorate(
			arrangereflect.Safe[http.RoundTripper](c.Transport, http.DefaultTransport),
			fns...,
		)
	})
}
