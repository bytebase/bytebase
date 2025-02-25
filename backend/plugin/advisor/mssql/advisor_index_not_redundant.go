package mssql

import (
	"fmt"

	"github.com/antlr4-go/antlr/v4"
	parser "github.com/bytebase/tsql-parser"
	"github.com/pkg/errors"

	"github.com/bytebase/bytebase/backend/plugin/advisor"
	"github.com/bytebase/bytebase/backend/plugin/parser/tsql"

	storepb "github.com/bytebase/bytebase/proto/generated-go/store"
)

func init() {
	advisor.Register(storepb.Engine_MSSQL, advisor.MSSQLIndexNotRedundant, IndexNotRedundantAdvisor{})
}

var dftMSSQLSchemaName = "dbo"

// We consider all the indexes are based on B-Tree. Only in this situation ordered column list makes sense.
// TODO(zp): However, some of the indexes in MSSQL are not implemented with B-Tree:
// ref: https://learn.microsoft.com/en-us/sql/relational-databases/system-catalog-views/sys-indexes-transact-sql?view=sql-server-ver16/.

type IndexNotRedundantAdvisor struct{}

type IndexNotRedundantChecker struct {
	*parser.BaseTSqlParserListener
	level      storepb.Advice_Status
	title      string
	curDB      string
	indexMap   *IndexMap
	adviceList []*storepb.Advice
}

// TODO(zp): we currently don't have runtime checks for indexes created in the statements.
func (IndexNotRedundantAdvisor) Check(ctx advisor.Context) ([]*storepb.Advice, error) {
	tree, ok := ctx.AST.(antlr.Tree)
	if !ok {
		return nil, errors.Errorf("failed to convert to AST tree")
	}

	level, err := advisor.NewStatusBySQLReviewRuleLevel(ctx.Rule.Level)
	if err != nil {
		return nil, err
	}

	checker := &IndexNotRedundantChecker{
		level:    level,
		title:    ctx.Rule.Type,
		curDB:    ctx.CurrentDatabase,
		indexMap: getIndexMapFromMetadata(ctx.DBSchema),
	}

	antlr.ParseTreeWalkerDefault.Walk(checker, tree)

	return checker.adviceList, nil
}

func (checker *IndexNotRedundantChecker) EnterCreate_index(ctx *parser.Create_indexContext) {
	idxSchemaName := dftMSSQLSchemaName
	idxTblName := ""

	// Get full table name.
	if fullTblName := ctx.Table_name(); fullTblName != nil {
		// TODO(zp): we only check indexes in the current database due to the lack of necessary metadata.
		// Case sensitive!
		if database := fullTblName.GetDatabase(); database != nil {
			if oriName, _ := tsql.NormalizeTSQLIdentifier(database); oriName != checker.curDB {
				return
			}
		}
		if schema := fullTblName.GetSchema(); schema != nil {
			idxSchemaName, _ = tsql.NormalizeTSQLIdentifier(schema)
		}
		idxTblName, _ = tsql.NormalizeTSQLIdentifier(fullTblName.GetTable())
	}

	// Get index name from statement.
	statIdxName, _ := tsql.NormalizeTSQLIdentifier(ctx.AllId_()[0])

	// Get ordered index list from metadata.
	findIdxKey := FindIndexesKey{
		schemaName: idxSchemaName,
		tblName:    idxTblName,
	}
	metaIdxList, ok := (*checker.indexMap)[findIdxKey]
	if !ok {
		return
	}

	// Get ordered column list from statement.
	statIdxColList := ctx.Column_name_list_with_order()
	if metaIdxName := containRedundantPrefix(metaIdxList, &statIdxColList); metaIdxName != "" {
		checker.adviceList = append(checker.adviceList, &storepb.Advice{
			Status: checker.level,
			Title:  checker.title,
			StartPosition: &storepb.Position{
				Line: int32(ctx.GetStart().GetLine()),
			},
			Code: advisor.RedundantIndex.Int32(),
			Content: fmt.Sprintf("Redundant indexes with the same prefix ('%s' and '%s') in '%s.%s' is not allowed",
				metaIdxName, statIdxName, findIdxKey.schemaName, findIdxKey.tblName),
		})
	}
}

type FindIndexesKey struct {
	schemaName string
	tblName    string
}

// The value in the map represents the column list of a certain index.
type IndexMap = map[FindIndexesKey][]*storepb.IndexMetadata

// Return the name of the index if redundant prefixes are found.
func containRedundantPrefix(metaIdxList []*storepb.IndexMetadata, statColumnList *parser.IColumn_name_list_with_orderContext) string {
	for _, metaIndex := range metaIdxList {
		if statColumnList != nil && len((*statColumnList).AllId_()) != 0 && len(metaIdxList) != 0 {
			statIdxCol, _ := tsql.NormalizeTSQLIdentifier((*statColumnList).AllId_()[0])
			if metaIndex.Expressions[0] == statIdxCol {
				return metaIndex.Name
			}
		}
	}
	return ""
}

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
