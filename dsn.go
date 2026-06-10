package tiden

import (
	"fmt"
	"net/url"
	"strings"
)

// DSN is a parsed Tiden DSN.
type DSN struct {
	// IngestURL is the full envelope endpoint, including the tiden_key query param.
	IngestURL string
	PublicKey string
	ProjectID string
}

// ParseDSN turns a DSN of the form http://<publicKey>@host[:port]/<projectId>
// into the ingest URL the Tiden edge expects:
//
//	<scheme>://host/api/<projectId>/envelope/?tiden_key=<publicKey>
//
// (wire-compatible with the other Tiden SDKs — the edge reads the tiden_key
// query param.)
func ParseDSN(dsn string) (DSN, error) {
	u, err := url.Parse(dsn)
	if err != nil {
		return DSN{}, fmt.Errorf("tiden: invalid DSN: %w", err)
	}

	publicKey := u.User.Username()
	projectID := strings.SplitN(strings.TrimPrefix(u.Path, "/"), "/", 2)[0]

	if u.Scheme == "" || u.Host == "" || publicKey == "" || projectID == "" {
		return DSN{}, fmt.Errorf("tiden: invalid DSN (expected http://<publicKey>@host/<projectId>)")
	}

	ingest := fmt.Sprintf(
		"%s://%s/api/%s/envelope/?tiden_key=%s",
		u.Scheme, u.Host, url.PathEscape(projectID), url.QueryEscape(publicKey),
	)
	return DSN{IngestURL: ingest, PublicKey: publicKey, ProjectID: projectID}, nil
}
