package pg

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
		text, caretOffset := catchCaret(t.Input)
		result, err := base.Completion(context.Background(), storepb.Engine_POSTGRES, base.CompletionContext{
			Scene:             base.SceneTypeAll,
			DefaultDatabase:   "db",
			Metadata:          getMetadataForTest,
			ListDatabaseNames: listDatbaseNamesForTest,
		}, text, 1, caretOffset)
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

func listDatbaseNamesForTest(_ context.Context, _ string) ([]string, error) {
	return []string{"db"}, nil
}

// Completion cost must be bounded by the caret's statement, not the whole
// sheet: a broken statement anywhere after the caret must not make the
// FROM-clause re-parse loop shrink through the entire trailing document
// (BYT-9886).
func TestCompletionWithBrokenTrailingStatementScalesLinearly(t *testing.T) {
	a := require.New(t)

	// The caret is unqualified, so column candidates can only come from the
	// tables parseTableReferences extracts out of the FROM clause; an
	// over-truncated (empty) fragment cannot pass this test.
	var sheet strings.Builder
	sheet.WriteString("SELECT  FROM t1 JOIN t2 a2 ON t1.c1 = a2.c1;\n")
	sheet.WriteString("SELEC broken FROM oops;\n")
	for i := range 2000 {
		fmt.Fprintf(&sheet, "SELECT col_a, col_b, col_c FROM table_%04d WHERE col_a = %d AND col_b LIKE 'pattern%%' ORDER BY col_c LIMIT 100;\n", i, i)
	}

	started := time.Now()
	result, err := base.Completion(context.Background(), storepb.Engine_POSTGRES, base.CompletionContext{
		Scene:             base.SceneTypeAll,
		DefaultDatabase:   "db",
		Metadata:          getMetadataForTest,
		ListDatabaseNames: listDatbaseNamesForTest,
	}, sheet.String(), 1, 7 /* caret in the select list, right after "SELECT " */)
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

func getMetadataForTest(_ context.Context, _, databaseName string) (string, *model.DatabaseMetadata, error) {
	if databaseName != "db" {
		return "", nil, nil
	}

	return "db", model.NewDatabaseMetadata(&storepb.DatabaseSchemaMetadata{
		Name: databaseName,
		Schemas: []*storepb.SchemaMetadata{
			{
				Name: "public",
				Tables: []*storepb.TableMetadata{
					{
						Name: "t1",
						Columns: []*storepb.ColumnMetadata{
							{
								Name: "c1",
								Type: "int",
							},
						},
					},
					{
						Name: "t2",
						Columns: []*storepb.ColumnMetadata{
							{
								Name: "c1",
								Type: "int",
							},
							{
								Name: "c2",
								Type: "int",
							},
						},
					},
				},
				Views: []*storepb.ViewMetadata{
					{
						Name:       "v1",
						Definition: `SELECT * FROM t1`,
					},
				},
				ExternalTables: []*storepb.ExternalTableMetadata{
					{
						Name: "ft1",
						Columns: []*storepb.ColumnMetadata{
							{
								Name: "f1",
								Type: "int",
							},
							{
								Name: "f2",
								Type: "varchar",
							},
						},
					},
				},
				MaterializedViews: []*storepb.MaterializedViewMetadata{
					{
						Name:       "mv1",
						Definition: "SELECT c1, c2 FROM t2",
					},
				},
				Sequences: []*storepb.SequenceMetadata{
					{
						Name:      "seq1",
						DataType:  "bigint",
						Start:     "1",
						Increment: "1",
						MinValue:  "1",
						MaxValue:  "9223372036854775807",
					},
					{
						Name:      "user_id_seq",
						DataType:  "bigint",
						Start:     "1",
						Increment: "1",
						MinValue:  "1",
						MaxValue:  "9223372036854775807",
					},
				},
			},
			{
				Name: "test",
				Tables: []*storepb.TableMetadata{
					{
						Name: "auto",
						Columns: []*storepb.ColumnMetadata{
							{
								Name: "id",
								Type: "int",
							},
							{
								Name: "name",
								Type: "varchar",
							},
						},
					},
					{
						Name: "users",
						Columns: []*storepb.ColumnMetadata{
							{
								Name: "user_id",
								Type: "int",
							},
							{
								Name: "username",
								Type: "varchar",
							},
						},
					},
				},
				Sequences: []*storepb.SequenceMetadata{
					{
						Name:      "order_id_seq",
						DataType:  "bigint",
						Start:     "1",
						Increment: "1",
						MinValue:  "1",
						MaxValue:  "9223372036854775807",
					},
				},
			},
		},
	}, nil, nil, storepb.Engine_POSTGRES, true /* isObjectCaseSensitive */), nil
}

func TestCompletionQuotedIdentifiers(t *testing.T) {
	tests := []candidatesTest{}

	const (
		record = false
	)
	var (
		filepath = "test-data/test_completion_quoted_identifiers.yaml"
	)

	a := require.New(t)
	yamlFile, err := os.Open(filepath)
	a.NoError(err)

	byteValue, err := io.ReadAll(yamlFile)
	a.NoError(yamlFile.Close())
	a.NoError(err)
	a.NoError(yaml.Unmarshal(byteValue, &tests))

	for i, t := range tests {
		text, caretOffset := catchCaret(t.Input)
		result, err := base.Completion(context.Background(), storepb.Engine_POSTGRES, base.CompletionContext{
			Scene:             base.SceneTypeAll,
			DefaultDatabase:   "db",
			Metadata:          getQuotedIdentifierMetadataForTest,
			ListDatabaseNames: listDatbaseNamesForTest,
		}, text, 1, caretOffset)
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

func getQuotedIdentifierMetadataForTest(_ context.Context, _, databaseName string) (string, *model.DatabaseMetadata, error) {
	if databaseName != "db" {
		return "", nil, nil
	}

	return "db", model.NewDatabaseMetadata(&storepb.DatabaseSchemaMetadata{
		Name: databaseName,
		Schemas: []*storepb.SchemaMetadata{
			{
				Name: "public",
				Tables: []*storepb.TableMetadata{
					{
						Name: "t1",
						Columns: []*storepb.ColumnMetadata{
							{
								Name: "c1",
								Type: "int",
							},
						},
					},
					{
						Name: "order",
						Columns: []*storepb.ColumnMetadata{
							{
								Name: "id",
								Type: "int",
							},
							{
								Name: "amount",
								Type: "numeric",
							},
						},
					},
					{
						Name: "MyTable",
						Columns: []*storepb.ColumnMetadata{
							{
								Name: "Id",
								Type: "int",
							},
							{
								Name: "Value",
								Type: "text",
							},
						},
					},
					{
						Name: "my-table",
						Columns: []*storepb.ColumnMetadata{
							{
								Name: "col1",
								Type: "int",
							},
						},
					},
				},
			},
		},
	}, nil, nil, storepb.Engine_POSTGRES, true /* isObjectCaseSensitive */), nil
}

func catchCaret(s string) (string, int) {
	for i, c := range s {
		if c == '|' {
			return s[:i] + s[i+1:], i
		}
	}
	return s, -1
}
