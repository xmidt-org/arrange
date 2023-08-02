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
