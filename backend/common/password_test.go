//nolint:revive
package common_test

import (
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/bytebase/bytebase/backend/common"
	storepb "github.com/bytebase/bytebase/backend/generated-go/store"
)

func TestValidatePassword(t *testing.T) {
	tests := []struct {
		name        string
		password    string
		restriction *storepb.WorkspaceProfileSetting_PasswordRestriction
		wantErr     string
	}{
		{
			name:     "minimum length",
			password: "Short1!",
			restriction: &storepb.WorkspaceProfileSetting_PasswordRestriction{
				MinLength: 8,
			},
			wantErr: "password length should no less than 8 characters",
		},
		{
			name:     "number required",
			password: "NoNumber!",
			restriction: &storepb.WorkspaceProfileSetting_PasswordRestriction{
				RequireNumber: true,
			},
			wantErr: "password must contains at least 1 number",
		},
		{
			name:     "letter required",
			password: "12345678!",
			restriction: &storepb.WorkspaceProfileSetting_PasswordRestriction{
				RequireLetter: true,
			},
			wantErr: "password must contains at least 1 lower case letter",
		},
		{
			name:     "uppercase letter required",
			password: "lowercase1!",
			restriction: &storepb.WorkspaceProfileSetting_PasswordRestriction{
				RequireUppercaseLetter: true,
			},
			wantErr: "password must contains at least 1 upper case letter",
		},
		{
			name:     "special character required",
			password: "Password1",
			restriction: &storepb.WorkspaceProfileSetting_PasswordRestriction{
				RequireSpecialCharacter: true,
			},
			wantErr: "password must contains at least 1 special character",
		},
		{
			name:     "all restrictions satisfied",
			password: "Password1!",
			restriction: &storepb.WorkspaceProfileSetting_PasswordRestriction{
				MinLength:               10,
				RequireNumber:           true,
				RequireLetter:           true,
				RequireUppercaseLetter:  true,
				RequireSpecialCharacter: true,
			},
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			err := common.ValidatePassword(test.password, test.restriction)
			if test.wantErr == "" {
				require.NoError(t, err)
				return
			}
			require.EqualError(t, err, test.wantErr)
		})
	}
}
