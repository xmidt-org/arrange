/**
 * Copyright 2023 Comcast Cable Communications Management, LLC
 *
 * Licensed under the Apache License, Version 2.0 (the "License");
 * you may not use this file except in compliance with the License.
 * You may obtain a copy of the License at
 *
 *     http://www.apache.org/licenses/LICENSE-2.0
 *
 * Unless required by applicable law or agreed to in writing, software
 * distributed under the License is distributed on an "AS IS" BASIS,
 * WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 * See the License for the specific language governing permissions and
 * limitations under the License.
 *
 */

package arrangehttp

import (
	"context"
	"log"
	"net"
	"net/http"

	"github.com/xmidt-org/arrange/internal/arrangereflect"
)

// ConnState returns a server option that sets or replaces the http.Server.ConnState function.
func ConnState(fn func(net.Conn, http.ConnState)) Option[http.Server] {
	return AsOption[http.Server](func(s *http.Server) {
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
func BaseContext[BCF BaseContextFunc](ctxFns ...BCF) Option[http.Server] {
	return AsOption[http.Server](func(s *http.Server) {
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
func ConnContext[CCF ConnContextFunc](ctxFns ...CCF) Option[http.Server] {
	return AsOption[http.Server](func(s *http.Server) {
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
func ErrorLog(l *log.Logger) Option[http.Server] {
	return AsOption[http.Server](func(s *http.Server) {
		s.ErrorLog = l
	})
}

// ServerMiddleware returns an option that applies any number of middleware functions
// to a server's handler.
func ServerMiddleware[M Middleware[http.Handler]](fns ...M) Option[http.Server] {
	return AsOption[http.Server](func(s *http.Server) {
		s.Handler = ApplyMiddleware(
			arrangereflect.Safe[http.Handler](s.Handler, http.DefaultServeMux),
			fns...,
		)
	})
}
