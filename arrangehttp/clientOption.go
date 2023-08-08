package arrangehttp

import (
	"github.com/xmidt-org/arrange/arrangemiddle"
	"github.com/xmidt-org/arrange/arrangeoption"
	"net/http"

	"github.com/xmidt-org/arrange/internal/arrangereflect"
)

// ClientMiddleware returns a ClientOption that applies the given middleware
// to the Transport (http.RoundTripper).
func ClientMiddleware[M arrangemiddle.Middleware[http.RoundTripper]](fns ...M) arrangeoption.Option[http.Client] {
	return arrangeoption.AsOption[http.Client](func(c *http.Client) {
		c.Transport = arrangemiddle.ApplyMiddleware(
			arrangereflect.Safe[http.RoundTripper](c.Transport, http.DefaultTransport),
			fns...,
		)
	})
}
