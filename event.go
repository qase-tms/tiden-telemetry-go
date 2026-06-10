package tiden

import (
	"crypto/rand"
	"encoding/hex"
)

// Version is the SDK version, reported in every event's sdk field.
const Version = "0.1.0"

const modulePrefix = "github.com/qase-tms/tiden-telemetry-go"

var sdkInfo = SDKInfo{Name: "tiden.go", Version: Version}

// Event is the payload serialized into an envelope and sent to the ingest edge.
type Event struct {
	EventID     string            `json:"event_id"`
	Timestamp   float64           `json:"timestamp"`
	Platform    string            `json:"platform"`
	Level       string            `json:"level"`
	Release     string            `json:"release,omitempty"`
	Environment string            `json:"environment,omitempty"`
	SDK         SDKInfo           `json:"sdk"`
	Message     string            `json:"message,omitempty"`
	Exception   *exceptionValues  `json:"exception,omitempty"`
	Tags        map[string]string `json:"tags,omitempty"`
	User        map[string]any    `json:"user,omitempty"`
	Extra       map[string]any    `json:"extra,omitempty"`
	Contexts    map[string]any    `json:"contexts,omitempty"`
	Request     *Request          `json:"request,omitempty"`
}

// SDKInfo identifies the SDK that produced an event.
type SDKInfo struct {
	Name    string `json:"name"`
	Version string `json:"version"`
}

type exceptionValues struct {
	Values []Exception `json:"values"`
}

// Exception is one error in an event's exception chain.
type Exception struct {
	Type       string      `json:"type"`
	Value      string      `json:"value"`
	Stacktrace *Stacktrace `json:"stacktrace,omitempty"`
}

// Stacktrace is an ordered list of frames (outermost first, crash frame last).
type Stacktrace struct {
	Frames []Frame `json:"frames"`
}

// Frame is a single stack frame.
type Frame struct {
	Function string `json:"function,omitempty"`
	Filename string `json:"filename,omitempty"`
	AbsPath  string `json:"abs_path,omitempty"`
	Lineno   int    `json:"lineno,omitempty"`
	InApp    bool   `json:"in_app"`
}

// Request describes the HTTP request in scope when an event was captured.
type Request struct {
	URL     string            `json:"url,omitempty"`
	Method  string            `json:"method,omitempty"`
	Headers map[string]string `json:"headers,omitempty"`
}

// newEventID returns a 32-char hex UUIDv4 (no dashes) — the event_id shape the
// edge stores.
func newEventID() string {
	var b [16]byte
	if _, err := rand.Read(b[:]); err != nil {
		return ""
	}
	b[6] = (b[6] & 0x0f) | 0x40
	b[8] = (b[8] & 0x3f) | 0x80
	return hex.EncodeToString(b[:])
}
