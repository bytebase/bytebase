package v1

import (
	"testing"

	"github.com/stretchr/testify/require"

	storepb "github.com/bytebase/bytebase/proto/generated-go/store"
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

func TestIsSecretValid(t *testing.T) {
	testCases := []struct {
		item    *storepb.SecretItem
		wantErr bool
	}{
		// Disallow empty name.
		{
			item: &storepb.SecretItem{
				Name:        "",
				Value:       "pwd",
				Description: "",
			},
			wantErr: true,
		},
		// Disallow empty value.
		{
			item: &storepb.SecretItem{
				Name:        "name",
				Value:       "",
				Description: "",
			},
			wantErr: true,
		},
		// Name cannot start with a number.
		{
			item: &storepb.SecretItem{
				Name:        "1name",
				Value:       "pwd",
				Description: "",
			},
			wantErr: true,
		},
		// Name cannot start with the 'BYTEBASE_' prefix.
		{
			item: &storepb.SecretItem{
				Name:        "BYTEBASE_NAME",
				Value:       "pwd",
				Description: "",
			},
			wantErr: true,
		},
		// Names can only contain alphanumeric characters ([a-z], [A-Z], [0-9]) or underscores (_). Spaces are not allowed.
		{
			item: &storepb.SecretItem{
				Name:        "name with space",
				Value:       "pwd",
				Description: "",
			},
			wantErr: true,
		},
		{
			item: &storepb.SecretItem{
				Name:        "name-with-hyphen",
				Value:       "pwd",
				Description: "",
			},
			wantErr: true,
		},
		{
			item: &storepb.SecretItem{
				Name:        "name_with_special_characters_©",
				Value:       "pwd",
				Description: "",
			},
			wantErr: true,
		},
		{
			item: &storepb.SecretItem{
				Name:        "name_with_special_characters_normal",
				Value:       "pwd",
				Description: "",
			},
			wantErr: false,
		},
	}

	for _, tc := range testCases {
		err := isSecretValid(tc.item)
		if tc.wantErr {
			require.Error(t, err)
		} else {
			require.NoError(t, err)
		}
	}
}
