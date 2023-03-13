package server

import (
	"testing"

	"github.com/pkg/errors"
	"github.com/stretchr/testify/require"
)

func TestParseSheetInfo(t *testing.T) {
	tests := []struct {
		filePath          string
		sheetPathTemplate string
		want              *SheetInfo
		err               error
	}{
		{
			filePath:          "sheet/test.sql",
			sheetPathTemplate: "sheet/{{NAME}}.sql",
			want: &SheetInfo{
				EnvironmentID: "",
				DatabaseName:  "",
				SheetName:     "test",
			},
			err: nil,
		},
		{
			filePath:          "sheet/dev##TEST##test.sql",
			sheetPathTemplate: "sheet/{{ENV_ID}}##{{DB_NAME}}##{{NAME}}.sql",
			want: &SheetInfo{
				EnvironmentID: "dev",
				DatabaseName:  "TEST",
				SheetName:     "test",
			},
			err: nil,
		},
		{
			filePath:          "sheet/dev##test.sql",
			sheetPathTemplate: "sheet/{{ENV_ID}}##{{NAME}}.sql",
			want: &SheetInfo{
				EnvironmentID: "dev",
				DatabaseName:  "",
				SheetName:     "test",
			},
			err: nil,
		},
		{
			filePath:          "sheet/employee##test.sql",
			sheetPathTemplate: "sheet/{{DB_NAME}}##{{NAME}}.sql",
			want: &SheetInfo{
				EnvironmentID: "",
				DatabaseName:  "employee",
				SheetName:     "test",
			},
			err: nil,
		},
		{
			filePath:          "sheet/db-name.sql",
			sheetPathTemplate: "sheet/{{DB_NAME}}.sql",
			want: &SheetInfo{
				EnvironmentID: "",
				DatabaseName:  "db-name",
				SheetName:     "",
			},
			err: nil,
		},
		{
			filePath:          "sheet/db/test.sql",
			sheetPathTemplate: "sheet/{{NAME}}.sql",
			want:              nil,
			err:               errors.Errorf("sheet path \"sheet/db/test.sql\" does not match sheet path template \"sheet/{{NAME}}.sql\""),
		},
		{
			filePath:          "my-sheet/test.sql",
			sheetPathTemplate: "sheet/{{NAME}}.sql",
			want:              nil,
			err:               errors.Errorf("sheet path \"my-sheet/test.sql\" does not match sheet path template \"sheet/{{NAME}}.sql\""),
		},
	}

	for _, test := range tests {
		result, err := parseSheetInfo(test.filePath, test.sheetPathTemplate)
		if err != nil {
			if test.err != nil {
				require.Equal(t, test.err.Error(), err.Error())
			} else {
				t.Error(err)
			}
		} else {
			require.Equal(t, test.want, result)
		}
	}
}
