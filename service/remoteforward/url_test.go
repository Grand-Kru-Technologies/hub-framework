package remoteforward

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestJoinForwardURL_EmptyRequestPathKeepsPrefixSlash(t *testing.T) {
	got := JoinForwardURL("http://python:5000", "/apply/exampleTaskPython/", "")
	assert.Equal(t, "http://python:5000/apply/exampleTaskPython/", got)
}

func TestValidateForwardHost(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		host    string
		wantErr bool
	}{
		{name: "ftp scheme", host: "ftp://upstream.example", wantErr: true},
		{name: "ws scheme", host: "ws://upstream.example", wantErr: true},
		{name: "wss scheme", host: "wss://upstream.example", wantErr: true},
		{name: "file scheme", host: "file:///etc/passwd", wantErr: true},
		{name: "missing scheme", host: "upstream.example", wantErr: true},
		{name: "http scheme without host", host: "http://", wantErr: true},
		{name: "https scheme without host", host: "https://", wantErr: true},
		{
			name:    "rejected host redacts userinfo password",
			host:    "ftp://user:super-secret@upstream.example/path?q=1#frag",
			wantErr: true,
		},
		{name: "http without port", host: "http://upstream.example", wantErr: false},
		{name: "https without port", host: "https://upstream.example", wantErr: false},
		{name: "http with port", host: "http://upstream.example:8080", wantErr: false},
		{name: "https with port", host: "https://upstream.example:8443", wantErr: false},
		{name: "http with path prefix", host: "http://upstream.example/api/v1/", wantErr: false},
		{name: "https uppercase scheme", host: "HTTPS://upstream.example", wantErr: false},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			err := ValidateForwardHost(tt.host)
			if tt.wantErr {
				require.Error(t, err)
				assert.Contains(t, err.Error(), "absolute http or https URL")
				assert.NotContains(t, err.Error(), "super-secret")
				if strings.Contains(tt.host, "super-secret") {
					assert.NotContains(t, err.Error(), "user:")
				}
				return
			}
			require.NoError(t, err)
		})
	}
}

func TestRedactHostConfigForError(t *testing.T) {
	t.Parallel()

	redacted := RedactHostConfigForError("https://user:secret@host.example:443/prefix?token=abc#section")
	assert.NotContains(t, redacted, "secret")
	assert.NotContains(t, redacted, "token=abc")
	assert.NotContains(t, redacted, "section")
	assert.Contains(t, redacted, "host.example")
}
