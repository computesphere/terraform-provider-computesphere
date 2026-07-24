package provider

import (
	"testing"
)

func TestNormalizeAPIURL(t *testing.T) {
	tests := []struct {
		name    string
		in      string
		want    string
		wantErr bool
	}{
		{
			name: "empty defaults to canonical v2 base",
			in:   "",
			want: "https://api.computesphere.com/v2",
		},
		{
			name: "bare host gets scheme and v2 path",
			in:   "api.computesphere.com",
			want: "https://api.computesphere.com/v2",
		},
		{
			name: "bare host with v2 path gets scheme",
			in:   "api.computesphere.com/v2",
			want: "https://api.computesphere.com/v2",
		},
		{
			name: "scheme without path gets v2 path",
			in:   "https://api.computesphere.com",
			want: "https://api.computesphere.com/v2",
		},
		{
			name: "trailing slash on root is stripped",
			in:   "https://api.computesphere.com/",
			want: "https://api.computesphere.com/v2",
		},
		{
			name: "explicit v2 is unchanged",
			in:   "https://api.computesphere.com/v2",
			want: "https://api.computesphere.com/v2",
		},
		{
			name: "explicit v2 with trailing slash is stripped",
			in:   "https://api.computesphere.com/v2/",
			want: "https://api.computesphere.com/v2",
		},
		{
			name: "legacy /api/v1 rewritten to /v2",
			in:   "https://api.computesphere.com/api/v1",
			want: "https://api.computesphere.com/v2",
		},
		{
			name: "legacy /api/v2 rewritten to /v2",
			in:   "https://api.computesphere.com/api/v2",
			want: "https://api.computesphere.com/v2",
		},
		{
			name: "custom path preserved for unusual topologies",
			in:   "https://selfhosted.example.com/custom/v2",
			want: "https://selfhosted.example.com/custom/v2",
		},
		{
			name: "http scheme preserved",
			in:   "http://localhost:8000",
			want: "http://localhost:8000/v2",
		},
		{
			name: "surrounding whitespace trimmed",
			in:   "  api.computesphere.com  ",
			want: "https://api.computesphere.com/v2",
		},
		{
			name:    "garbage input errors",
			in:      "://not a url",
			wantErr: true,
		},
		{
			name:    "scheme without host errors",
			in:      "https://",
			wantErr: true,
		},
		{
			name:    "non-http scheme errors",
			in:      "ftp://api.computesphere.com",
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := normalizeAPIURL(tt.in)
			if tt.wantErr {
				if err == nil {
					t.Fatalf("normalizeAPIURL(%q) = %q, expected an error", tt.in, got)
				}
				return
			}
			if err != nil {
				t.Fatalf("normalizeAPIURL(%q) returned unexpected error: %v", tt.in, err)
			}
			if got != tt.want {
				t.Errorf("normalizeAPIURL(%q) = %q, want %q", tt.in, got, tt.want)
			}
		})
	}
}
