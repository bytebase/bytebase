package mysql

import (
	"context"
	"fmt"
	"io"
	"os"
	"slices"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	"gopkg.in/yaml.v3"

	"github.com/bytebase/bytebase/backend/common/yamltest"
	storepb "github.com/bytebase/bytebase/backend/generated-go/store"
	"github.com/bytebase/bytebase/backend/plugin/parser/base"
	"github.com/bytebase/bytebase/backend/store/model"
)

type candidatesTest struct {
	Input string
	Want  []base.Candidate
}

func TestCompletion(t *testing.T) {
	tests := []candidatesTest{}

	const (
		record = false
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
		text, caretLine, caretCol := catchCaretLineColumn(t.Input)
		result, err := base.Completion(context.Background(), storepb.Engine_MYSQL, base.CompletionContext{
			Scene:             base.SceneTypeAll,
			DefaultDatabase:   "db",
			Metadata:          getMetadataForTest,
			ListDatabaseNames: listDatabaseNamesForTest,
		}, text, caretLine, caretCol)
		a.NoError(err)
		var filteredResult []base.Candidate
		for _, r := range result {
			switch r.Type {
			case base.CandidateTypeKeyword, base.CandidateTypeFunction:
				continue
			default:
				filteredResult = append(filteredResult, r)
			}
		}
		slices.SortFunc(filteredResult, func(x, y base.Candidate) int {
			if x.Type != y.Type {
				if x.Type < y.Type {
					return -1
				}
				return 1
			}
			if x.Text != y.Text {
				if x.Text < y.Text {
					return -1
				}
				return 1
			}
			if x.Definition < y.Definition {
				return -1
			} else if x.Definition > y.Definition {
				return 1
			}
			return 0
		})

		if record {
			tests[i].Want = filteredResult
		} else {
			a.Equal(t.Want, filteredResult, t.Input)
		}
	}

	if record {
		yamltest.Record(t, filepath, tests)
	}
}

// Completion cost must be bounded by the caret's statement, not the whole
// sheet: a broken statement anywhere after the caret must not make the
// FROM-clause re-parse loop shrink through the entire trailing document
// (BYT-9886).
func TestCompletionWithBrokenTrailingStatementScalesLinearly(t *testing.T) {
	a := require.New(t)

	// The caret completes through the JOIN alias a2: resolving a2 to t2's
	// columns requires parseTableReferences to actually extract the FROM
	// clause, so an over-truncated (empty) fragment cannot pass this test.
	var sheet strings.Builder
	sheet.WriteString("SELECT a2. FROM t1 JOIN t2 a2 ON t1.c1 = a2.c1;\n")
	sheet.WriteString("SELEC broken FROM oops;\n")
	for i := range 800 {
		fmt.Fprintf(&sheet, "SELECT col_a, col_b, col_c FROM table_%04d WHERE col_a = %d AND col_b LIKE 'pattern%%' ORDER BY col_c LIMIT 100;\n", i, i)
	}

	started := time.Now()
	result, err := base.Completion(context.Background(), storepb.Engine_MYSQL, base.CompletionContext{
		Scene:             base.SceneTypeAll,
		DefaultDatabase:   "db",
		Metadata:          getMetadataForTest,
		ListDatabaseNames: listDatabaseNamesForTest,
	}, sheet.String(), 1, 10 /* caret right after "SELECT a2." */)
	elapsed := time.Since(started)

	a.NoError(err)
	var texts []string
	for _, candidate := range result {
		if candidate.Type == base.CandidateTypeColumn {
			texts = append(texts, candidate.Text)
		}
	}
	a.Contains(texts, "c1")
	a.Contains(texts, "c2")
	a.Less(elapsed, 2*time.Second)
}

func listDatabaseNamesForTest(_ context.Context, _ string) ([]string, error) {
	return []string{"db"}, nil
}

func getMetadataForTest(_ context.Context, _, databaseName string) (string, *model.DatabaseMetadata, error) {
	if databaseName != "db" {
		return "", nil, nil
	}

	return "db", model.NewDatabaseMetadata(&storepb.DatabaseSchemaMetadata{
		Name: databaseName,
		Schemas: []*storepb.SchemaMetadata{
			{
				Name: "",
				Tables: []*storepb.TableMetadata{
					{
						Name: "t1",
						Columns: []*storepb.ColumnMetadata{
							{
								Name: "c1",
							},
						},
					},
					{
						Name: "t2",
						Columns: []*storepb.ColumnMetadata{
							{
								Name: "c1",
							},
							{
								Name: "c2",
							},
						},
					},
				},
				Views: []*storepb.ViewMetadata{
					{
						Name: "v1",
						Definition: `CREATE VIEW v1 AS
						SELECT *
						FROM t1
						`,
					},
				},
			},
		},
	}, nil, nil, storepb.Engine_MYSQL, true /* isObjectCaseSensitive */), nil
}

// catchCaretLineColumn returns the SQL without the caret marker and the
// 1-based line + 0-based column of the caret position. Handles multiline input.
func catchCaretLineColumn(s string) (string, int, int) {
	for i, c := range s {
		if c == '|' {
			text := s[:i] + s[i+1:]
			line := 1
			col := 0
			for _, ch := range s[:i] {
				if ch == '\n' {
					line++
					col = 0
				} else {
					col++
				}
			}
			return text, line, col
		}
	}
	return s, 1, -1
}
