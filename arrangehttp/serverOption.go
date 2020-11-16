package arrangehttp

import (
	"context"
	"log"
	"net"
	"net/http"
)

// ServerOption is a convenient type definition for function types which
// can be used with Server.Options.
type ServerOption func(*http.Server) error

// BaseContext returns a server option that sets or replaces the http.Server.BaseContext function
func BaseContext(fn func(net.Listener) context.Context) ServerOption {
	return func(s *http.Server) error {
		s.BaseContext = fn
		return nil
	}
}

// ConnContext returns a server option that sets or replaces the http.Server.ConnContext function
func ConnContext(fn func(context.Context, net.Conn) context.Context) ServerOption {
	return func(s *http.Server) error {
		s.ConnContext = fn
		return nil
	}
}

// ErrorLog returns a server option that sets or replaces the http.Server.ErrorLog
func ErrorLog(l *log.Logger) ServerOption {
	return func(s *http.Server) error {
		s.ErrorLog = l
		return nil
	}
}
