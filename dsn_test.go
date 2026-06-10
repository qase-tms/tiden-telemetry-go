package tiden

import "testing"

func TestParseDSN(t *testing.T) {
	tests := []struct {
		name      string
		dsn       string
		wantURL   string
		wantKey   string
		wantProj  string
		wantError bool
	}{
		{
			name:     "with port",
			dsn:      "http://pub@host:1140/proj-1",
			wantURL:  "http://host:1140/api/proj-1/envelope/?tiden_key=pub",
			wantKey:  "pub",
			wantProj: "proj-1",
		},
		{
			name:     "no port, https",
			dsn:      "https://k@ingest.tiden.ai/42",
			wantURL:  "https://ingest.tiden.ai/api/42/envelope/?tiden_key=k",
			wantKey:  "k",
			wantProj: "42",
		},
		{name: "missing key", dsn: "http://host/proj", wantError: true},
		{name: "missing project", dsn: "http://pub@host/", wantError: true},
		{name: "garbage", dsn: "://nope", wantError: true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := ParseDSN(tt.dsn)
			if tt.wantError {
				if err == nil {
					t.Fatalf("expected error for %q", tt.dsn)
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if got.IngestURL != tt.wantURL {
				t.Errorf("IngestURL = %q, want %q", got.IngestURL, tt.wantURL)
			}
			if got.PublicKey != tt.wantKey {
				t.Errorf("PublicKey = %q, want %q", got.PublicKey, tt.wantKey)
			}
			if got.ProjectID != tt.wantProj {
				t.Errorf("ProjectID = %q, want %q", got.ProjectID, tt.wantProj)
			}
		})
	}
}
