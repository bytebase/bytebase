package base

import (
	"context"
	"testing"

	omnianalysis "github.com/bytebase/omni/analysis"
	metadatapb "github.com/bytebase/omni/metadata"
	"github.com/pkg/errors"
	"github.com/stretchr/testify/require"

	storepb "github.com/bytebase/bytebase/backend/generated-go/store"
	"github.com/bytebase/bytebase/backend/store/model"
)

func TestByteOffsetToRunePositionAndShift(t *testing.T) {
	sql := "ab\n€d"
	require.Equal(t, &storepb.Position{Line: 1, Column: 1}, ByteOffsetToRunePosition(sql, -5))
	// Offset 4 lands mid-rune inside '€' (3 bytes at offset 3..6).
	require.Equal(t, &storepb.Position{Line: 2, Column: 1}, ByteOffsetToRunePosition(sql, 4))
	require.Equal(t, &storepb.Position{Line: 2, Column: 2}, ByteOffsetToRunePosition(sql, 6))
	require.Equal(t, &storepb.Position{Line: 2, Column: 3}, ByteOffsetToRunePosition(sql, 999))

	basePos := &storepb.Position{Line: 10, Column: 5}
	require.Equal(t, &storepb.Position{Line: 10, Column: 7}, ShiftPosition(&storepb.Position{Line: 1, Column: 3}, basePos))
	require.Equal(t, &storepb.Position{Line: 11, Column: 3}, ShiftPosition(&storepb.Position{Line: 2, Column: 3}, basePos))
	require.Nil(t, ShiftPosition(nil, basePos))
}

func TestBuildStatementsAndMapByteRangesToLSP(t *testing.T) {
	sql := "SELECT 1; SELECT '🚀';"
	type seg struct {
		start, end int
		empty      bool
	}
	segs := []seg{{0, 8, false}, {9, len(sql) - 1, false}}

	withSemi := BuildStatements(sql, segs, func(s seg) (int, int, bool) { return s.start, s.end, s.empty }, true)
	require.Len(t, withSemi, 2)
	require.Equal(t, "SELECT 1;", withSemi[0].Text)
	require.Equal(t, " SELECT '🚀';", withSemi[1].Text)

	exact := BuildStatements(sql, segs, func(s seg) (int, int, bool) { return s.start, s.end, s.empty }, false)
	require.Equal(t, "SELECT 1", exact[0].Text)
	require.Nil(t, BuildStatements(sql, []seg(nil), func(s seg) (int, int, bool) { return s.start, s.end, s.empty }, false))

	omniStmts := OmniStatements(sql, []omnianalysis.Statement{{Start: 0, End: 9}, {Start: 9, End: len(sql)}})
	require.Len(t, omniStmts, 2)
	require.Equal(t, "SELECT 1;", omniStmts[0].Text)
	require.Equal(t, " SELECT '🚀';", omniStmts[1].Text)

	require.Nil(t, MapByteRangesToLSP(sql, nil))
	lspRanges := MapByteRangesToLSP(sql, []omnianalysis.ByteRange{
		{Start: 10, End: len(sql) - 1},
		{Start: 0, End: 8},
		{Start: 5, End: 2},
	})
	require.Len(t, lspRanges, 2)
	require.Equal(t, uint32(10), lspRanges[0].Start.Character)
	// Surrogate pair '🚀' adds 2 UTF-16 code units.
	require.Equal(t, uint32(21), lspRanges[0].End.Character)
	require.Equal(t, uint32(0), lspRanges[1].Start.Character)
	require.Equal(t, uint32(8), lspRanges[1].End.Character)
}

func TestOmniSpanHelpers(t *testing.T) {
	require.Equal(t, Select, OmniQueryType(omnianalysis.Select))
	require.Equal(t, Explain, OmniQueryType(omnianalysis.Explain))
	require.Equal(t, SelectInfoSchema, OmniQueryType(omnianalysis.SelectInfoSchema))
	require.Equal(t, DDL, OmniQueryType(omnianalysis.DDL))
	require.Equal(t, DML, OmniQueryType(omnianalysis.DML))
	require.Equal(t, QueryTypeUnknown, OmniQueryType(omnianalysis.QueryType(99)))

	omniCol := omnianalysis.ColumnResource{
		Instance: "inst",
		Server:   "srv",
		Database: "db",
		Schema:   "public",
		Table:    "users",
		Column:   "id",
	}
	wantCol := ColumnResource{
		Instance: "inst",
		Server:   "srv",
		Database: "db",
		Schema:   "public",
		Table:    "users",
		Column:   "id",
	}
	require.Equal(t, wantCol, OmniColumnResource(omniCol))

	require.Nil(t, OmniColumnSet(nil))
	require.Equal(t, SourceColumnSet{wantCol: true}, OmniColumnSet(omnianalysis.ColumnSet{omniCol: true}))

	require.Nil(t, OmniQuerySpanResults(nil))
	omniResults := []omnianalysis.QuerySpanResult{{
		Name:           "id",
		SourceColumns:  omnianalysis.ColumnSet{omniCol: true},
		IsPlainField:   true,
		SelectAsterisk: true,
		Invisible:      true,
	}}
	require.Equal(t, []QuerySpanResult{{
		Name:           "id",
		SourceColumns:  SourceColumnSet{wantCol: true},
		IsPlainField:   true,
		SelectAsterisk: true,
		Invisible:      true,
	}}, OmniQuerySpanResults(omniResults))

	require.Nil(t, OmniResolvedSpan(nil))
	dbName := "db"
	tblName := "missing"
	resolved := OmniResolvedSpan(&omnianalysis.ResolvedSpan{
		Type:             omnianalysis.Select,
		ExplainAnalyze:   true,
		SourceColumns:    omnianalysis.ColumnSet{{Database: "db", Table: "users"}: true},
		Results:          omniResults,
		PredicateColumns: omnianalysis.ColumnSet{omniCol: true},
		NotFoundError: &omnianalysis.ResourceNotFoundError{
			Database: &dbName,
			Table:    &tblName,
		},
		UnresolvedColumnsError: &omnianalysis.UnresolvedColumnsError{
			Relations: []omnianalysis.ColumnResource{{Database: "db", Table: "empty"}},
		},
	})
	require.Equal(t, Select, resolved.Type)
	require.True(t, resolved.ExplainAnalyze)
	require.Equal(t, SourceColumnSet{{Database: "db", Table: "users"}: true}, resolved.SourceColumns)
	require.Equal(t, SourceColumnSet{wantCol: true}, resolved.PredicateColumns)
	require.Len(t, resolved.Results, 1)
	var nf *ResourceNotFoundError
	require.ErrorAs(t, resolved.NotFoundError, &nf)
	require.Equal(t, "db", *nf.Database)
	require.Equal(t, "missing", *nf.Table)
	require.Equal(t, &UnresolvedColumnsError{Relations: []ColumnResource{{Database: "db", Table: "empty"}}}, resolved.UnresolvedColumnsError)

	// Nil NotFoundError must leave QuerySpan.NotFoundError as an untyped nil error.
	noErrSpan := OmniResolvedSpan(&omnianalysis.ResolvedSpan{Type: omnianalysis.DML})
	require.NoError(t, noErrSpan.NotFoundError)
	require.Nil(t, noErrSpan.UnresolvedColumnsError)
}

func TestOmniErrorHelpers(t *testing.T) {
	require.Nil(t, OmniResourceNotFoundError(nil))
	require.NoError(t, OmniNotFoundError(nil))
	require.NoError(t, OmniFunctionNotSupportedError(nil))
	require.Nil(t, OmniTypeNotSupportedError(nil))
	require.Nil(t, OmniUnresolvedColumnsError(nil))

	db := "app"
	col := "secret"
	baseNF := &ResourceNotFoundError{Database: &db, Column: &col}
	omniErr := ToOmniNotFoundError(baseNF)
	var omniNF *omnianalysis.ResourceNotFoundError
	require.ErrorAs(t, omniErr, &omniNF)
	require.Equal(t, "app", *omniNF.Database)
	require.Equal(t, "secret", *omniNF.Column)

	backErr := OmniNotFoundError(errors.Wrap(omniErr, "wrapped"))
	var roundTripped *ResourceNotFoundError
	require.ErrorAs(t, backErr, &roundTripped)
	require.Equal(t, "app", *roundTripped.Database)
	require.Equal(t, "secret", *roundTripped.Column)

	other := errors.New("other")
	require.Equal(t, other, ToOmniNotFoundError(other))
	require.Equal(t, other, OmniNotFoundError(other))

	fnErr := OmniFunctionNotSupportedError(&omnianalysis.FunctionNotSupportedError{
		Function:   "my_fn",
		Definition: "SELECT 1",
	})
	var baseFnErr *FunctionNotSupportedError
	require.ErrorAs(t, fnErr, &baseFnErr)
	require.Equal(t, "my_fn", baseFnErr.Function)
	require.Equal(t, "SELECT 1", baseFnErr.Definition)
	require.Equal(t, other, OmniFunctionNotSupportedError(other))

	typeErr := OmniTypeNotSupportedError(&omnianalysis.TypeNotSupportedError{Type: "PIVOT", Name: "p", Extra: "x"})
	require.Equal(t, &TypeNotSupportedError{Type: "PIVOT", Name: "p", Extra: "x"}, typeErr)

	require.Equal(t, MixUserSystemTablesError, ConvertOmniMixTablesError(omnianalysis.ErrMixUserSystemTables))
	wrappedMix := errors.Wrap(omnianalysis.ErrMixUserSystemTables, "in cte")
	convertedMix := ConvertOmniMixTablesError(wrappedMix)
	require.ErrorIs(t, convertedMix, MixUserSystemTablesError)
	require.ErrorIs(t, convertedMix, wrappedMix)
	require.NoError(t, ConvertOmniMixTablesError(other))
}

type dummyAST struct{}

func (dummyAST) ASTStartPosition() *storepb.Position { return nil }

func TestOmniResourceChanges(t *testing.T) {
	summary, err := ExtractChangedResourcesFromASTs(nil, []AST{dummyAST{}}, true, func(_ AST) (omnianalysis.StatementChanges, bool, error) {
		return omnianalysis.StatementChanges{
			Resources: []omnianalysis.ChangedResource{
				{Database: "db1", Schema: "public", Table: "t1", AffectsData: true},
				{Database: "db1", Schema: "public", Table: ""},
				{Database: "db2", DatabaseOnly: true},
			},
			DMLStatements: []string{"DELETE FROM t1"},
			DMLCount:      1,
			InsertCount:   2,
		}, true, nil
	})
	require.NoError(t, err)
	require.Equal(t, 1, summary.DMLCount)
	require.Equal(t, 2, summary.InsertCount)
	require.Equal(t, []string{"DELETE FROM t1"}, summary.DMLStatements)
	require.Equal(t, []string{"db2"}, summary.ChangedResources.GetDatabaseOnlyTargets())
	built := summary.ChangedResources.Build()
	require.Len(t, built.GetDatabases(), 1)
	require.Equal(t, "db1", built.GetDatabases()[0].GetName())
	require.Len(t, built.GetDatabases()[0].GetSchemas()[0].GetTables(), 1)
	require.Equal(t, "t1", built.GetDatabases()[0].GetSchemas()[0].GetTables()[0].GetName())
}

func TestCompletionHelpers(t *testing.T) {
	loadNil, listNil := CompletionMetadataLoaders(CompletionContext{})
	require.Nil(t, loadNil)
	require.Nil(t, listNil)

	wantProto := &metadatapb.DatabaseSchemaMetadata{Name: "appdb"}
	cCtx := CompletionContext{
		InstanceID: "inst-1",
		Metadata: func(_ context.Context, instanceID, databaseName string) (string, *model.DatabaseMetadata, error) {
			require.Equal(t, "inst-1", instanceID)
			if databaseName == "appdb" {
				return "appdb", model.NewDatabaseMetadata(wantProto, nil, nil, storepb.Engine_POSTGRES, false), nil
			}
			return "", nil, nil
		},
		ListDatabaseNames: func(_ context.Context, instanceID string) ([]string, error) {
			require.Equal(t, "inst-1", instanceID)
			return []string{"appdb"}, nil
		},
	}
	loadMeta, listNames := CompletionMetadataLoaders(cCtx)
	gotProto, err := loadMeta(context.Background(), "appdb")
	require.NoError(t, err)
	require.Equal(t, wantProto, gotProto)
	missingProto, err := loadMeta(context.Background(), "missing")
	require.NoError(t, err)
	require.Nil(t, missingProto)
	names, err := listNames(context.Background())
	require.NoError(t, err)
	require.Equal(t, []string{"appdb"}, names)

	cands := DeduplicateCandidates([]string{"b", "a", "b"}, func(s string) (Candidate, bool) {
		return Candidate{Type: CandidateTypeColumn, Text: s}, true
	}, CompareCandidate)
	require.Equal(t, []Candidate{
		{Type: CandidateTypeColumn, Text: "a"},
		{Type: CandidateTypeColumn, Text: "b"},
	}, cands)
}

func TestExplainAndDiagnoseHelpers(t *testing.T) {
	require.ErrorIs(t, OmniExplainAnalyzeError(omnianalysis.ErrExplainAnalyzeExecutes), ErrExplainAnalyzeExecutes)
	other := errors.New("other")
	require.Equal(t, other, OmniExplainAnalyzeError(other))

	require.Empty(t, DiagnoseSyntaxError("SELECT 1", nil))
	diags := ConvertOffsetDiagnostics("SELECT\n  *", []int{9}, func(off int) (int, string) {
		return off, "bad token"
	})
	require.Len(t, diags, 1)
	require.Equal(t, uint32(1), diags[0].Range.Start.Line)
	require.Equal(t, uint32(2), diags[0].Range.Start.Character)
}
