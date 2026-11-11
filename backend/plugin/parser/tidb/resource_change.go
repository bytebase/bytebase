package tidb

import (
	"slices"
	"strings"
	"unicode"

	omniast "github.com/bytebase/omni/tidb/ast"
	"github.com/pkg/errors"

	storepb "github.com/bytebase/bytebase/backend/generated-go/store"
	"github.com/bytebase/bytebase/backend/plugin/parser/base"
	"github.com/bytebase/bytebase/backend/store/model"
	"github.com/bytebase/bytebase/backend/utils"
)

func init() {
	base.RegisterExtractChangedResourcesFunc(storepb.Engine_TIDB, extractChangedResources)
}

func extractChangedResources(database string, _ string, dbMetadata *model.DatabaseMetadata, asts []base.AST, _ string) (*base.ChangeSummary, error) {
	summary := &base.ChangeSummary{
		ChangedResources: model.NewChangedResources(dbMetadata),
	}
	for _, a := range asts {
		node, ok := GetNode(a)
		if !ok {
			return nil, errors.New("expected TiDB AST")
		}
		recordResourceChanges(database, node, summary.ChangedResources)
		if wrapped, ok := a.(*AST); ok {
			countDML(summary, wrapped)
		}
	}
	return summary, nil
}

// recordResourceChanges records the tables and databases a statement changes.
//
// A table-level DDL records the table it changes (DROP/ALTER/TRUNCATE also mark
// its data affected). A DML statement records the tables it writes. A non-table
// object DDL (view/sequence) records only its target database, and only when its
// name carries an explicit qualifier (SUP-222 / BYT-9698); an unqualified name
// records nothing and falls back to the request database.
func recordResourceChanges(database string, node omniast.Node, changedResources *model.ChangedResources) {
	addTable := func(ref *omniast.TableRef, affectsData bool) {
		if ref == nil {
			return
		}
		d := ref.Schema
		if d == "" {
			d = database
		}
		changedResources.AddTable(d, "", &storepb.ChangedResourceTable{Name: ref.Name}, affectsData)
	}
	objectDatabase := func(ref *omniast.TableRef) {
		if ref != nil && ref.Schema != "" {
			changedResources.AddDatabase(ref.Schema)
		}
	}

	switch n := node.(type) {
	case *omniast.CreateTableStmt:
		addTable(n.Table, false)
	case *omniast.DropTableStmt:
		for _, name := range n.Tables {
			addTable(name, true)
		}
	case *omniast.DropViewStmt:
		for _, view := range n.Views {
			objectDatabase(view)
		}
	case *omniast.AlterTableStmt:
		addTable(n.Table, true)
	case *omniast.TruncateStmt:
		for _, name := range n.Tables {
			addTable(name, true)
		}
	case *omniast.LoadDataStmt:
		addTable(n.Table, false)
	case *omniast.ImportIntoStmt:
		addTable(n.Table, false)
	case *omniast.RenameTableStmt:
		for _, pair := range n.Pairs {
			addTable(pair.Old, false)
			addTable(pair.New, false)
		}
	case *omniast.CreateIndexStmt:
		addTable(n.Table, false)
	case *omniast.DropIndexStmt:
		addTable(n.Table, false)
	case *omniast.CreateViewStmt:
		objectDatabase(n.Name)
	case *omniast.CreateSequenceStmt:
		objectDatabase(n.Name)
	case *omniast.AlterSequenceStmt:
		objectDatabase(n.Name)
	case *omniast.DropSequenceStmt:
		for _, seq := range n.Sequences {
			objectDatabase(seq)
		}
	case *omniast.InsertStmt:
		if n.Table != nil {
			addDMLTables(addTable, []omniast.TableExpr{n.Table})
		}
	case *omniast.UpdateStmt:
		addDMLTables(addTable, n.Tables)
	case *omniast.DeleteStmt:
		addDMLTables(addTable, slices.Concat(n.Tables, n.Using))
	case *omniast.BatchStmt:
		if n.DryRun == omniast.BatchDryRunNone {
			recordResourceChanges(database, n.DML, changedResources)
		}
	default:
	}
}

// addDMLTables records every base table named by a DML statement's table
// expressions, descending through joins. A derived table (subquery) names no
// base table, matching the previous pingcap behavior.
func addDMLTables(addTable func(*omniast.TableRef, bool), tableExprs []omniast.TableExpr) {
	var add func(expr omniast.TableExpr)
	add = func(expr omniast.TableExpr) {
		switch e := expr.(type) {
		case *omniast.TableRef:
			addTable(e, false)
		case *omniast.JoinClause:
			add(e.Left)
			add(e.Right)
		default:
		}
	}
	for _, expr := range tableExprs {
		add(expr)
	}
}

func countDML(summary *base.ChangeSummary, wrapped *AST) {
	switch n := wrapped.Node.(type) {
	case *omniast.InsertStmt:
		switch {
		case len(n.Values) > 0:
			summary.InsertCount += len(n.Values)
		case len(n.SetList) > 0:
			summary.InsertCount++
		default:
			addSampleDML(summary, wrapped.Text)
		}
	case *omniast.UpdateStmt, *omniast.DeleteStmt:
		addSampleDML(summary, wrapped.Text)
	case *omniast.BatchStmt:
		if n.DryRun == omniast.BatchDryRunNone {
			addBatchSampleDML(summary, wrapped)
		}
	default:
	}
}

// addBatchSampleDML samples the DML inside a BATCH statement, because EXPLAIN rejects the
// BATCH syntax. A BATCH without an omni AST is counted but not sampled.
func addBatchSampleDML(summary *base.ChangeSummary, ast base.AST) {
	wrapped, ok := ast.(*AST)
	if !ok {
		summary.DMLCount++
		return
	}
	batch, ok := wrapped.Node.(*omniast.BatchStmt)
	if !ok {
		summary.DMLCount++
		return
	}
	// The parser reads the body of an executable comment, so the DML's location can fall inside one
	// and the slice would leave the comment unclosed.
	if strings.Contains(wrapped.Text, "/*!") || strings.Contains(wrapped.Text, "/*T!") || strings.Contains(wrapped.Text, "/*M!") {
		summary.DMLCount++
		return
	}
	var loc omniast.Loc
	switch dml := batch.DML.(type) {
	case *omniast.DeleteStmt:
		loc = dml.Loc
	case *omniast.UpdateStmt:
		loc = dml.Loc
	case *omniast.InsertStmt:
		loc = dml.Loc
	default:
		summary.DMLCount++
		return
	}
	end := loc.End
	if end <= loc.Start || end > len(wrapped.Text) {
		end = len(wrapped.Text)
	}
	if loc.Start <= 0 || loc.Start >= end {
		summary.DMLCount++
		return
	}
	addSampleDML(summary, wrapped.Text[loc.Start:end])
}

func addSampleDML(summary *base.ChangeSummary, text string) {
	summary.DMLCount++
	summary.DMLStatements = append(summary.DMLStatements, trimStatement(text))
}

func trimStatement(statement string) string {
	return strings.TrimLeftFunc(strings.TrimRightFunc(statement, utils.IsSpaceOrSemicolon), unicode.IsSpace)
}
