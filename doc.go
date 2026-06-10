// Package tiden is the error-tracking SDK for Go applications. It captures
// errors and panics and reports them to a Tiden project (https://tiden.ai),
// with a built-in net/http recover middleware. It has no third-party
// error-SDK dependency and emits the same envelope wire format as the other
// Tiden telemetry SDKs.
//
// Quick start:
//
//	func main() {
//	    if err := tiden.Init(tiden.Options{
//	        DSN:         os.Getenv("TIDEN_DSN"), // http://<publicKey>@<host:ingestPort>/<projectId>
//	        Release:     "my-app@1.2.3",
//	        Environment: "production",
//	    }); err != nil {
//	        log.Fatal(err)
//	    }
//
//	    if err := doWork(); err != nil {
//	        tiden.CaptureException(err)
//	    }
//	}
//
// Capture methods never panic into the host application.
package tiden
