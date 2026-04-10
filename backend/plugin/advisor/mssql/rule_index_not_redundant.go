package mssql

import (
	"context"
	"fmt"
	"strings"

	"github.com/bytebase/omni/mssql/ast"

	storepb "github.com/bytebase/bytebase/backend/generated-go/store"
	"github.com/bytebase/bytebase/backend/plugin/advisor"
	"github.com/bytebase/bytebase/backend/plugin/advisor/code"
)

func init() {
	advisor.Register(storepb.Engine_MSSQL, storepb.SQLReviewRule_INDEX_NOT_REDUNDANT, &IndexNotRedundantAdvisor{})
}

var dftMSSQLSchemaName = "dbo"

// We consider all the indexes are based on B-Tree. Only in this situation ordered column list makes sense.
// TODO(zp): However, some of the indexes in MSSQL are not implemented with B-Tree:
// ref: https://learn.microsoft.com/en-us/sql/relational-databases/system-catalog-views/sys-indexes-transact-sql?view=sql-server-ver16/.

type IndexNotRedundantAdvisor struct{}

// TODO(zp): we currently don't have runtime checks for indexes created in the statements.
func (IndexNotRedundantAdvisor) Check(_ context.Context, checkCtx advisor.Context) ([]*storepb.Advice, error) {
	level, err := advisor.NewStatusBySQLReviewRuleLevel(checkCtx.Rule.Level)
	if err != nil {
		return nil, err
	}

	rule := &indexNotRedundantRule{
		OmniBaseRule: OmniBaseRule{Level: level, Title: checkCtx.Rule.Type.String()},
		curDB:        checkCtx.CurrentDatabase,
		indexMap:     getIndexMapFromMetadata(checkCtx.DBSchema),
	}
	return RunOmniRules(checkCtx.ParsedStatements, []OmniRule{rule}), nil
}

type indexNotRedundantRule struct {
	OmniBaseRule
	curDB    string
	indexMap *IndexMap
}

func (*indexNotRedundantRule) Name() string {
	return "IndexNotRedundantRule"
}

func (r *indexNotRedundantRule) OnStatement(node ast.Node) {
	ci, ok := node.(*ast.CreateIndexStmt)
	if !ok {
		return
	}

	idxSchemaName := dftMSSQLSchemaName
	idxTblName := ""

	if ci.Table != nil {
		// TODO(zp): we only check indexes in the current database due to the lack of necessary metadata.
		if ci.Table.Database != "" && !strings.EqualFold(ci.Table.Database, r.curDB) {
			return
		}
		if ci.Table.Schema != "" {
			idxSchemaName = ci.Table.Schema
		}
		idxTblName = ci.Table.Object
	}

	statIdxName := ci.Name

	findIdxKey := FindIndexesKey{
		schemaName: idxSchemaName,
		tblName:    idxTblName,
	}
	metaIdxList, ok := (*r.indexMap)[findIdxKey]
	if !ok {
		return
	}

	// Get the first column from the statement's index column list.
	if ci.Columns == nil || ci.Columns.Len() == 0 {
		return
	}
	firstCol := ""
	if idxCol, ok := ci.Columns.Items[0].(*ast.IndexColumn); ok && idxCol != nil {
		firstCol = idxCol.Name
	}
	if firstCol == "" {
		return
	}

	for _, metaIndex := range metaIdxList {
		if len(metaIndex.Expressions) > 0 && metaIndex.Expressions[0] == firstCol {
			r.AddAdvice(&storepb.Advice{
				Status: r.Level,
				Title:  r.Title,
				Code:   code.RedundantIndex.Int32(),
				Content: fmt.Sprintf("Redundant indexes with the same prefix ('%s' and '%s') in '%s.%s' is not allowed",
					metaIndex.Name, statIdxName, findIdxKey.schemaName, findIdxKey.tblName),
				StartPosition: &storepb.Position{Line: r.LocToLine(ci.Loc)},
			})
			return
		}
	}
}

type FindIndexesKey struct {
	schemaName string
	tblName    string
}

// The value in the map represents the column list of a certain index.
type IndexMap = map[FindIndexesKey][]*storepb.IndexMetadata

func getIndexMapFromMetadata(dbMetadata *storepb.DatabaseSchemaMetadata) *IndexMap {
	indexMap := IndexMap{}
	if dbMetadata == nil {
		return &indexMap
	}
	for _, schema := range dbMetadata.Schemas {
		if schema.Name == "" {
			continue
		}
		for _, tbl := range schema.Tables {
			if tbl.Name == "" {
				continue
			}
			indexMap[FindIndexesKey{
				schemaName: schema.Name,
				tblName:    tbl.Name,
			}] = tbl.Indexes
		}
	}
	return &indexMap
}
