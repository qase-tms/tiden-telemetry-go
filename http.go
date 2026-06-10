package tiden

import (
	"fmt"
	"net/http"
	"strings"
)

// Middleware wraps an http.Handler so panics are reported to the package-level
// client and the server responds 500 instead of crashing. Configure the client
// with Init first.
//
//	mux := http.NewServeMux()
//	mux.HandleFunc("/", handler)
//	http.ListenAndServe(":8080", tiden.Middleware(mux))
func Middleware(next http.Handler) http.Handler {
	return HandlerWithClient(nil, next)
}

// HandlerWithClient is like Middleware but uses an explicit client (nil falls
// back to the package-level client).
func HandlerWithClient(client *Client, next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		defer func() {
			rec := recover()
			if rec == nil {
				return
			}

			c := client
			if c == nil {
				c = CurrentClient()
			}
			if c != nil {
				var msg string
				if err, ok := rec.(error); ok {
					msg = err.Error()
				} else {
					msg = fmt.Sprintf("%v", rec)
				}
				ev := c.newEvent("fatal")
				ev.Exception = &exceptionValues{Values: []Exception{{
					Type:       "panic",
					Value:      msg,
					Stacktrace: newStacktrace(2),
				}}}
				ev.Request = requestInfo(r)
				c.capture(ev)
			}

			w.WriteHeader(http.StatusInternalServerError)
		}()

		next.ServeHTTP(w, r)
	})
}

// requestInfo collects request context. Sensitive headers are redacted later by
// the scrubber unless SendDefaultPII is set.
func requestInfo(r *http.Request) *Request {
	headers := make(map[string]string, len(r.Header))
	for k, v := range r.Header {
		headers[k] = strings.Join(v, ", ")
	}
	scheme := "http"
	if r.TLS != nil {
		scheme = "https"
	}
	return &Request{
		URL:     scheme + "://" + r.Host + r.URL.Path,
		Method:  r.Method,
		Headers: headers,
	}
}
