package tsql

import (
	"context"
	"io"
	"os"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
	"gopkg.in/yaml.v3"

	"github.com/bytebase/bytebase/backend/plugin/parser/base"
	"github.com/bytebase/bytebase/backend/store/model"
)

type candidatesTest struct {
	Description string           `yaml:"description"`
	Input       string           `yaml:"input"`
	Want        []base.Candidate `yaml:"want"`
}

// TestCompletion tests the Transact-SQL auto-completion, all the test cases are stored in the file.
//
// - Description: The description of the test case.
//
// - Input: The input statement with the caret position marked by "|".
//
// - Want: The expected completion candidates.
//
// Our Test suite will determine the caret position - line(0-based) and column(1-based) by the position of the "|", actually,
// this caret position is as same as the position in the monaco-editor(LSP?).
func TestCompletion(t *testing.T) {
	tests := []candidatesTest{}

	const (
		record = true
	)
	var (
		filepath = "test-data/test_completion.yaml"
	)

	a := require.New(t)
	yamlFile, err := os.Open(filepath)
	a.NoError(err)

	byteValue, err := io.ReadAll(yamlFile)
	a.NoError(yamlFile.Close())
	a.NoError(err)
	a.NoError(yaml.Unmarshal(byteValue, &tests))

	for i, t := range tests {
		statement, caretLine, caretPosition := getCaretPosition(t.Input)
		getter, lister := buildMockDatabaseMetadataGetterLister()
		results, err := Completion(context.Background(), statement, caretLine, caretPosition, "db", getter, lister)
		a.NoErrorf(err, "Case %02d: %s", i, t.Description)
		if record {
			tests[i].Want = results
		} else {
			a.Equalf(t.Want, results, t.Input, "Case %02d: %s", i, t.Description)
		}
	}

	if record {
		byteValue, err := yaml.Marshal(tests)
		a.NoError(err)
		err = os.WriteFile(filepath, byteValue, 0644)
		a.NoError(err)
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
