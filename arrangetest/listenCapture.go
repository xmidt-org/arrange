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

func listenReceive(ch <-chan net.Addr, t <-chan time.Time) (net.Addr, bool) {
	select {
	case a := <-ch:
		return a, true
	case <-t:
		return nil, false
	}
}

// ListenReceive returns the first net.Addr received on a channel, typically previously
// passed to ListenCapture.  If timeout elapses, this function return nil, false.  Otherwise,
// the received net.Addr is returned along with true.
func ListenReceive(ch <-chan net.Addr, timeout time.Duration) (net.Addr, bool) {
	t := time.NewTimer(timeout)
	defer t.Stop()
	return listenReceive(ch, t.C)
}
