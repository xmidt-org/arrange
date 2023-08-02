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

package arrangepprof

import (
	"net/http"
	"net/http/pprof"
	"path"
)

// DefaultPathPrefix is used as the path prefix for HTTP pprof handlers
// when no HTTP.PathPrefix field is supplied
const DefaultPathPrefix = "/debug/pprof"

// HTTP is the strategy for attaching pprof routes to an arbitrary *http.ServeMux.
type HTTP struct {
	// PathPrefix is the prefix URL for all the pprof routes.  If unset,
	// DefaultPathPrefix is used instead.  To bind the pprof routes to the
	// root URL, set this field to "/".
	PathPrefix string
}

// New constructs an *http.ServeMux with pprof routes configured.  This method
// can be passed to provide and annotated.
func (h HTTP) New() *http.ServeMux {
	mux := http.NewServeMux()
	return h.Apply(mux)
}

// Apply configures the pprof routes on an existing *http.ServeMux.  This method
// is primarily useful as an fx decorator.
func (h HTTP) Apply(mux *http.ServeMux) *http.ServeMux {
	prefix := h.PathPrefix
	if len(prefix) == 0 {
		prefix = DefaultPathPrefix
	}

	// special processing for the index handler:
	// register both a path with and without the trailing /
	indexPath := path.Join(prefix, "/")
	mux.HandleFunc(indexPath, pprof.Index)
	if indexPath[len(indexPath)-1] != '/' {
		mux.HandleFunc(indexPath+"/", pprof.Index)
	}

	mux.HandleFunc(path.Join(prefix, "/cmdline"), pprof.Cmdline)
	mux.HandleFunc(path.Join(prefix, "/profile"), pprof.Profile)
	mux.HandleFunc(path.Join(prefix, "/symbol"), pprof.Symbol)
	mux.HandleFunc(path.Join(prefix, "/trace"), pprof.Trace)

	return mux
}
