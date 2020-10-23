package arrangehttp

import (
	"context"
	"log"
	"net"
	"net/http"

	"github.com/gorilla/mux"
	"github.com/xmidt-org/arrange"
)

// ServerOption is a functional option type that can be converted to an SOption.
// This type exists primarily to make fx.Provide constructors more concise.
type ServerOption func(*http.Server) error

// NopServerOption is a ServerOption that does nothing.  Used mainly in cases where
// no input is supplied to something that would otherwise build an option.
func NopServerOption(*http.Server) error { return nil }

// ServerMiddlewareChain is a strategy for decorating an http.Handler.  Various
// packages implement this interface, such as justinas/alice.
type ServerMiddlewareChain interface {
	Then(http.Handler) http.Handler
}

// sOption converts this ServerOption into the more general internal sOption
func (so ServerOption) sOption(server *http.Server, _ *mux.Router, lc ListenerChain) (ListenerChain, error) {
	return lc, so(server)
}

// ServerOptions binds several options into one.  Useful when providing
// several options as a component.
func ServerOptions(o ...ServerOption) ServerOption {
	if len(o) == 1 {
		return o[0]
	}

	return func(server *http.Server) error {
		for _, f := range o {
			if err := f(server); err != nil {
				return err
			}
		}

		return nil
	}
}

// RouterOption is a functional option type that can be converted to an SOption.
// This type exists primarily to make fx.Provide constructors more concise.
type RouterOption func(*mux.Router) error

// sOption converts this RouterOption into the more general internal sOption
func (ro RouterOption) sOption(_ *http.Server, router *mux.Router, lc ListenerChain) (ListenerChain, error) {
	return lc, ro(router)
}

// RouterOptions binds several options into one.  Useful when providing
// several options as a component.
func RouterOptions(o ...RouterOption) RouterOption {
	if len(o) == 1 {
		return o[0]
	}

	return func(router *mux.Router) error {
		for _, f := range o {
			if err := f(router); err != nil {
				return err
			}
		}

		return nil
	}
}

// BaseContext defines a ServerOption that sets http.Server.BaseContext.  The base context
// is built from one or more closures that accept a parent context and return a new context.
// Each closure is invoked with the context from the previous closure.
//
// Note that any previous BaseContext is overwritten by the returned option.
//
// If builders is empty, the returned option does nothing.
func BaseContext(builders ...func(context.Context, net.Listener) context.Context) ServerOption {
	if len(builders) == 0 {
		return NopServerOption
	}

	return func(s *http.Server) error {
		s.BaseContext = func(l net.Listener) context.Context {
			ctx := context.Background()
			for _, f := range builders {
				ctx = f(ctx, l)
			}

			return ctx
		}

		return nil
	}
}

// ConnContext defines a ServerOption that sets http.Server.ConnContext.  The connection context
// is built from one or more closures that accept a parent context and return a new context.
// Each closure is invoked with the context from the previous closure.
//
// Note that any previous ConnContext is overwritten by the returned option.
//
// If builders is empty, the returned option does nothing.
func ConnContext(builders ...func(context.Context, net.Conn) context.Context) ServerOption {
	if len(builders) == 0 {
		return NopServerOption
	}

	return func(s *http.Server) error {
		s.ConnContext = func(ctx context.Context, c net.Conn) context.Context {
			for _, f := range builders {
				ctx = f(ctx, c)
			}

			return ctx
		}

		return nil
	}
}

// ErrorLog defines a ServerOption that sets http.Server.ErrorLog.  This option overwrites
// any previous value for ErrorLog, even if its l parameter is nil.
func ErrorLog(l *log.Logger) ServerOption {
	return func(s *http.Server) error {
		s.ErrorLog = l
		return nil
	}
}

// ConnState defines a ServerOption that sets http.Server.ConnState.  Each closure
// is invoked in order.  If no closures are defined, the returned option does nothing.
func ConnState(cf ...func(net.Conn, http.ConnState)) ServerOption {
	if len(cf) == 0 {
		return NopServerOption
	}

	return func(s *http.Server) error {
		s.ConnState = func(c net.Conn, cs http.ConnState) {
			for _, f := range cf {
				f(c, cs)
			}
		}

		return nil
	}
}

// sOption is the internal option type used to configure an http.Server, its
// associated mux.Router, and any listener decoration.
type sOption func(*http.Server, *mux.Router, ListenerChain) (ListenerChain, error)

// newSOption reflects v to determine if it can be used as a functional option
// for building an HTTP server.  If v is not a recognized type, this function returns nil.
func newSOption(v interface{}) sOption {
	var so sOption
	arrange.TryConvert(
		v,
		func(o ServerOption) {
			so = o.sOption
		},
		func(o RouterOption) {
			so = o.sOption
		},
		func(m func(http.Handler) http.Handler) {
			so = RouterOption(func(router *mux.Router) error {
				router.Use(m)
				return nil
			}).sOption
		},
		// NOTE: supports value groups
		func(m []mux.MiddlewareFunc) {
			so = RouterOption(func(router *mux.Router) error {
				router.Use(m...)
				return nil
			}).sOption
		},
		func(smc ServerMiddlewareChain) {
			so = RouterOption(func(router *mux.Router) error {
				router.Use(smc.Then)
				return nil
			}).sOption
		},
		func(lc ListenerChain) {
			so = func(_ *http.Server, _ *mux.Router, chain ListenerChain) (ListenerChain, error) {
				return chain.Extend(lc), nil
			}
		},
		func(o ListenerConstructor) {
			so = func(_ *http.Server, _ *mux.Router, lc ListenerChain) (ListenerChain, error) {
				return lc.Append(o), nil
			}
		},
		// separate support for a slice of constructors allows injection of value groups
		func(o []ListenerConstructor) {
			so = func(_ *http.Server, _ *mux.Router, lc ListenerChain) (ListenerChain, error) {
				return lc.Append(o...), nil
			}
		},
	)

	return so
}
