package arrangehttp

import (
	"context"
	"log"
	"net"
	"net/http"
	"reflect"
	"strings"

	"github.com/xmidt-org/arrange/internal/arrangereflect"
	"go.uber.org/multierr"
)

// InvalidServerOptionTypeError is returned by a ServerOption produced by AsServerOption
// to indicate that a type could not be converted.
type InvalidServerOptionTypeError struct {
	Type reflect.Type
}

// Error describes the type that could not be converted.
func (isote *InvalidServerOptionTypeError) Error() string {
	var o strings.Builder
	o.WriteString(isote.Type.String())
	o.WriteString(" cannot be converted to a ServerOption")

	return o.String()
}

// ServerOption is a general-purpose modifier for an http.Server.  Typically, these will
// created as value groups within an enclosing fx application.
type ServerOption interface {
	// ApplyToServer modifies the given server.  This method may return an error
	// to indicate that the option either made no sense given the state of the server
	// or was incorrectly applied.
	ApplyToServer(*http.Server) error
}

// ServerOptionFunc is a convenient function type that implements ServerOption.
type ServerOptionFunc func(*http.Server) error

// ApplyToServer invokes the function itself.
func (sof ServerOptionFunc) ApplyToServer(s *http.Server) error { return sof(s) }

// ServerOptions is an aggregate ServerOption that acts as a single option.
type ServerOptions []ServerOption

// ApplyToServer invokes each option in order.  Options are always invoked, even when
// one or more errors occur.  The returned error may be an aggregate error
// and can always be inspected via go.uber.org/multierr.
func (so ServerOptions) ApplyToServer(s *http.Server) (err error) {
	for _, o := range so {
		err = multierr.Append(err, o.ApplyToServer(s))
	}

	return
}

// AsServerOption converts a value into a ServerOption.  This function never returns nil
// and does not panic if v cannot be converted.
//
// Any of the following kinds of values can be converted:
//   - any type that implements ServerOption
//   - any type that supplies an ApplyToServer(*http.Server) method that returns no error
//   - an underlying type of func(*http.Server)
//   - an underlying type of func(*http.Server) error
//
// Any other kind of value will result in a ServerOption that returns an error indicating
// that the type cannot be converted.
//
// A common use case is to wrap a non-error closure:
//
//	arrangehttp.AsServerOption(func(s *http.Server) {
//	  s.Addr = ":something"
//	})
func AsServerOption(v any) ServerOption {
	type serverOptionNoError interface {
		ApplyToServer(*http.Server)
	}

	if so, ok := v.(ServerOption); ok {
		return so
	} else if so, ok := v.(serverOptionNoError); ok {
		return ServerOptionFunc(func(s *http.Server) error {
			so.ApplyToServer(s)
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

	return InvalidServerOption(
		&InvalidServerOptionTypeError{
			Type: reflect.TypeOf(v),
		},
	)
}

// InvalidServerOption returns an option that simply returns the given error.
// Useful in code where panics aren't desireable but indication of an error
// still needs to be returned during server construction.
func InvalidServerOption(err error) ServerOption {
	return ServerOptionFunc(func(_ *http.Server) error {
		return err
	})
}

// ConnState returns a server option that sets or replaces the http.Server.ConnState function.
func ConnState(fn func(net.Conn, http.ConnState)) ServerOption {
	return AsServerOption(func(s *http.Server) {
		s.ConnState = fn
	})
}

// BaseContextFunc is a composable type that is used to build http.Server.BaseContext functions.
type BaseContextFunc interface {
	~func(context.Context, net.Listener) context.Context
}

type baseContextFuncs[BCF BaseContextFunc] []BCF

func (bcf baseContextFuncs[BCF]) build(l net.Listener) (ctx context.Context) {
	ctx = context.Background()
	for _, fn := range bcf {
		ctx = fn(ctx, l)
	}

	return
}

// BaseContext returns a server option that sets or replaces the http.Server.BaseContext function.
// Each individual context function is composed to produce the context for the given listener.
func BaseContext[BCF BaseContextFunc](ctxFns ...BCF) ServerOption {
	return AsServerOption(func(s *http.Server) {
		if len(ctxFns) > 0 {
			bcf := make(baseContextFuncs[BCF], 0, len(ctxFns))
			bcf = append(bcf, ctxFns...)
			s.BaseContext = bcf.build
		}
	})
}

// ConnContextFunc is the type of function required by net/http.Server.ConnContext.
type ConnContextFunc interface {
	~func(context.Context, net.Conn) context.Context
}

type connContextFuncs[CCF ConnContextFunc] []CCF

func (ccf connContextFuncs[CCF]) build(ctx context.Context, c net.Conn) context.Context {
	for _, fn := range ccf {
		ctx = fn(ctx, c)
	}

	return ctx
}

// ConnContext returns a server option that sets or augments the http.Server.ConnContext function.
// Any existing ConnContext on the server is merged with the given functions to create a single
// ConnContext closure that uses each function to build the context for each server connection.
func ConnContext[CCF ConnContextFunc](ctxFns ...CCF) ServerOption {
	return AsServerOption(func(s *http.Server) {
		size := len(ctxFns)
		if size == 0 {
			return
		} else if s.ConnContext != nil {
			size += 1
		}

		ccf := make(connContextFuncs[CCF], 0, size)
		if s.ConnContext != nil {
			ccf = append(ccf, s.ConnContext)
		}

		ccf = append(ccf, ctxFns...)
		s.ConnContext = ccf.build
	})
}

// ErrorLog returns a server option that sets or replaces the http.Server.ErrorLog
func ErrorLog(l *log.Logger) ServerOption {
	return AsServerOption(func(s *http.Server) {
		s.ErrorLog = l
	})
}

// ServerMiddlewareFunc is the underlying type for any serverside middleware.
type ServerMiddlewareFunc interface {
	~func(http.Handler) http.Handler
}

// ServerMiddleware returns an option that applies any number of middleware functions
// to a server's handler.
func ServerMiddleware[M ServerMiddlewareFunc](fns ...M) ServerOption {
	return AsServerOption(func(s *http.Server) {
		s.Handler = arrangereflect.Decorate(
			arrangereflect.Safe[http.Handler](s.Handler, http.DefaultServeMux),
			fns...,
		)
	})
}
