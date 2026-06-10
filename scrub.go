package tiden

import "strings"

const redacted = "[Filtered]"

// Redacts likely-PII before send (defense-in-depth; the edge also scrubs request
// headers server-side). Applied unless Options.SendDefaultPII is true.
var headerDenylist = map[string]bool{
	"authorization":       true,
	"cookie":              true,
	"set-cookie":          true,
	"x-api-key":           true,
	"x-auth-token":        true,
	"proxy-authorization": true,
}

var keyDenylist = map[string]bool{
	"password":      true,
	"passwd":        true,
	"secret":        true,
	"token":         true,
	"api_key":       true,
	"apikey":        true,
	"authorization": true,
	"credit_card":   true,
	"card_number":   true,
	"cvv":           true,
}

func scrub(ev *Event) {
	if ev.Request != nil && ev.Request.Headers != nil {
		for name := range ev.Request.Headers {
			if headerDenylist[strings.ToLower(name)] {
				ev.Request.Headers[name] = redacted
			}
		}
	}
	ev.User = scrubKeys(ev.User)
	ev.Extra = scrubKeys(ev.Extra)
	ev.Contexts = scrubKeys(ev.Contexts)
}

func scrubKeys(m map[string]any) map[string]any {
	if m == nil {
		return nil
	}
	for k, v := range m {
		if keyDenylist[strings.ToLower(k)] {
			m[k] = redacted
			continue
		}
		if sub, ok := v.(map[string]any); ok {
			m[k] = scrubKeys(sub)
		}
	}
	return m
}
