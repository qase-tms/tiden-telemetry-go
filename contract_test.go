package tiden

import (
	"encoding/json"
	"errors"
	"strings"
	"testing"
)

// captureTransport records the last envelope instead of sending it.
type captureTransport struct{ last []byte }

func (t *captureTransport) Send(envelope []byte) { t.last = envelope }

// TestWireContract pins the exact bytes the SDK puts on the wire so they stay in
// sync with the ingest backend. If the backend changes the ingest interface
// (auth param, media type, envelope framing, or event field names), update the
// SDK + these assertions together — a drift makes this test fail loudly.
func TestWireContract(t *testing.T) {
	// (1) ingest URL + auth param name
	dsn, err := ParseDSN("http://pub@host:1140/proj-1")
	if err != nil {
		t.Fatalf("ParseDSN: %v", err)
	}
	if want := "http://host:1140/api/proj-1/envelope/?tiden_key=pub"; dsn.IngestURL != want {
		t.Fatalf("ingest URL = %q, want %q", dsn.IngestURL, want)
	}

	// (2) envelope media type
	if contentType != "application/x-tiden-envelope" {
		t.Fatalf("content type = %q", contentType)
	}

	tr := &captureTransport{}
	c, err := NewClient(Options{
		DSN:         "http://pub@host:1140/proj-1",
		Release:     "app@1.2.3",
		Environment: "production",
		Transport:   tr,
	})
	if err != nil {
		t.Fatalf("NewClient: %v", err)
	}
	c.CaptureException(errors.New("boom"))

	if tr.last == nil {
		t.Fatal("nothing sent")
	}

	// (3) envelope framing: {header}\n{item}\n{body}\n
	lines := strings.Split(strings.TrimRight(string(tr.last), "\n"), "\n")
	if len(lines) != 3 {
		t.Fatalf("expected 3 envelope lines, got %d", len(lines))
	}

	var header map[string]any
	var item map[string]any
	var event map[string]any
	mustJSON(t, lines[0], &header)
	mustJSON(t, lines[1], &item)
	mustJSON(t, lines[2], &event)

	if item["type"] != "event" {
		t.Errorf("item.type = %v", item["type"])
	}
	if item["content_type"] != "application/json" {
		t.Errorf("item.content_type = %v", item["content_type"])
	}
	if got := int(item["length"].(float64)); got != len(lines[2]) {
		t.Errorf("item.length = %d, want %d (byte len of body)", got, len(lines[2]))
	}
	if header["event_id"] != event["event_id"] {
		t.Errorf("header.event_id %v != event.event_id %v", header["event_id"], event["event_id"])
	}

	// (4) event schema the backend normalizer reads
	if event["platform"] != "go" {
		t.Errorf("platform = %v", event["platform"])
	}
	if event["level"] != "error" {
		t.Errorf("level = %v", event["level"])
	}
	if event["release"] != "app@1.2.3" {
		t.Errorf("release = %v", event["release"])
	}
	if event["environment"] != "production" {
		t.Errorf("environment = %v", event["environment"])
	}

	values := event["exception"].(map[string]any)["values"].([]any)
	ex := values[0].(map[string]any)
	if ex["type"] != "*errors.errorString" {
		t.Errorf("exception type = %v", ex["type"])
	}
	if ex["value"] != "boom" {
		t.Errorf("exception value = %v", ex["value"])
	}
	frames := ex["stacktrace"].(map[string]any)["frames"].([]any)
	if len(frames) == 0 {
		t.Fatal("no stack frames")
	}
	last := frames[len(frames)-1].(map[string]any)
	if _, ok := last["function"]; !ok {
		t.Error("last frame missing function")
	}
	if _, ok := last["in_app"]; !ok {
		t.Error("last frame missing in_app")
	}

	// event_id shape: 32-char hex
	id, _ := event["event_id"].(string)
	if len(id) != 32 {
		t.Errorf("event_id = %q (want 32 hex chars)", id)
	}
}

func mustJSON(t *testing.T, s string, v any) {
	t.Helper()
	if err := json.Unmarshal([]byte(s), v); err != nil {
		t.Fatalf("json %q: %v", s, err)
	}
}
