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
			filePath:          "sheet/db/test.sql",
			sheetPathTemplate: "sheet/{{NAME}}.sql",
			want:              nil,
			err:               fmt.Errorf("sheet path \"sheet/db/test.sql\" does not match sheet path template \"sheet/{{NAME}}.sql\""),
		},
		{
			filePath:          "sheet/db-name.sql",
			sheetPathTemplate: "sheet/{{DB_NAME}}.sql",
			want:              nil,
			err:               fmt.Errorf("sheet name cannot be empty from sheet path \"sheet/db-name.sql\" and template \"sheet/{{DB_NAME}}.sql\""),
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
			require.Equal(t, test.err.Error(), err.Error())
		} else {
			require.Equal(t, test.want, result)
		}
	}
}

func TestParseBasePathFromTemplate(t *testing.T) {
	tests := []struct {
		sheetPathTemplate string
		want              string
	}{
		{
			sheetPathTemplate: "sheet/{{NAME}}.sql",
			want:              "sheet/",
		},
		{
			sheetPathTemplate: "sheets/dir/{{ENV_NAME}}__{{DB_NAME}}__{{NAME}}.sql",
			want:              "sheets/dir/",
		},
		{
			sheetPathTemplate: "sheets/dir__{{NAME}}.sql",
			want:              "sheets/",
		},
		{
			sheetPathTemplate: "{{NAME}}.sql",
			want:              "",
		},
		{
			sheetPathTemplate: "sheets/dir/",
			want:              "sheets/dir/",
		},
	}

	for _, test := range tests {
		result := parseBasePathFromTemplate(test.sheetPathTemplate)
		require.Equal(t, test.want, result)
	}
}
