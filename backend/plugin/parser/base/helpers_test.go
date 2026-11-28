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
	require.Equal(t, &storepb.Position{Line: 10, Column: 7}, shiftPosition(&storepb.Position{Line: 1, Column: 3}, basePos))
	require.Equal(t, &storepb.Position{Line: 11, Column: 3}, shiftPosition(&storepb.Position{Line: 2, Column: 3}, basePos))
	require.Nil(t, shiftPosition(nil, basePos))
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
	require.Equal(t, wantCol, omniColumnResource(omniCol))

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

	// A lookup error stands in for the span's not-found error.
	lookupErr := errors.New(`ERROR: column "nope" does not exist (SQLSTATE 42703)`)
	require.Equal(t, lookupErr, OmniResolvedSpan(&omnianalysis.ResolvedSpan{LookupError: lookupErr}).NotFoundError)
}

func TestOmniErrorHelpers(t *testing.T) {
	require.Nil(t, OmniResourceNotFoundError(nil))
	require.NoError(t, OmniFunctionNotSupportedError(nil))
	require.Nil(t, OmniTypeNotSupportedError(nil))
	require.Nil(t, omniUnresolvedColumnsError(nil))

	db := "app"
	col := "secret"
	baseNF := &ResourceNotFoundError{Database: &db, Column: &col}
	wrappedBaseNF := errors.Wrap(baseNF, "lookup failed")
	omniErr := ToOmniNotFoundError(wrappedBaseNF)
	var omniNF *omnianalysis.ResourceNotFoundError
	require.ErrorAs(t, omniErr, &omniNF)
	require.Equal(t, "app", *omniNF.Database)
	require.Equal(t, "secret", *omniNF.Column)
	require.Equal(t, wrappedBaseNF.Error(), omniErr.Error())
	require.ErrorIs(t, omniErr, baseNF)

	other := errors.New("other")
	require.Equal(t, other, ToOmniNotFoundError(other))

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

// changeExtractorFunc is an omni ChangeExtractor in a function.
type changeExtractorFunc func(omnianalysis.ChangeRequest, []omnianalysis.ParsedStatement) (*omnianalysis.Changes, error)

func (f changeExtractorFunc) ExtractChanges(request omnianalysis.ChangeRequest, statements []omnianalysis.ParsedStatement) (*omnianalysis.Changes, error) {
	return f(request, statements)
}

func TestOmniChanges(t *testing.T) {
	dbMetadata := model.NewDatabaseMetadata(&metadatapb.DatabaseSchemaMetadata{Name: "db1"}, nil, nil, storepb.Engine_POSTGRES, true)
	var nilAST *NodeAST[any]
	asts := []AST{&NodeAST[any]{Node: "node", Text: "DELETE FROM t1"}, nilAST, dummyAST{}}
	var gotRequest omnianalysis.ChangeRequest
	var gotStatements []omnianalysis.ParsedStatement
	extract := changeExtractorFunc(func(request omnianalysis.ChangeRequest, statements []omnianalysis.ParsedStatement) (*omnianalysis.Changes, error) {
		gotRequest, gotStatements = request, statements
		return &omnianalysis.Changes{
			Resources: []omnianalysis.ChangedResource{
				{Database: "db1", Schema: "public", Table: "t1", AffectsData: true},
				{Database: "db2", DatabaseOnly: true},
			},
			DMLStatements: []omnianalysis.DMLStatement{{Text: "DELETE FROM t1;"}, {Text: "UPDATE t2 SET a = 1;", SearchPath: []string{"s1"}}},
			DMLCount:      2,
			InsertCount:   3,
		}, nil
	})
	summary, err := omniChanges(extract, "db1", "public", dbMetadata, asts, "the script")
	require.NoError(t, err)

	require.Equal(t, omnianalysis.ChangeRequest{
		DefaultDatabase: "db1",
		DefaultSchema:   "public",
		Database:        &omnianalysis.Database{Name: "db1", Metadata: dbMetadata.GetProto(), ObjectCaseSensitive: true},
		Script:          "the script",
	}, gotRequest)
	// A nil wrapper reaches the engine as a nil node, and a foreign AST as itself.
	require.Equal(t, []omnianalysis.ParsedStatement{
		{Statement: omnianalysis.Statement{Text: "DELETE FROM t1"}, Node: "node"},
		{Node: nil},
		{Node: dummyAST{}},
	}, gotStatements)

	want := model.NewChangedResources(dbMetadata)
	want.AddTable("db1", "public", &storepb.ChangedResourceTable{Name: "t1"}, true)
	want.AddDatabase("db2")
	require.Equal(t, want, summary.ChangedResources)
	require.Equal(t, []string{"DELETE FROM t1;", WithSearchPath("UPDATE t2 SET a = 1;", []string{"s1"})}, summary.DMLStatements)
	require.Equal(t, 2, summary.DMLCount)
	require.Equal(t, 3, summary.InsertCount)

	_, err = omniChanges(changeExtractorFunc(func(request omnianalysis.ChangeRequest, _ []omnianalysis.ParsedStatement) (*omnianalysis.Changes, error) {
		require.Nil(t, request.Database)
		return nil, errors.New("boom")
	}), "db1", "", nil, nil, "")
	require.EqualError(t, err, "boom")
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

	for _, tc := range []struct {
		in   omnianalysis.CandidateType
		want CandidateType
	}{
		{omnianalysis.CandidateTypeNone, CandidateTypeNone},
		{omnianalysis.CandidateKeyword, CandidateTypeKeyword},
		{omnianalysis.CandidateType_, CandidateTypeKeyword},
		{omnianalysis.CandidateShowTarget, CandidateTypeKeyword},
		{omnianalysis.CandidateCatalog, CandidateTypeDatabase},
		{omnianalysis.CandidateDatabase, CandidateTypeDatabase},
		{omnianalysis.CandidateSchema, CandidateTypeSchema},
		{omnianalysis.CandidateTable, CandidateTypeTable},
		{omnianalysis.CandidateSynonym, CandidateTypeTable},
		{omnianalysis.CandidateCollection, CandidateTypeTable},
		{omnianalysis.CandidateView, CandidateTypeView},
		{omnianalysis.CandidateMaterializedView, CandidateTypeMaterializedView},
		{omnianalysis.CandidateColumn, CandidateTypeColumn},
		{omnianalysis.CandidateFunction, CandidateTypeFunction},
		{omnianalysis.CandidateMethod, CandidateTypeFunction},
		{omnianalysis.CandidateCursorMethod, CandidateTypeFunction},
		{omnianalysis.CandidateAggStage, CandidateTypeFunction},
		{omnianalysis.CandidateQueryOperator, CandidateTypeFunction},
		{omnianalysis.CandidateBSONHelper, CandidateTypeFunction},
		{omnianalysis.CandidateDbMethod, CandidateTypeFunction},
		{omnianalysis.CandidateRsMethod, CandidateTypeFunction},
		{omnianalysis.CandidateShMethod, CandidateTypeFunction},
		{omnianalysis.CandidateProcedure, CandidateTypeRoutine},
		{omnianalysis.CandidatePackage, CandidateTypeNone},
		{omnianalysis.CandidateSequence, CandidateTypeSequence},
		{omnianalysis.CandidateIndex, CandidateTypeIndex},
		{omnianalysis.CandidateTrigger, CandidateTypeTrigger},
		{omnianalysis.CandidateEvent, CandidateTypeEvent},
		{omnianalysis.CandidateVariable, CandidateTypeSystemVar},
		{omnianalysis.CandidateCharset, CandidateTypeCharset},
		{omnianalysis.CandidateEngine, CandidateTypeEngine},
		{omnianalysis.CandidatePolicy, CandidateTypeNone},
		{omnianalysis.CandidateExtension, CandidateTypeNone},
		{omnianalysis.CandidateLogin, CandidateTypeNone},
		{omnianalysis.CandidateUser, CandidateTypeNone},
		{omnianalysis.CandidateRole, CandidateTypeNone},
		{omnianalysis.CandidateDatabaseLink, CandidateTypeNone},
		{omnianalysis.CandidateConstraint, CandidateTypeNone},
		{omnianalysis.CandidateType(999), CandidateTypeNone},
	} {
		require.Equal(t, tc.want, OmniCandidateType(tc.in), "type=%s", tc.in)
	}

	type rawCand struct {
		text    string
		typ     omnianalysis.CandidateType
		def     string
		comment string
	}
	converted := ConvertOmniCandidates([]rawCand{
		{text: "ignored", typ: omnianalysis.CandidatePolicy},
		{text: "id", typ: omnianalysis.CandidateColumn, def: "t2.id | int"},
		{text: "id", typ: omnianalysis.CandidateColumn, def: "t1.id | int", comment: "pk"},
		{text: "INT", typ: omnianalysis.CandidateType_},
		{text: "INT", typ: omnianalysis.CandidateKeyword},
	}, func(c rawCand) (string, omnianalysis.CandidateType, string, string) {
		return c.text, c.typ, c.def, c.comment
	}, CompareCandidate)
	require.Equal(t, []Candidate{
		{Type: CandidateTypeColumn, Text: "id", Definition: "t1.id | int", Comment: "pk"},
		{Type: CandidateTypeColumn, Text: "id", Definition: "t2.id | int"},
		{Type: CandidateTypeKeyword, Text: "INT"},
	}, converted)
}

func TestExplainAndDiagnoseHelpers(t *testing.T) {
	other := errors.New("other")

	plan, err := ExplainStatementResult("EXPLAIN SELECT 1", nil, nil)
	require.NoError(t, err)
	require.Equal(t, "EXPLAIN SELECT 1", plan)
	_, err = ExplainStatementResult("", omnianalysis.ErrExplainAnalyzeExecutes, func(error) error { return errors.New("unexpected") })
	require.ErrorIs(t, err, ErrExplainAnalyzeExecutes)
	convertedErr := errors.New("converted")
	_, err = ExplainStatementResult("", other, func(error) error { return convertedErr })
	require.ErrorIs(t, err, convertedErr)
	_, err = ExplainStatementResult("", other, nil)
	require.ErrorIs(t, err, other)

	require.Equal(t, "SELECT 1 LIMIT 5", RewriteResultLimit("SELECT 1", 5, func(string, int) (string, error) {
		return "SELECT 1 LIMIT 5", nil
	}, "FALLBACK %s %d"))
	require.Equal(t, "SELECT * FROM (SELECT 1) LIMIT 5", RewriteResultLimit("  SELECT 1; ", 5, func(string, int) (string, error) {
		return "", errors.New("cannot rewrite")
	}, "SELECT * FROM (%s) LIMIT %d"))

	require.Empty(t, DiagnoseSyntaxError("SELECT 1", nil))
	diags := ConvertOffsetDiagnostics("SELECT\n  *", []int{9}, func(off int) (int, string) {
		return off, "bad token"
	})
	require.Len(t, diags, 1)
	require.Equal(t, uint32(1), diags[0].Range.Start.Line)
	require.Equal(t, uint32(2), diags[0].Range.Start.Character)

	type rawDiag struct {
		text   string
		offset int
		msg    string
	}
	nilDiags := DiagnoseFirstError[rawDiag]("SELECT 1", nil, func(d *rawDiag) (string, int, string) {
		return d.text, d.offset, d.msg
	})
	require.NotNil(t, nilDiags)
	require.Empty(t, nilDiags)

	firstDiags := DiagnoseFirstError("SELECT\n  *", &rawDiag{text: "SELECT\n  *", offset: 9, msg: "bad token"}, func(d *rawDiag) (string, int, string) {
		return d.text, d.offset, d.msg
	})
	require.Len(t, firstDiags, 1)
	require.Equal(t, uint32(1), firstDiags[0].Range.Start.Line)
	require.Equal(t, uint32(2), firstDiags[0].Range.Start.Character)
	require.Equal(t, "bad token", firstDiags[0].Message)
}

func TestNodeASTAndExtractNode(t *testing.T) {
	pos := &storepb.Position{Line: 3, Column: 2}
	wrapped := &NodeAST[string]{
		Node:          "select-node",
		Text:          "SELECT 1;",
		StartPosition: pos,
	}
	var a AST = wrapped
	require.Equal(t, pos, a.ASTStartPosition())
	require.Equal(t, "SELECT 1;", wrapped.ASTSource())

	node, ok := ExtractNode[string](a)
	require.True(t, ok)
	require.Equal(t, "select-node", node)

	// Nil interface, typed nil pointer, and wrong AST type all return (zero, false).
	_, ok = ExtractNode[string](nil)
	require.False(t, ok)

	var typedNil *NodeAST[string]
	_, ok = ExtractNode[string](typedNil)
	require.False(t, ok)

	_, ok = ExtractNode[int](a)
	require.False(t, ok)

	_, ok = ExtractNode[string](dummyAST{})
	require.False(t, ok)
}
