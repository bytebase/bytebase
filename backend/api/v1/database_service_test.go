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
				Name:        "NAME",
				Value:       "",
				Description: "",
			},
			wantErr: true,
		},
		// Name cannot start with a number.
		{
			item: &storepb.SecretItem{
				Name:        "1NAME",
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
		// Names can only contain alphanumeric characters ([A-Z], [0-9]) or underscores (_). Spaces are not allowed.
		{
			item: &storepb.SecretItem{
				Name:        "NAME WITH SPACE",
				Value:       "pwd",
				Description: "",
			},
			wantErr: true,
		},
		{
			item: &storepb.SecretItem{
				Name:        "NAME-WITH-DASH",
				Value:       "pwd",
				Description: "",
			},
			wantErr: true,
		},
		{
			item: &storepb.SecretItem{
				Name:        "NAME_WITH_SPECIAL_CHARACTER_©",
				Value:       "pwd",
				Description: "",
			},
			wantErr: true,
		},
		{
			item: &storepb.SecretItem{
				Name:        "NAME_WITH_SPECIAL_CHARACTER_ç",
				Value:       "pwd",
				Description: "",
			},
			wantErr: true,
		},
		{
			item: &storepb.SecretItem{
				Name:        "NAME_WITH_SPECIAL_CHARACTER_Ⅷ",
				Value:       "pwd",
				Description: "",
			},
			wantErr: true,
		},
		{
			item: &storepb.SecretItem{
				Name:        "NAME_WITH_LOWER_CASE_a",
				Value:       "pwd",
				Description: "",
			},
			wantErr: true,
		},
		{
			item: &storepb.SecretItem{
				Name:        "NORMAL_NAME",
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

func TestGetDatabasesFromExpression(t *testing.T) {
	tests := []struct {
		expression string
		want       []string
	}{
		{
			expression: "",
			want:       nil,
		},
		{
			expression: `request.time < timestamp("2023-12-16T06:16:57.064Z") && ((resource.database in ["instances/new-instance/databases/d2"]) || (resource.database == "instances/new-instance/databases/largedb" && resource.schema == "" && resource.table in ["hello0"]))`,
			want:       []string{"instances/new-instance/databases/d2", "instances/new-instance/databases/largedb"},
		},
	}

	for _, tc := range tests {
		got := getDatabasesFromExpression(tc.expression)
		require.Equal(t, tc.want, got)
	}
}
