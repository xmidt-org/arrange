// SPDX-FileCopyrightText: 2023 Comcast Cable Communications Management, LLC
// SPDX-License-Identifier: Apache-2.0

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
