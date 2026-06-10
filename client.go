package tiden

import (
	"fmt"
	"reflect"
	"sync"
	"time"
)

// Options configures a Client.
type Options struct {
	// DSN is required: http://<publicKey>@<host:ingestPort>/<projectId>.
	DSN string
	// Release is the app version, e.g. "my-app@1.2.3".
	Release string
	// Environment is the deployment environment, e.g. "production".
	Environment string
	// SendDefaultPII disables PII scrubbing when true (off by default).
	SendDefaultPII bool
	// BeforeSend, if set, can mutate or drop an event (return nil to drop).
	BeforeSend func(*Event) *Event
	// Transport overrides the default synchronous HTTP transport (mainly tests).
	Transport Transport
	// HTTPTimeout bounds each synchronous send (default 2s).
	HTTPTimeout time.Duration
}

// Client builds events and sends them to a Tiden project. The zero value is not
// usable — construct one with NewClient.
type Client struct {
	opts      Options
	dsn       DSN
	transport Transport

	mu   sync.RWMutex
	tags map[string]string
	user map[string]any
}

// NewClient validates the DSN and returns a ready Client.
func NewClient(opts Options) (*Client, error) {
	dsn, err := ParseDSN(opts.DSN)
	if err != nil {
		return nil, err
	}

	timeout := opts.HTTPTimeout
	if timeout <= 0 {
		timeout = 2 * time.Second
	}
	tr := opts.Transport
	if tr == nil {
		tr = newHTTPTransport(dsn.IngestURL, timeout)
	}

	return &Client{opts: opts, dsn: dsn, transport: tr, tags: map[string]string{}}, nil
}

// SetUser attaches a user to subsequent events (nil clears it).
func (c *Client) SetUser(u map[string]any) {
	c.mu.Lock()
	c.user = u
	c.mu.Unlock()
}

// SetTag attaches a tag to subsequent events.
func (c *Client) SetTag(key, value string) {
	c.mu.Lock()
	c.tags[key] = value
	c.mu.Unlock()
}

// CaptureException reports an error and returns its event_id ("" if dropped).
func (c *Client) CaptureException(err error) string {
	if err == nil {
		return ""
	}
	ev := c.newEvent("error")
	ev.Exception = &exceptionValues{Values: []Exception{{
		Type:       reflect.TypeOf(err).String(),
		Value:      err.Error(),
		Stacktrace: newStacktrace(1),
	}}}
	return c.capture(ev)
}

// CaptureMessage reports a message at the given level (default "info").
func (c *Client) CaptureMessage(message, level string) string {
	if level == "" {
		level = "info"
	}
	ev := c.newEvent(level)
	ev.Message = message
	return c.capture(ev)
}

// CaptureEvent sends a pre-built event, filling in any missing required fields.
func (c *Client) CaptureEvent(ev *Event) string {
	if ev == nil {
		return ""
	}
	if ev.EventID == "" {
		ev.EventID = newEventID()
	}
	if ev.Timestamp == 0 {
		ev.Timestamp = nowUnix()
	}
	if ev.Platform == "" {
		ev.Platform = "go"
	}
	if ev.Level == "" {
		ev.Level = "error"
	}
	if ev.SDK.Name == "" {
		ev.SDK = sdkInfo
	}
	return c.capture(ev)
}

// Recover reports an in-flight panic and re-panics so the program's behavior is
// unchanged. Use it deferred:
//
//	defer client.Recover()
func (c *Client) Recover() {
	if r := recover(); r != nil {
		c.captureRecovered(r, 3)
		panic(r)
	}
}

func (c *Client) captureRecovered(r any, skip int) string {
	var msg string
	if err, ok := r.(error); ok {
		msg = err.Error()
	} else {
		msg = fmt.Sprintf("%v", r)
	}
	ev := c.newEvent("fatal")
	ev.Exception = &exceptionValues{Values: []Exception{{
		Type:       "panic",
		Value:      msg,
		Stacktrace: newStacktrace(skip),
	}}}
	return c.capture(ev)
}

func (c *Client) newEvent(level string) *Event {
	ev := &Event{
		EventID:     newEventID(),
		Timestamp:   nowUnix(),
		Platform:    "go",
		Level:       level,
		Release:     c.opts.Release,
		Environment: c.opts.Environment,
		SDK:         sdkInfo,
	}

	c.mu.RLock()
	if len(c.tags) > 0 {
		ev.Tags = make(map[string]string, len(c.tags))
		for k, v := range c.tags {
			ev.Tags[k] = v
		}
	}
	if c.user != nil {
		ev.User = c.user
	}
	c.mu.RUnlock()

	return ev
}

func (c *Client) capture(ev *Event) (id string) {
	// Our own code must never panic into the host app.
	defer func() { _ = recover() }()

	if !c.opts.SendDefaultPII {
		scrub(ev)
	}
	if c.opts.BeforeSend != nil {
		ev = c.opts.BeforeSend(ev)
		if ev == nil {
			return ""
		}
	}

	body, err := serializeEnvelope(ev, time.Now())
	if err != nil {
		return ""
	}
	c.transport.Send(body)
	return ev.EventID
}

func nowUnix() float64 {
	return float64(time.Now().UnixNano()) / 1e9
}
