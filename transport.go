package tiden

import (
	"bytes"
	"io"
	"net/http"
	"strconv"
	"sync"
	"time"
)

// Transport delivers a serialized envelope to the ingest edge.
type Transport interface {
	Send(envelope []byte)
}

// httpTransport posts envelopes synchronously. Every failure is swallowed —
// monitoring must never crash the host app. Honors HTTP 429 + Retry-After with a
// process-local gate.
type httpTransport struct {
	url    string
	client *http.Client

	mu               sync.Mutex
	rateLimitedUntil time.Time
}

func newHTTPTransport(url string, timeout time.Duration) *httpTransport {
	return &httpTransport{url: url, client: &http.Client{Timeout: timeout}}
}

func (t *httpTransport) Send(envelope []byte) {
	t.mu.Lock()
	limited := time.Now().Before(t.rateLimitedUntil)
	t.mu.Unlock()
	if limited {
		return
	}

	req, err := http.NewRequest(http.MethodPost, t.url, bytes.NewReader(envelope))
	if err != nil {
		return
	}
	req.Header.Set("Content-Type", contentType)

	resp, err := t.client.Do(req)
	if err != nil {
		return
	}
	defer func() {
		_, _ = io.Copy(io.Discard, resp.Body)
		_ = resp.Body.Close()
	}()

	if resp.StatusCode == http.StatusTooManyRequests {
		retry := 60 * time.Second
		if ra := resp.Header.Get("Retry-After"); ra != "" {
			if secs, err := strconv.Atoi(ra); err == nil && secs > 0 {
				retry = time.Duration(secs) * time.Second
			}
		}
		t.mu.Lock()
		t.rateLimitedUntil = time.Now().Add(retry)
		t.mu.Unlock()
	}
}
