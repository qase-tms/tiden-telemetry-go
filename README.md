# tiden-telemetry-go

[![CI](https://github.com/qase-tms/tiden-telemetry-go/actions/workflows/ci.yml/badge.svg)](https://github.com/qase-tms/tiden-telemetry-go/actions/workflows/ci.yml)
[![Go Reference](https://pkg.go.dev/badge/github.com/qase-tms/tiden-telemetry-go.svg)](https://pkg.go.dev/github.com/qase-tms/tiden-telemetry-go)
[![Go Report Card](https://goreportcard.com/badge/github.com/qase-tms/tiden-telemetry-go)](https://goreportcard.com/report/github.com/qase-tms/tiden-telemetry-go)

Error-tracking SDK for **Go** apps. Captures errors and panics and reports them
to your [Tiden](https://tiden.ai) project, with a built-in `net/http` recover
middleware. **Zero dependencies** (standard library only), no third-party
error-SDK dependency.

## Install

```bash
go get github.com/qase-tms/tiden-telemetry-go
```

## Quick start

```go
package main

import (
	"log"
	"os"

	tiden "github.com/qase-tms/tiden-telemetry-go"
)

func main() {
	if err := tiden.Init(tiden.Options{
		DSN:         os.Getenv("TIDEN_DSN"), // http://<publicKey>@<host:ingestPort>/<projectId>
		Release:     "my-app@1.2.3",
		Environment: "production",
	}); err != nil {
		log.Fatal(err)
	}

	if err := doWork(); err != nil {
		tiden.CaptureException(err)
	}
}
```

## Capturing

```go
tiden.CaptureException(err)
tiden.CaptureMessage("checkout completed", "info")

tiden.SetTag("plan", "pro")
tiden.SetUser(map[string]any{"id": "u_123"})

// report an in-flight panic, then re-panic so behavior is unchanged
func worker() {
	defer tiden.Recover()
	mightPanic()
}
```

## net/http middleware

Recovers panics in your handlers, reports them, and responds `500` instead of
crashing the server:

```go
mux := http.NewServeMux()
mux.HandleFunc("/", handler)

log.Fatal(http.ListenAndServe(":8080", tiden.Middleware(mux)))
```

The captured event includes request URL, method, and headers (sensitive headers
are scrubbed unless `SendDefaultPII` is set).

## Options

`tiden.Options`:

| Field | Type | Default | Description |
|---|---|---|---|
| `DSN` | `string` | — | **Required.** `http://<publicKey>@<host:ingestPort>/<projectId>`. |
| `Release` | `string` | — | App version, e.g. `my-app@1.2.3`. |
| `Environment` | `string` | — | e.g. `production`, `staging`. |
| `SendDefaultPII` | `bool` | `false` | Send likely-PII. Off by default — common PII is scrubbed before sending. |
| `BeforeSend` | `func(*Event) *Event` | — | Inspect, mutate, or drop an event. Return `nil` to drop. |
| `HTTPTimeout` | `time.Duration` | `2s` | Bounds each synchronous send. |
| `Transport` | `Transport` | HTTP | Override the transport (mainly for tests). |

For multiple clients (or to avoid global state), use `tiden.NewClient(opts)` and
call methods on the returned `*Client`; wrap handlers with
`tiden.HandlerWithClient(client, next)`.

## How it works

- Parses the DSN to `/api/<projectId>/envelope/?tiden_key=…`.
- Builds an event (`exception.values[]` with stack frames; `in_app` heuristic
  excludes the standard library and the module cache).
- Serializes the envelope and POSTs it with `Content-Type:
  application/x-tiden-envelope` — **synchronous, never panics into the host**;
  honors HTTP 429 + `Retry-After`.
- Scrubs likely-PII (auth headers, secret-ish keys) unless `SendDefaultPII`.

## Develop

```bash
go test -race ./...
go vet ./...
golangci-lint run ./...
```

## License

[MIT](LICENSE) © Tiden
