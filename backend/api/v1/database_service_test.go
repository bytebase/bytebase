package v1

import (
	"testing"

	"github.com/stretchr/testify/require"

	storepb "github.com/bytebase/bytebase/proto/generated-go/store"
)

func TestIsSecretValid(t *testing.T) {
	testCases := []struct {
		item    *storepb.Secret
		wantErr bool
	}{
		// Disallow empty name.
		{
			item: &storepb.Secret{
				Name:        "",
				Value:       "pwd",
				Description: "",
			},
			wantErr: true,
		},
		// Disallow empty value.
		{
			item: &storepb.Secret{
				Name:        "NAME",
				Value:       "",
				Description: "",
			},
			wantErr: true,
		},
		// Name cannot start with a number.
		{
			item: &storepb.Secret{
				Name:        "1NAME",
				Value:       "pwd",
				Description: "",
			},
			wantErr: true,
		},
		// Name cannot start with the 'BYTEBASE_' prefix.
		{
			item: &storepb.Secret{
				Name:        "BYTEBASE_NAME",
				Value:       "pwd",
				Description: "",
			},
			wantErr: true,
		},
		// Names can only contain alphanumeric characters ([A-Z], [0-9]) or underscores (_). Spaces are not allowed.
		{
			item: &storepb.Secret{
				Name:        "NAME WITH SPACE",
				Value:       "pwd",
				Description: "",
			},
			wantErr: true,
		},
		{
			item: &storepb.Secret{
				Name:        "NAME-WITH-DASH",
				Value:       "pwd",
				Description: "",
			},
			wantErr: true,
		},
		{
			item: &storepb.Secret{
				Name:        "NAME_WITH_SPECIAL_CHARACTER_©",
				Value:       "pwd",
				Description: "",
			},
			wantErr: true,
		},
		{
			item: &storepb.Secret{
				Name:        "NAME_WITH_SPECIAL_CHARACTER_ç",
				Value:       "pwd",
				Description: "",
			},
			wantErr: true,
		},
		{
			item: &storepb.Secret{
				Name:        "NAME_WITH_SPECIAL_CHARACTER_Ⅷ",
				Value:       "pwd",
				Description: "",
			},
			wantErr: true,
		},
		{
			item: &storepb.Secret{
				Name:        "NAME_WITH_LOWER_CASE_a",
				Value:       "pwd",
				Description: "",
			},
			wantErr: true,
		},
		{
			item: &storepb.Secret{
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
