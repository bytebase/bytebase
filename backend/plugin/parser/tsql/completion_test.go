package tsql

import (
	"context"
	"fmt"
	"strings"
	"testing"

	"github.com/bytebase/bytebase/backend/plugin/parser/base"
	"github.com/bytebase/bytebase/backend/store/model"
)

func TestCompletion(t *testing.T) {
	t.Parallel()
	tests := []struct {
		input string
	}{
		{
			input: "SEL|",
		},
	}
	for i, tt := range tests {
		fmt.Printf("[%02d]: TestCompletion %s\n", i, tt.input)
		newInput, caretLine, caretOffset := getCaretPosition(tt.input)
		getter, lister := buildMockDatabaseMetadataGetterLister()
		candidate, err := Completion(context.TODO(), newInput, caretLine, caretOffset, "Bytebase", getter, lister)
		if err != nil {
			t.Errorf("unexpected error: %v", err)
		}
		if len(candidate) == 0 {
			t.Errorf("no candidate found")
		}
	}
}

func getCaretPosition(statement string) (string, int, int) {
	lines := strings.Split(statement, "\n")
	for i, line := range lines {
		if offset := strings.Index(line, "|"); offset != -1 {
			newLine := strings.Replace(line, "|", "", 1)
			lines[i] = newLine
			return strings.Join(lines, "\n"), i + 1, offset
		}
	}
	panic("caret position not found")
}

func buildMockDatabaseMetadataGetterLister() (base.GetDatabaseMetadataFunc, base.ListDatabaseNamesFunc) {
	return func(context.Context, string) (string, *model.DatabaseMetadata, error) {
			return "", nil, nil
		}, func(context.Context) ([]string, error) {
			return nil, nil
		}
}
