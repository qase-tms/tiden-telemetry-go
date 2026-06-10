package tiden

import (
	"bytes"
	"encoding/json"
	"time"
)

// contentType is the envelope media type the ingest backend accepts. Part of the
// wire contract (see contract_test.go).
const contentType = "application/x-tiden-envelope"

// serializeEnvelope produces the bytes the edge parses:
//
//	{envelope header}\n{item header (with byte length)}\n{event body}\n
func serializeEnvelope(ev *Event, sentAt time.Time) ([]byte, error) {
	body, err := marshalNoEscapeHTML(ev)
	if err != nil {
		return nil, err
	}

	header, err := marshalNoEscapeHTML(map[string]string{
		"event_id": ev.EventID,
		"sent_at":  sentAt.UTC().Format("2006-01-02T15:04:05Z"),
	})
	if err != nil {
		return nil, err
	}

	item, err := marshalNoEscapeHTML(map[string]any{
		"type":         "event",
		"length":       len(body), // byte length used by the edge for framing
		"content_type": "application/json",
	})
	if err != nil {
		return nil, err
	}

	out := make([]byte, 0, len(header)+len(item)+len(body)+3)
	out = append(out, header...)
	out = append(out, '\n')
	out = append(out, item...)
	out = append(out, '\n')
	out = append(out, body...)
	out = append(out, '\n')
	return out, nil
}

// marshalNoEscapeHTML is json.Marshal without HTML escaping (<, >, & stay
// literal) so error messages and URLs go on the wire unchanged.
func marshalNoEscapeHTML(v any) ([]byte, error) {
	var buf bytes.Buffer
	enc := json.NewEncoder(&buf)
	enc.SetEscapeHTML(false)
	if err := enc.Encode(v); err != nil {
		return nil, err
	}
	return bytes.TrimRight(buf.Bytes(), "\n"), nil // Encode appends a newline
}
