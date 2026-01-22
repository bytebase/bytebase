package ghost

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestParseGhostDirective(t *testing.T) {
	tests := []struct {
		name    string
		content string
		want    map[string]string
		wantErr bool
	}{
		{
			name:    "no directive",
			content: "ALTER TABLE users ADD COLUMN status VARCHAR(50);",
			want:    nil,
		},
		{
			name:    "empty flags",
			content: "-- gh-ost = {}\nALTER TABLE users ADD COLUMN status VARCHAR(50);",
			want:    map[string]string{},
		},
		{
			name:    "single flag",
			content: "-- gh-ost = {\"max-lag-millis\":\"1500\"}\nALTER TABLE users ADD COLUMN status VARCHAR(50);",
			want:    map[string]string{"max-lag-millis": "1500"},
		},
		{
			name:    "multiple flags",
			content: "-- gh-ost = {\"max-lag-millis\":\"1500\",\"cut-over-lock-timeout-seconds\":\"10\"}\nALTER TABLE users ADD COLUMN status VARCHAR(50);",
			want:    map[string]string{"max-lag-millis": "1500", "cut-over-lock-timeout-seconds": "10"},
		},
		{
			name:    "with other directives",
			content: "-- txn-mode = on\n-- gh-ost = {\"max-lag-millis\":\"1500\"}\nALTER TABLE users ADD COLUMN status VARCHAR(50);",
			want:    map[string]string{"max-lag-millis": "1500"},
		},
		{
			name:    "case insensitive",
			content: "-- GH-OST = {\"max-lag-millis\":\"1500\"}\nALTER TABLE users ADD COLUMN status VARCHAR(50);",
			want:    map[string]string{"max-lag-millis": "1500"},
		},
		{
			name:    "with spaces around equals",
			content: "--  gh-ost  =  {\"max-lag-millis\":\"1500\"}\nALTER TABLE users ADD COLUMN status VARCHAR(50);",
			want:    map[string]string{"max-lag-millis": "1500"},
		},
		{
			name:    "invalid json",
			content: "-- gh-ost = {invalid}\nALTER TABLE users ADD COLUMN status VARCHAR(50);",
			wantErr: true,
		},
		{
			name:    "empty flags with comment",
			content: "-- gh-ost = {} /*default config*/\nALTER TABLE users ADD COLUMN status VARCHAR(50);",
			want:    map[string]string{},
		},
		{
			name:    "empty flags with new default comment",
			content: "-- gh-ost = {} /* using default config */\nALTER TABLE users ADD COLUMN status VARCHAR(50);",
			want:    map[string]string{},
		},
		{
			name:    "with flags and comment",
			content: "-- gh-ost = {\"max-lag-millis\":\"1500\"} /*custom config*/\nALTER TABLE users ADD COLUMN status VARCHAR(50);",
			want:    map[string]string{"max-lag-millis": "1500"},
		},
		{
			name:    "with flags and new merge comment",
			content: "-- gh-ost = {\"max-lag-millis\":\"1500\"} /* merges with default config */\nALTER TABLE users ADD COLUMN status VARCHAR(50);",
			want:    map[string]string{"max-lag-millis": "1500"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := ParseGhostDirective(tt.content)
			if tt.wantErr {
				require.Error(t, err)
				return
			}
			require.NoError(t, err)
			require.Equal(t, tt.want, got)
		})
	}
}

func TestIsGhostEnabled(t *testing.T) {
	tests := []struct {
		name    string
		content string
		want    bool
	}{
		{
			name:    "no directive",
			content: "ALTER TABLE users ADD COLUMN status VARCHAR(50);",
			want:    false,
		},
		{
			name:    "with directive",
			content: "-- gh-ost = {}\nALTER TABLE users ADD COLUMN status VARCHAR(50);",
			want:    true,
		},
		{
			name:    "with flags",
			content: "-- gh-ost = {\"max-lag-millis\":\"1500\"}\nALTER TABLE users ADD COLUMN status VARCHAR(50);",
			want:    true,
		},
		{
			name:    "with comment",
			content: "-- gh-ost = {} /*default config*/\nALTER TABLE users ADD COLUMN status VARCHAR(50);",
			want:    true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := IsGhostEnabled(tt.content)
			require.Equal(t, tt.want, got)
		})
	}
}

func TestRemoveGhostDirective(t *testing.T) {
	tests := []struct {
		name    string
		content string
		want    string
	}{
		{
			name:    "no directive",
			content: "ALTER TABLE users ADD COLUMN status VARCHAR(50);",
			want:    "ALTER TABLE users ADD COLUMN status VARCHAR(50);",
		},
		{
			name:    "with directive",
			content: "-- gh-ost = {\"max-lag-millis\":\"1500\"}\nALTER TABLE users ADD COLUMN status VARCHAR(50);",
			want:    "ALTER TABLE users ADD COLUMN status VARCHAR(50);",
		},
		{
			name:    "with other directives",
			content: "-- txn-mode = on\n-- gh-ost = {\"max-lag-millis\":\"1500\"}\nALTER TABLE users ADD COLUMN status VARCHAR(50);",
			want:    "-- txn-mode = on\nALTER TABLE users ADD COLUMN status VARCHAR(50);",
		},
		{
			name:    "with comment",
			content: "-- gh-ost = {} /*default config*/\nALTER TABLE users ADD COLUMN status VARCHAR(50);",
			want:    "ALTER TABLE users ADD COLUMN status VARCHAR(50);",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := RemoveGhostDirective(tt.content)
			require.Equal(t, tt.want, got)
		})
	}
}
