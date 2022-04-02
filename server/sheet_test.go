package server

import (
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestParseSheetInfo(t *testing.T) {
	tests := []struct {
		filePath          string
		sheetPathTemplate string
		want              SheetInfo
	}{
		{
			filePath:          "sheet/test.sql",
			sheetPathTemplate: "sheet/{{NAME}}.sql",
			want: SheetInfo{
				EnvironmentName: "",
				DatabaseName:    "",
				SheetName:       "test",
			},
		},
		{
			filePath:          "sheet/DEV__TEST__test.sql",
			sheetPathTemplate: "sheet/{{ENV_NAME}}__{{DB_NAME}}__{{NAME}}.sql",
			want: SheetInfo{
				EnvironmentName: "DEV",
				DatabaseName:    "TEST",
				SheetName:       "test",
			},
		},
	}

	println(filepath.Dir("sheets/{{asd}}/{{Name}}.sql"))

	for _, test := range tests {
		result, err := parseSheetInfo(test.filePath, test.sheetPathTemplate)
		if err != nil {
			t.Errorf("Parse sheet path template error %v.", err)
		}
		require.Equal(t, *result, test.want)
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
			sheetPathTemplate: "sheets/123/{{ENV_NAME}}__{{DB_NAME}}__{{NAME}}.sql",
			want:              "sheets/123/",
		},
	}

	for _, test := range tests {
		result := parseBasePathFromTemplate(test.sheetPathTemplate)
		require.Equal(t, result, test.want)
	}
}
