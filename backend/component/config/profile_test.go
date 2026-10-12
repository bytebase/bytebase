package config

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestParseEgressIPs(t *testing.T) {
	tests := []struct {
		name        string
		raw         string
		wantValid   []string
		wantInvalid []string
	}{
		{name: "unset"},
		{name: "one address", raw: "203.0.113.7", wantValid: []string{"203.0.113.7"}},
		{
			name:      "addresses and ranges keep their order",
			raw:       "203.0.113.7,198.51.100.0/24,2001:db8::1",
			wantValid: []string{"203.0.113.7", "198.51.100.0/24", "2001:db8::1"},
		},
		{
			name:      "surrounding spaces and empty entries are ignored",
			raw:       " 203.0.113.7 ,, 198.51.100.9 ,",
			wantValid: []string{"203.0.113.7", "198.51.100.9"},
		},
		{
			name:        "an entry that is neither an address nor a range is dropped",
			raw:         "203.0.113.7,egress.example.com,203.0.113.300",
			wantValid:   []string{"203.0.113.7"},
			wantInvalid: []string{"egress.example.com", "203.0.113.300"},
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			valid, invalid := ParseEgressIPs(test.raw)
			require.Equal(t, test.wantValid, valid)
			require.Equal(t, test.wantInvalid, invalid)
		})
	}
}
