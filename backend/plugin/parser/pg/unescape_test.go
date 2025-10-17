package pg

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestUnescapePostgreSQLString(t *testing.T) {
	tests := []struct {
		name    string
		input   string
		want    string
		wantErr bool
	}{
		// Dollar-quoted strings
		{
			name:  "simple dollar-quoted",
			input: "$$SELECT 1$$",
			want:  "SELECT 1",
		},
		{
			name:  "dollar-quoted with tag",
			input: "$tag$SELECT 1$tag$",
			want:  "SELECT 1",
		},
		{
			name:  "dollar-quoted with newlines",
			input: "$$\nSELECT\n  1\n$$",
			want:  "\nSELECT\n  1\n",
		},
		{
			name:  "dollar-quoted with single quotes",
			input: "$$SELECT 'hello'$$",
			want:  "SELECT 'hello'",
		},
		{
			name:  "dollar-quoted empty",
			input: "$$$$",
			want:  "",
		},
		{
			name:    "dollar-quoted unterminated",
			input:   "$incomplete",
			wantErr: true,
		},

		// Standard string constants
		{
			name:  "simple string constant",
			input: "'hello'",
			want:  "hello",
		},
		{
			name:  "string constant with escaped quote",
			input: "'hello''world'",
			want:  "hello'world",
		},
		{
			name:  "string constant multiple escaped quotes",
			input: "''''",
			want:  "'",
		},
		{
			name:  "string constant with spaces",
			input: "'  hello  world  '",
			want:  "  hello  world  ",
		},
		{
			name:  "string constant empty",
			input: "''",
			want:  "",
		},
		{
			name:  "string constant with newline (literal)",
			input: "'hello\nworld'",
			want:  "hello\nworld",
		},

		// Escape string constants
		{
			name:  "escape string with newline",
			input: "E'hello\\nworld'",
			want:  "hello\nworld",
		},
		{
			name:  "escape string with tab",
			input: "E'hello\\tworld'",
			want:  "hello\tworld",
		},
		{
			name:  "escape string with backslash",
			input: "E'back\\\\slash'",
			want:  "back\\slash",
		},
		{
			name:  "escape string with single quote",
			input: "E'hello\\'world'",
			want:  "hello'world",
		},
		{
			name:  "escape string with doubled single quote",
			input: "E'hello''world'",
			want:  "hello'world",
		},
		{
			name:  "escape string with carriage return",
			input: "E'hello\\rworld'",
			want:  "hello\rworld",
		},
		{
			name:  "escape string with backspace",
			input: "E'hello\\bworld'",
			want:  "hello\bworld",
		},
		{
			name:  "escape string with form feed",
			input: "E'hello\\fworld'",
			want:  "hello\fworld",
		},
		{
			name:  "escape string with double quote",
			input: "E'hello\\\"world'",
			want:  "hello\"world",
		},
		{
			name:  "escape string lowercase e",
			input: "e'hello\\nworld'",
			want:  "hello\nworld",
		},
		{
			name:  "escape string multiple escapes",
			input: "E'line1\\nline2\\tindented\\\\backslash'",
			want:  "line1\nline2\tindented\\backslash",
		},
		{
			name:    "escape string unterminated",
			input:   "E'hello",
			wantErr: true,
		},

		// Unicode escape string constants
		{
			name:  "unicode escape string simple",
			input: "U&'hello'",
			want:  "hello",
		},
		{
			name:  "unicode escape string with escaped quote",
			input: "U&'hello''world'",
			want:  "hello'world",
		},
		{
			name:  "unicode escape string lowercase",
			input: "u&'hello'",
			want:  "hello",
		},
		{
			name:    "unicode escape string unterminated",
			input:   "U&'hello",
			wantErr: true,
		},

		// Error cases
		{
			name:    "empty string",
			input:   "",
			wantErr: true,
		},
		{
			name:    "too short",
			input:   "x",
			wantErr: true,
		},
		{
			name:    "unknown type",
			input:   "hello",
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := unescapePostgreSQLString(tt.input)
			if tt.wantErr {
				assert.Error(t, err, "expected error but got none")
				return
			}
			require.NoError(t, err, "unexpected error")
			assert.Equal(t, tt.want, got, "unescaped string mismatch")
		})
	}
}

func TestUnescapeEscapeString(t *testing.T) {
	tests := []struct {
		name  string
		input string
		want  string
	}{
		{
			name:  "no escapes",
			input: "hello world",
			want:  "hello world",
		},
		{
			name:  "newline",
			input: "hello\\nworld",
			want:  "hello\nworld",
		},
		{
			name:  "tab",
			input: "hello\\tworld",
			want:  "hello\tworld",
		},
		{
			name:  "backslash",
			input: "hello\\\\world",
			want:  "hello\\world",
		},
		{
			name:  "single quote with backslash",
			input: "hello\\'world",
			want:  "hello'world",
		},
		{
			name:  "single quote doubled",
			input: "hello''world",
			want:  "hello'world",
		},
		{
			name:  "all escapes",
			input: "\\n\\t\\r\\b\\f\\\\\\'\\\"",
			want:  "\n\t\r\b\f\\'\"",
		},
		{
			name:  "unknown escape",
			input: "hello\\xworld",
			want:  "hello\\xworld",
		},
		{
			name:  "trailing backslash",
			input: "hello\\",
			want:  "hello\\",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := unescapeEscapeString(tt.input)
			assert.Equal(t, tt.want, got, "unescaped string mismatch")
		})
	}
}

func TestUnescapeUnicodeEscapeString(t *testing.T) {
	tests := []struct {
		name  string
		input string
		want  string
	}{
		{
			name:  "no escapes",
			input: "hello world",
			want:  "hello world",
		},
		{
			name:  "escaped quote",
			input: "hello''world",
			want:  "hello'world",
		},
		{
			name:  "multiple escaped quotes",
			input: "''hello''",
			want:  "'hello'",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := unescapeUnicodeEscapeString(tt.input)
			assert.Equal(t, tt.want, got, "unescaped string mismatch")
		})
	}
}
