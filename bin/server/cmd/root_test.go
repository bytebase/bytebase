package cmd

import (
	"fmt"
	"testing"
)

func TestDefaultExternalURLFromHost(t *testing.T) {
	tests := []struct {
		host string
		port int
		want string
	}{
		{
			host: "http://localhost",
			port: 3000,
			want: "http://localhost:3000",
		},
		{
			host: "   http://localhost  ",
			port: 3000,
			want: "http://localhost:3000",
		},
		{
			host: "https://localhost",
			port: 3000,
			want: "https://localhost:3000",
		},
		{
			host: "localhost",
			port: 3000,
			want: "http://localhost:3000",
		},
		{
			host: "0.0.0.0",
			port: 3000,
			want: "http://localhost:3000",
		},
	}
	for _, tt := range tests {
		t.Run(fmt.Sprintf("%s:%d", tt.host, tt.port), func(t *testing.T) {
			g := defaultExternalURLFromHostPort(tt.host, tt.port)
			if tt.want != g {
				t.Errorf("expect %s, got %s", tt.want, g)
			}
		})
	}
}

func TestNormalizeExternalURL(t *testing.T) {
	tests := []struct {
		url     string
		want    string
		wantErr bool
	}{
		{
			url:     "http://localhost:3000",
			want:    "http://localhost:3000",
			wantErr: false,
		},
		{
			url:     "https://localhost:3000",
			want:    "https://localhost:3000",
			wantErr: false,
		},
		{
			url:     "https://localhost",
			want:    "https://localhost",
			wantErr: false,
		},
		{
			url:     "http://localhost:80",
			want:    "http://localhost",
			wantErr: false,
		},
		{
			url:     "https://localhost:443",
			want:    "https://localhost",
			wantErr: false,
		},
		{
			url:     "  https://localhost:3000/ ",
			want:    "https://localhost:3000",
			wantErr: false,
		},
		// Missing http:// or https://
		{
			url:     "localhost:3000",
			want:    "",
			wantErr: true,
		},
		// Invalid port
		{
			url:     "http://localhost:xxx",
			want:    "",
			wantErr: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.url, func(t *testing.T) {
			g, err := normalizeExternalURL(tt.url)
			if err != nil {
				if !tt.wantErr {
					t.Errorf("expect no error, got %s", err.Error())
				}
			} else {
				if tt.wantErr {
					t.Errorf("expect error")
				} else if tt.want != g {
					t.Errorf("expect %s, got %s", tt.want, g)
				}
			}
		})
	}
}
