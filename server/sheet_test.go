package server

import (
	"fmt"
	"testing"

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
				EnvironmentName: "",
				DatabaseName:    "",
				SheetName:       "test",
			},
			err: nil,
		},
		{
			filePath:          "sheet/DEV__TEST__test.sql",
			sheetPathTemplate: "sheet/{{ENV_NAME}}__{{DB_NAME}}__{{NAME}}.sql",
			want: &SheetInfo{
				EnvironmentName: "DEV",
				DatabaseName:    "TEST",
				SheetName:       "test",
			},
			err: nil,
		},
		{
			filePath:          "sheet/DEV__test.sql",
			sheetPathTemplate: "sheet/{{ENV_NAME}}__{{NAME}}.sql",
			want: &SheetInfo{
				EnvironmentName: "DEV",
				DatabaseName:    "",
				SheetName:       "test",
			},
			err: nil,
		},
		{
			filePath:          "sheet/employee__test.sql",
			sheetPathTemplate: "sheet/{{DB_NAME}}__{{NAME}}.sql",
			want: &SheetInfo{
				EnvironmentName: "",
				DatabaseName:    "employee",
				SheetName:       "test",
			},
			err: nil,
		},
		{
			filePath:          "sheet/db-name.sql",
			sheetPathTemplate: "sheet/{{DB_NAME}}.sql",
			want: &SheetInfo{
				EnvironmentName: "",
				DatabaseName:    "db-name",
				SheetName:       "",
			},
			err: nil,
		},
		{
			filePath:          "sheet/db/test.sql",
			sheetPathTemplate: "sheet/{{NAME}}.sql",
			want:              nil,
			err:               fmt.Errorf("sheet path \"sheet/db/test.sql\" does not match sheet path template \"sheet/{{NAME}}.sql\""),
		},
		{
			filePath:          "my-sheet/test.sql",
			sheetPathTemplate: "sheet/{{NAME}}.sql",
			want:              nil,
			err:               fmt.Errorf("sheet path \"my-sheet/test.sql\" does not match sheet path template \"sheet/{{NAME}}.sql\""),
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
