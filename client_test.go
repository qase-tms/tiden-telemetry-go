package tiden

import (
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func decodeEvent(t *testing.T, envelope []byte) map[string]any {
	t.Helper()
	lines := strings.Split(strings.TrimRight(string(envelope), "\n"), "\n")
	if len(lines) != 3 {
		t.Fatalf("expected 3 lines, got %d", len(lines))
	}
	var ev map[string]any
	if err := json.Unmarshal([]byte(lines[2]), &ev); err != nil {
		t.Fatalf("decode event: %v", err)
	}
	return ev
}

func newTestClient(t *testing.T, opts Options) (*Client, *captureTransport) {
	t.Helper()
	tr := &captureTransport{}
	opts.DSN = "http://k@host/p"
	opts.Transport = tr
	c, err := NewClient(opts)
	if err != nil {
		t.Fatalf("NewClient: %v", err)
	}
	return c, tr
}

func TestCaptureMessage(t *testing.T) {
	c, tr := newTestClient(t, Options{})
	id := c.CaptureMessage("checkout completed", "info")
	if id == "" {
		t.Fatal("empty event id")
	}
	ev := decodeEvent(t, tr.last)
	if ev["message"] != "checkout completed" {
		t.Errorf("message = %v", ev["message"])
	}
	if ev["level"] != "info" {
		t.Errorf("level = %v", ev["level"])
	}
}

func TestBeforeSendDrop(t *testing.T) {
	c, tr := newTestClient(t, Options{
		BeforeSend: func(*Event) *Event { return nil },
	})
	id := c.CaptureException(errors.New("x"))
	if id != "" {
		t.Errorf("expected drop, got id %q", id)
	}
	if tr.last != nil {
		t.Error("event was sent despite BeforeSend dropping it")
	}
}

func TestBeforeSendMutate(t *testing.T) {
	c, tr := newTestClient(t, Options{
		BeforeSend: func(e *Event) *Event { e.Level = "warning"; return e },
	})
	c.CaptureException(errors.New("x"))
	ev := decodeEvent(t, tr.last)
	if ev["level"] != "warning" {
		t.Errorf("level = %v, want warning", ev["level"])
	}
}

func TestScrubExtra(t *testing.T) {
	c, tr := newTestClient(t, Options{})
	ev := c.newEvent("error")
	ev.Extra = map[string]any{"password": "hunter2", "ok": "keep"}
	c.capture(ev)

	got := decodeEvent(t, tr.last)
	extra := got["extra"].(map[string]any)
	if extra["password"] != "[Filtered]" {
		t.Errorf("password not scrubbed: %v", extra["password"])
	}
	if extra["ok"] != "keep" {
		t.Errorf("ok was altered: %v", extra["ok"])
	}
}

func TestSendDefaultPIIDisablesScrub(t *testing.T) {
	c, tr := newTestClient(t, Options{SendDefaultPII: true})
	ev := c.newEvent("error")
	ev.Extra = map[string]any{"password": "hunter2"}
	c.capture(ev)
	got := decodeEvent(t, tr.last)
	if got["extra"].(map[string]any)["password"] != "hunter2" {
		t.Error("scrub ran despite SendDefaultPII")
	}
}

func TestSetTagAndUser(t *testing.T) {
	c, tr := newTestClient(t, Options{})
	c.SetTag("plan", "pro")
	c.SetUser(map[string]any{"id": "u_1"})
	c.CaptureMessage("hi", "info")
	ev := decodeEvent(t, tr.last)
	if ev["tags"].(map[string]any)["plan"] != "pro" {
		t.Errorf("tag missing: %v", ev["tags"])
	}
	if ev["user"].(map[string]any)["id"] != "u_1" {
		t.Errorf("user missing: %v", ev["user"])
	}
}

func TestCaptureNilError(t *testing.T) {
	c, tr := newTestClient(t, Options{})
	if id := c.CaptureException(nil); id != "" {
		t.Errorf("nil error should be a no-op, got %q", id)
	}
	if tr.last != nil {
		t.Error("nil error sent an event")
	}
}

func TestStacktraceOrderCrashLast(t *testing.T) {
	c, tr := newTestClient(t, Options{})
	func() { c.CaptureException(errors.New("boom")) }()
	ev := decodeEvent(t, tr.last)
	frames := ev["exception"].(map[string]any)["values"].([]any)[0].(map[string]any)["stacktrace"].(map[string]any)["frames"].([]any)
	last := frames[len(frames)-1].(map[string]any)
	fn, _ := last["function"].(string)
	if !strings.Contains(fn, "TestStacktraceOrderCrashLast") {
		t.Errorf("crash frame should be last; last function = %q", fn)
	}
}

func TestHTTPMiddlewareRecoversPanic(t *testing.T) {
	tr := &captureTransport{}
	c, err := NewClient(Options{DSN: "http://k@host/p", Transport: tr})
	if err != nil {
		t.Fatal(err)
	}
	h := HandlerWithClient(c, http.HandlerFunc(func(http.ResponseWriter, *http.Request) {
		panic("handler boom")
	}))

	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "http://example/x?y=1", nil)
	req.Header.Set("Authorization", "secret")
	h.ServeHTTP(rec, req)

	if rec.Code != http.StatusInternalServerError {
		t.Errorf("status = %d, want 500", rec.Code)
	}
	if tr.last == nil {
		t.Fatal("panic not captured")
	}
	ev := decodeEvent(t, tr.last)
	ex := ev["exception"].(map[string]any)["values"].([]any)[0].(map[string]any)
	if ex["value"] != "handler boom" {
		t.Errorf("captured value = %v", ex["value"])
	}
	req2 := ev["request"].(map[string]any)
	if req2["method"] != "GET" {
		t.Errorf("request.method = %v", req2["method"])
	}
	if hdrs, ok := req2["headers"].(map[string]any); ok {
		if hdrs["Authorization"] != "[Filtered]" {
			t.Errorf("Authorization header not scrubbed: %v", hdrs["Authorization"])
		}
	}
}
