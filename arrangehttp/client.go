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
	"errors"
	"net/http"

	"github.com/xmidt-org/arrange"
	"go.uber.org/fx"
)

var (
	// ErrClientNameRequired indicates that ProvideClient or ProvideClientCustom was called
	// with an empty client name.
	ErrClientNameRequired = errors.New("A client name is required")
)

// NewClient is the primary client constructor for arrange.  Use this when you are creating a client
// from a (possibly unmarshaled) ClientConfig.  The options can be annotated to come from a value group,
// which is useful when there are multiple clients in a single fx.App.
func NewClient(cc ClientConfig, opts ...Option[http.Client]) (*http.Client, error) {
	return NewClientCustom(cc, opts...)
}

// NewClientCustom is an *http.Client constructor that allows customization of the concrete
// ClientFactory used to create the *http.Client.  This function is useful when you have a
// custom (possibly unmarshaled) configuration struct that implements ClientFactory.
//
// If the ClientFactory type also implements Option[http.Client], it is applied after
// all the other options are applied.
func NewClientCustom[F ClientFactory](cf F, opts ...Option[http.Client]) (c *http.Client, err error) {
	c, err = cf.NewClient()
	if err == nil {
		c, err = ApplyOptions(c, opts...)
	}

	if co, ok := any(cf).(Option[http.Client]); ok && err == nil {
		err = co.Apply(c)
	}

	return
}

// ProvideClient assembles a client out of application components in a standard, opinionated way.
// The clientName parameter is used as both the name of the *http.Client component and a prefix
// for that server's dependencies:
//
//   - NewClient is used to create the client as a component named clientName
//   - ClientConfig is an optional dependency with the name clientName+".config"
//   - []ClientOption is an value group dependency with the name clientName+".options"
//
// The external set of options, if supplied, is applied to the client after any injected options.
// This allows for options that come from outside the enclosing fx.App, as might be the case
// for options driven by the command line.
func ProvideClient(clientName string, external ...Option[http.Client]) fx.Option {
	return ProvideClientCustom[ClientConfig](clientName, external...)
}

// ProvideClientCustom is like ProvideClient, but it allows customization of the concrete
// ClientFactory dependency.
func ProvideClientCustom[F ClientFactory](clientName string, external ...Option[http.Client]) fx.Option {
	if len(clientName) == 0 {
		return fx.Error(ErrClientNameRequired)
	}

	return fx.Provide(
		fx.Annotate(
			NewClientCustom[F],
			arrange.Tags().
				OptionalName(clientName+".config").
				Group(clientName+".options").
				ParamTags(),
			arrange.Tags().Name(clientName).ResultTags(),
		),
	)
}
