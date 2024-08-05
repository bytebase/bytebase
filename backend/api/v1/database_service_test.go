package v1

import (
	"testing"

	"github.com/stretchr/testify/require"

	storepb "github.com/bytebase/bytebase/proto/generated-go/store"
)

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
