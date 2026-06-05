package trino

import (
	"context"
	"strings"
	"testing"

	"github.com/pkg/errors"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	storepb "github.com/bytebase/bytebase/backend/generated-go/store"
	"github.com/bytebase/bytebase/backend/plugin/parser/base"
	"github.com/bytebase/bytebase/backend/store/model"
)

// TestCompletion exercises the omni-backed Trino completer through the public
// Completion entry point.
//
// DIVERGENCE FROM THE LEGACY ANTLR COMPLETER (intentional, documented):
// The previous Completion was a ~1730-line CodeCompletionCore (c3) port. The
// omni Trino completer is explicitly NOT a c3 port (see the omni completion
// package doc) — it is a self-contained, lexer+context completer. Three
// behavioural differences fall out of that and out of Trino's identifier rules:
//
//  1. Identifier case. Trino folds unquoted identifiers to lower case, and the
//     omni catalog stores/returns the normalized (lower-case) form. So a table
//     stored as "Employees" is offered as "employees". The legacy completer
//     preserved the metadata's original case.
//  2. No Definition / Priority. omni candidates carry only {Text, Type}; the
//     legacy "catalog.schema.table | type, NOT NULL" Definition string and the
//     priority ranking are not produced, so those fields are empty.
//  3. Candidate set in column / FROM contexts. omni resolves in-scope columns
//     via query-span analysis against the *current* catalog/schema (set here
//     from DefaultDatabase/DefaultSchema). The legacy completer additionally
//     surfaced every column of the default schema ranked by priority; omni does
//     not.
//
// These cases therefore assert the candidate *set* (Text, Type) the omni
// completer actually returns, rather than the legacy YAML fixture.
func TestCompletion(t *testing.T) {
	type want struct {
		text string
		typ  base.CandidateType
	}
	tests := []struct {
		description string
		input       string // caret marked by "|"
		// wantPresent candidates that MUST appear.
		wantPresent []want
		// wantAbsentText candidate texts that must NOT appear (any type).
		wantAbsentText []string
	}{
		{
			description: "FROM offers catalogs, schemas, and current-schema tables",
			input:       "SELECT * FROM |",
			wantPresent: []want{
				{"company", base.CandidateTypeDatabase},
				{"school", base.CandidateTypeDatabase},
				{"dbo", base.CandidateTypeSchema},
				{"myschema", base.CandidateTypeSchema},
				{"address", base.CandidateTypeTable},
				{"employees", base.CandidateTypeTable},
			},
		},
		{
			description: "Schema-qualified FROM offers that schema's tables",
			input:       "SELECT * FROM dbo.|",
			wantPresent: []want{
				{"address", base.CandidateTypeTable},
				{"employees", base.CandidateTypeTable},
			},
			wantAbsentText: []string{"salarylevel"},
		},
		{
			description: "Schema-qualified FROM for a non-current schema",
			input:       "SELECT * FROM myschema.|",
			wantPresent: []want{
				{"salarylevel", base.CandidateTypeTable},
			},
		},
		{
			description: "SELECT column context offers in-scope columns",
			input:       "SELECT | FROM Employees",
			wantPresent: []want{
				{"id", base.CandidateTypeColumn},
				{"name", base.CandidateTypeColumn},
			},
		},
		{
			description: "WHERE column context offers in-scope columns",
			input:       "SELECT * FROM Employees WHERE |",
			wantPresent: []want{
				{"id", base.CandidateTypeColumn},
				{"name", base.CandidateTypeColumn},
			},
		},
		{
			description: "Alias-qualified column context",
			input:       "SELECT tableAlias.| FROM Employees AS tableAlias",
			wantPresent: []want{
				{"id", base.CandidateTypeColumn},
				{"name", base.CandidateTypeColumn},
			},
		},
		{
			description: "CTE name offered as a relation after FROM",
			input:       "WITH MyCTE_01 AS (SELECT * FROM dbo.Employees) SELECT * FROM |",
			wantPresent: []want{
				{"mycte_01", base.CandidateTypeTable},
			},
		},
	}

	a := require.New(t)
	for _, test := range tests {
		t.Run(test.description, func(t *testing.T) {
			statement, caretLine, caretOffset := getCaretPosition(test.input)
			getter, lister := buildMockDatabaseMetadataGetterLister()
			results, err := Completion(context.Background(), base.CompletionContext{
				Scene:             base.SceneTypeAll,
				DefaultDatabase:   "Company",
				DefaultSchema:     "dbo",
				Metadata:          getter,
				ListDatabaseNames: lister,
			}, statement, caretLine, caretOffset)
			a.NoErrorf(err, "%s", test.description)

			for _, w := range test.wantPresent {
				assert.Truef(t, hasCandidate(results, w.text, w.typ),
					"%s: expected candidate {%s, %s}; got %v", test.description, w.text, w.typ, candidateSummary(results))
			}
			for _, text := range test.wantAbsentText {
				for _, r := range results {
					assert.NotEqualf(t, text, r.Text, "%s: candidate %q should be absent", test.description, text)
				}
			}
		})
	}
}

// TestCompletionNilMetadata verifies Completion does not panic and returns only
// keyword/CTE candidates (no object candidates) when no metadata is wired.
func TestCompletionNilMetadata(t *testing.T) {
	results, err := Completion(context.Background(), base.CompletionContext{
		Scene: base.SceneTypeAll,
	}, "SELECT * FROM ", 1, len("SELECT * FROM "))
	require.NoError(t, err)
	for _, r := range results {
		switch r.Type {
		case base.CandidateTypeDatabase, base.CandidateTypeSchema, base.CandidateTypeTable, base.CandidateTypeView, base.CandidateTypeColumn:
			t.Errorf("nil metadata produced object candidate {%s, %s}", r.Text, r.Type)
		default:
		}
	}
}

// TestCompletionLoadsOnlyNeededCatalogs verifies that completion loads full
// metadata only for the default catalog and catalogs named in the statement, not
// for every catalog on the instance. On a Trino instance federating many
// catalogs, eager loading would fan out a metadata fetch per keystroke.
func TestCompletionLoadsOnlyNeededCatalogs(t *testing.T) {
	metas := []*storepb.DatabaseSchemaMetadata{
		{Name: "Company", Schemas: []*storepb.SchemaMetadata{{Name: "dbo", Tables: []*storepb.TableMetadata{{Name: "Employees", Columns: []*storepb.ColumnMetadata{{Name: "Id", Type: "int"}}}}}}},
		{Name: "School", Schemas: []*storepb.SchemaMetadata{{Name: "sch", Tables: []*storepb.TableMetadata{{Name: "Student", Columns: []*storepb.ColumnMetadata{{Name: "Id", Type: "int"}}}}}}},
		{Name: "Warehouse", Schemas: []*storepb.SchemaMetadata{{Name: "wh", Tables: []*storepb.TableMetadata{{Name: "Item", Columns: []*storepb.ColumnMetadata{{Name: "Id", Type: "int"}}}}}}},
	}
	loads := map[string]int{}
	getter := func(_ context.Context, _, databaseName string) (string, *model.DatabaseMetadata, error) {
		loads[databaseName]++
		for _, m := range metas {
			if m.Name == databaseName {
				return "", model.NewDatabaseMetadata(m, nil, nil, storepb.Engine_MYSQL, false /* isObjectCaseSensitive */), nil
			}
		}
		return "", nil, errors.Errorf("database %q not found", databaseName)
	}
	lister := func(context.Context, string) ([]string, error) {
		return []string{"Company", "School", "Warehouse"}, nil
	}
	cCtx := base.CompletionContext{
		Scene:             base.SceneTypeAll,
		DefaultDatabase:   "Company",
		DefaultSchema:     "dbo",
		Metadata:          getter,
		ListDatabaseNames: lister,
	}

	// Completing within the default catalog must not load School or Warehouse.
	stmt := "SELECT * FROM "
	_, err := Completion(context.Background(), cCtx, stmt, 1, len(stmt))
	require.NoError(t, err)
	assert.Zerof(t, loads["School"], "unreferenced catalog School must not be loaded; loads=%v", loads)
	assert.Zerof(t, loads["Warehouse"], "unreferenced catalog Warehouse must not be loaded; loads=%v", loads)
	assert.NotZerof(t, loads["Company"], "default catalog Company should be loaded; loads=%v", loads)

	// Naming a catalog in the statement loads it — and still not the others.
	for k := range loads {
		delete(loads, k)
	}
	stmt = "SELECT * FROM School."
	_, err = Completion(context.Background(), cCtx, stmt, 1, len(stmt))
	require.NoError(t, err)
	assert.NotZerof(t, loads["School"], "referenced catalog School should be loaded; loads=%v", loads)
	assert.Zerof(t, loads["Warehouse"], "still-unreferenced catalog Warehouse must not be loaded; loads=%v", loads)
}

func hasCandidate(cands []base.Candidate, text string, typ base.CandidateType) bool {
	for _, c := range cands {
		if c.Text == text && c.Type == typ {
			return true
		}
	}
	return false
}

func candidateSummary(cands []base.Candidate) []string {
	out := make([]string, 0, len(cands))
	for _, c := range cands {
		if c.Type == base.CandidateTypeKeyword {
			continue
		}
		out = append(out, string(c.Type)+":"+c.Text)
	}
	return out
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

var databaseMetadatas = []*storepb.DatabaseSchemaMetadata{
	{
		Name: "Company",
		Schemas: []*storepb.SchemaMetadata{
			{
				Name: "dbo",
				Tables: []*storepb.TableMetadata{
					{
						Name: "Employees",
						Columns: []*storepb.ColumnMetadata{
							{Name: "Id", Type: "int"},
							{Name: "Name", Type: "varchar"},
						},
					},
					{
						Name: "Address",
						Columns: []*storepb.ColumnMetadata{
							{Name: "EmployeeId", Type: "int"},
							{Name: "Street", Type: "varchar"},
						},
					},
				},
			},
			{
				Name: "MySchema",
				Tables: []*storepb.TableMetadata{
					{
						Name: "SalaryLevel",
						Columns: []*storepb.ColumnMetadata{
							{Name: "Id", Type: "int"},
							{Name: "SalaryUpBound", Type: "int"},
						},
					},
				},
			},
		},
	},
	{
		Name: "School",
		Schemas: []*storepb.SchemaMetadata{
			{
				Name: "dbo",
				Tables: []*storepb.TableMetadata{
					{
						Name: "Student",
						Columns: []*storepb.ColumnMetadata{
							{Name: "Id", Type: "int"},
							{Name: "ParentName", Type: "varchar"},
						},
					},
				},
			},
		},
	},
}

func buildMockDatabaseMetadataGetterLister() (base.GetDatabaseMetadataFunc, base.ListDatabaseNamesFunc) {
	return func(_ context.Context, _, databaseName string) (string, *model.DatabaseMetadata, error) {
			m := make(map[string]*model.DatabaseMetadata)
			for _, metadata := range databaseMetadatas {
				m[metadata.Name] = model.NewDatabaseMetadata(metadata, nil, nil, storepb.Engine_MYSQL, false /* isObjectCaseSensitive */)
			}

			if databaseMetadata, ok := m[databaseName]; ok {
				return "", databaseMetadata, nil
			}

			return "", nil, errors.Errorf("database %q not found", databaseName)
		}, func(context.Context, string) ([]string, error) {
			var names []string
			for _, metadata := range databaseMetadatas {
				names = append(names, metadata.Name)
			}
			return names, nil
		}
}
