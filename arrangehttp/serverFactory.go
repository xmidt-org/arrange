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
	"net"
	"net/http"
	"time"

	"github.com/xmidt-org/arrange/arrangetls"
	"github.com/xmidt-org/arrange/internal/arrangereflect"
	"github.com/xmidt-org/httpaux"
	"github.com/xmidt-org/httpaux/server"
)

// ServerFactory is the strategy for instantiating an *http.Server and an associated net.Listener.
// ServerConfig is this package's implementation of this interface, and allows a ServerFactory instance
// to be read from an external source.
//
// A custom ServerFactory implementation can be injected and used via NewServerCustom
// or ProvideServerCustom.
type ServerFactory interface {
	ListenerFactory

	// NewServer constructs an *http.Server.
	NewServer() (*http.Server, error)
}

// ServerConfig is the built-in ServerFactory implementation for this package.
// This struct can be unmarshaled from an external source, or supplied literally
// to the *fx.App.
type ServerConfig struct {
	// Network is the tcp network to listen on.  The default is "tcp".
	Network string `json:"network" yaml:"network"`

	// Address is the bind address of the server.  If unset, the server binds to
	// the first port available.  In that case, CaptureListenAddress can be used
	// to obtain the bind address for the server.
	Address string `json:"address" yaml:"address"`

	// ReadTimeout corresponds to http.Server.ReadTimeout
	ReadTimeout time.Duration `json:"readTimeout" yaml:"readTimeout"`

	// ReadHeaderTimeout corresponds to http.Server.ReadHeaderTimeout
	ReadHeaderTimeout time.Duration `json:"readHeaderTimeout" yaml:"readHeaderTimeout"`

	// WriteTime corresponds to http.Server.WriteTimeout
	WriteTimeout time.Duration `json:"writeTimeout" yaml:"writeTimeout"`

	// IdleTimeout corresponds to http.Server.IdleTimeout
	IdleTimeout time.Duration `json:"idleTimeout" yaml:"idleTimeout"`

	// MaxHeaderBytes corresponds to http.Server.MaxHeaderBytes
	MaxHeaderBytes int `json:"maxHeaderBytes" yaml:"maxHeaderBytes"`

	// KeepAlive corresponds to net.ListenConfig.KeepAlive.  This value is
	// only used for listeners created via Listen.
	KeepAlive time.Duration `json:"keepAlive" yaml:"keepAlive"`

	// Header supplies HTTP headers to emit on every response from this server
	Header http.Header `json:"header" yaml:"header"`

	// TLS is the optional unmarshaled TLS configuration.  If set, the resulting
	// server will use HTTPS.
	TLS *arrangetls.Config `json:"tls" yaml:"tls"`
}

// NewServer is the built-in implementation of ServerFactory in this package.
// This should serve most needs.  Nothing needs to be done to use this implementation.
// By default, a Fluent Builder chain begun with Server() will use ServerConfig.
func (sc ServerConfig) NewServer() (server *http.Server, err error) {
	server = &http.Server{
		Addr:              sc.Address,
		ReadTimeout:       sc.ReadTimeout,
		ReadHeaderTimeout: sc.ReadHeaderTimeout,
		WriteTimeout:      sc.WriteTimeout,
		IdleTimeout:       sc.IdleTimeout,
		MaxHeaderBytes:    sc.MaxHeaderBytes,
	}

	server.TLSConfig, err = sc.TLS.New()
	return
}

// Listen is the ListenerFactory implementation driven by ServerConfig
func (sc ServerConfig) Listen(ctx context.Context, s *http.Server) (net.Listener, error) {
	return DefaultListenerFactory{
		ListenConfig: net.ListenConfig{
			KeepAlive: sc.KeepAlive,
		},
		Network: sc.Network,
	}.Listen(ctx, s)
}

// Apply allows this configuration object to be seen as an Option[http.Server].
// This method adds the configured headers to every response.
func (sc ServerConfig) Apply(s *http.Server) error {
	if len(sc.Header) > 0 {
		header := httpaux.NewHeader(sc.Header)
		s.Handler = server.Header(header.SetTo)(
			arrangereflect.Safe[http.Handler](s.Handler, http.DefaultServeMux),
		)
	}

	return nil
}
