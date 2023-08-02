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

package arrangetest

import (
	"net"
	"time"
)

// ListenCapture returns a middleware that captures the bind address for
// a net.Listener.  The given channel receives net.Listener.Addr(), but the
// returned middleware does not decorate the listener at all.
//
// A typical use case is using ListenCapture as an external ListenerMiddleware
// during tests to capture the actual bind address for a test server, which will
// typically have an Addr of ":0".
func ListenCapture(ch chan<- net.Addr) func(net.Listener) net.Listener {
	return func(l net.Listener) net.Listener {
		ch <- l.Addr()
		return l
	}
}

// ListenReceive returns the first net.Addr received on a channel, typically previously
// passed to ListenCapture.  If timeout elapses, the enclosing test is failed.
//
// The v parameter must be convertible via AsTestable, or this function panics.
func ListenReceive(v any, ch <-chan net.Addr, timeout time.Duration) (addr net.Addr) {
	t := AsTestable(v)

	timer := time.NewTimer(timeout)
	defer timer.Stop()
	select {
	case addr = <-ch:
		// passing
	case <-timer.C:
		// fail
	}

	if addr == nil {
		t.Errorf("no listen address captured within %s", timeout)
		t.FailNow()
	}

	return
}
