package util

import (
	"testing"

	"github.com/stretchr/testify/require"
	"google.golang.org/protobuf/encoding/protojson"
)

func TestBuildTimestampOrStringRowValue(t *testing.T) {
	tests := []struct {
		name       string
		value      string
		wantString bool
	}{
		{
			name:       "valid timestamp",
			value:      "2026-05-11 12:34:56.007",
			wantString: false,
		},
		{
			name:       "zero year timestamp",
			value:      "0000-01-01 00:00:00.007",
			wantString: true,
		},
		{
			name:       "zero date timestamp",
			value:      "0000-00-00 00:00:00",
			wantString: true,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got := BuildTimestampOrStringRowValue(tc.value, 3)
			_, err := protojson.Marshal(got)
			require.NoError(t, err)

			if tc.wantString {
				require.Equal(t, tc.value, got.GetStringValue())
				return
			}

			require.NotNil(t, got.GetTimestampValue())
			require.EqualValues(t, 3, got.GetTimestampValue().GetAccuracy())
		})
	}
}
