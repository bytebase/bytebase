package v1

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestSimpleParseCron(t *testing.T) {
	type result struct {
		hourOfDay int
		dayOfWeek int
	}

	testCases := []struct {
		cronStr string
		wantErr bool
		want    result
	}{
		{
			cronStr: "* * * * *",
			wantErr: true,
		},
		{
			cronStr: "* 24 * * 7",
			wantErr: true,
		},
		// 8:00 AM on Saturday
		{
			cronStr: "* 8 * * 6",
			wantErr: false,
			want: result{
				hourOfDay: 8,
				dayOfWeek: 6,
			},
		},
		// 8:00 AM every day
		{
			cronStr: "* 8 * * *",
			wantErr: false,
			want: result{
				hourOfDay: 8,
				dayOfWeek: -1,
			},
		},
	}

	for _, tc := range testCases {
		hourOfDay, dayOfWeek, err := parseSimpleCron(tc.cronStr)
		if tc.wantErr {
			require.Error(t, err)
		} else {
			require.Equal(t, tc.want.hourOfDay, hourOfDay)
			require.Equal(t, tc.want.dayOfWeek, dayOfWeek)
		}
	}
}
