package tiden

import "sync"

// Package-level client used by the top-level Init/Capture* helpers. Most apps
// configure one client at startup via Init and use these helpers everywhere.

var (
	globalMu sync.RWMutex
	global   *Client
)

// Init configures the package-level client. Call once at startup.
func Init(opts Options) error {
	c, err := NewClient(opts)
	if err != nil {
		return err
	}
	globalMu.Lock()
	global = c
	globalMu.Unlock()
	return nil
}

// CurrentClient returns the package-level client (nil if Init was not called).
func CurrentClient() *Client {
	globalMu.RLock()
	defer globalMu.RUnlock()
	return global
}

// CaptureException reports an error via the package-level client.
func CaptureException(err error) string {
	if c := CurrentClient(); c != nil {
		return c.CaptureException(err)
	}
	return ""
}

// CaptureExceptionAt reports an error returned by a handler via the
// package-level client, with the handler's location as the crash-site frame.
// See Client.CaptureExceptionAt.
func CaptureExceptionAt(err error, srv any, method string) string {
	if c := CurrentClient(); c != nil {
		return c.captureExceptionAt(err, srv, method)
	}
	return ""
}

// CaptureMessage reports a message via the package-level client.
func CaptureMessage(message, level string) string {
	if c := CurrentClient(); c != nil {
		return c.CaptureMessage(message, level)
	}
	return ""
}

// SetTag attaches a tag to subsequent events on the package-level client.
func SetTag(key, value string) {
	if c := CurrentClient(); c != nil {
		c.SetTag(key, value)
	}
}

// SetUser attaches a user to subsequent events on the package-level client.
func SetUser(user map[string]any) {
	if c := CurrentClient(); c != nil {
		c.SetUser(user)
	}
}

// Recover reports an in-flight panic via the package-level client and re-panics.
// Use it deferred: defer tiden.Recover().
func Recover() {
	if r := recover(); r != nil {
		if c := CurrentClient(); c != nil {
			c.captureRecovered(r, 3)
		}
		panic(r)
	}
}
