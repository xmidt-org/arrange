package arrangehttp

import (
	"context"
	"errors"
	"log"
	"net"
	"net/http"

	"go.uber.org/multierr"
)

type ServerOption interface {
	Apply(*http.Server) error
}

type ServerOptionFunc func(*http.Server) error

func (sof ServerOptionFunc) Apply(s *http.Server) error { return sof(s) }

type ServerOptions []ServerOption

func (so ServerOptions) Apply(s *http.Server) (err error) {
	for _, o := range so {
		err = multierr.Append(err, o.Apply(s))
	}

	return
}

func (so *ServerOptions) Add(opts ...any) {
	if len(opts) == 0 {
		return
	}

	if cap(*so) < (len(*so) + len(opts)) {
		bigger := make(ServerOptions, 0, len(*so)+len(opts))
		bigger = append(bigger, *so...)
		*so = bigger
	}

	for _, o := range opts {
		*so = append(*so, AsServerOption(o))
	}
}

func AsServerOption(v any) ServerOption {
	type serverOptionNoError interface {
		Apply(*http.Server)
	}

	if so, ok := v.(ServerOption); ok {
		return so
	} else if so, ok := v.(serverOptionNoError); ok {
		return ServerOptionFunc(func(s *http.Server) error {
			so.Apply(s)
			return nil
		})
	} else if f, ok := v.(func(*http.Server) error); ok {
		return ServerOptionFunc(f)
	} else if f, ok := v.(func(*http.Server)); ok {
		return ServerOptionFunc(func(s *http.Server) error {
			f(s)
			return nil
		})
	}

	return ServerOptionFunc(func(_ *http.Server) error {
		return errors.New("TODO")
	})
}

// BaseContext returns a server option that sets or replaces the http.Server.BaseContext function
func BaseContext(fn func(net.Listener) context.Context) ServerOption {
	return AsServerOption(func(s *http.Server) {
		s.BaseContext = fn
	})
}

// ConnContext returns a server option that sets or replaces the http.Server.ConnContext function
func ConnContext(fn func(context.Context, net.Conn) context.Context) ServerOption {
	return AsServerOption(func(s *http.Server) {
		s.ConnContext = fn
	})
}

// ErrorLog returns a server option that sets or replaces the http.Server.ErrorLog
func ErrorLog(l *log.Logger) ServerOption {
	return AsServerOption(func(s *http.Server) {
		s.ErrorLog = l
	})
}
