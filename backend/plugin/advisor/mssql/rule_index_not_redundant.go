package mssql

import (
	"context"
	"fmt"

	"github.com/bytebase/bytebase/backend/plugin/advisor/code"

	"github.com/antlr4-go/antlr/v4"
	parser "github.com/bytebase/parser/tsql"
	"github.com/pkg/errors"

	"github.com/bytebase/bytebase/backend/common"
	"github.com/bytebase/bytebase/backend/plugin/advisor"
	"github.com/bytebase/bytebase/backend/plugin/parser/tsql"

	storepb "github.com/bytebase/bytebase/backend/generated-go/store"
)

func init() {
	advisor.Register(storepb.Engine_MSSQL, advisor.SchemaRuleIndexNotRedundant, &IndexNotRedundantAdvisor{})
}

var dftMSSQLSchemaName = "dbo"

// We consider all the indexes are based on B-Tree. Only in this situation ordered column list makes sense.
// TODO(zp): However, some of the indexes in MSSQL are not implemented with B-Tree:
// ref: https://learn.microsoft.com/en-us/sql/relational-databases/system-catalog-views/sys-indexes-transact-sql?view=sql-server-ver16/.

type IndexNotRedundantAdvisor struct{}

// TODO(zp): we currently don't have runtime checks for indexes created in the statements.
func (IndexNotRedundantAdvisor) Check(_ context.Context, checkCtx advisor.Context) ([]*storepb.Advice, error) {
	tree, ok := checkCtx.AST.(antlr.Tree)
	if !ok {
		return nil, errors.Errorf("failed to convert to AST tree")
	}

	level, err := advisor.NewStatusBySQLReviewRuleLevel(checkCtx.Rule.Level)
	if err != nil {
		return nil, err
	}

	// Create the rule
	rule := NewIndexNotRedundantRule(level, string(checkCtx.Rule.Type), checkCtx.CurrentDatabase, checkCtx.DBSchema)

	// Create the generic checker with the rule
	checker := NewGenericChecker([]Rule{rule})

	antlr.ParseTreeWalkerDefault.Walk(checker, tree)

	return checker.GetAdviceList(), nil
}

// IndexNotRedundantRule checks for redundant indexes.
type IndexNotRedundantRule struct {
	BaseRule
	curDB    string
	indexMap *IndexMap
}

// NewIndexNotRedundantRule creates a new IndexNotRedundantRule.
func NewIndexNotRedundantRule(level storepb.Advice_Status, title string, currentDB string, dbSchema *storepb.DatabaseSchemaMetadata) *IndexNotRedundantRule {
	return &IndexNotRedundantRule{
		BaseRule: BaseRule{
			level: level,
			title: title,
		},
		curDB:    currentDB,
		indexMap: getIndexMapFromMetadata(dbSchema),
	}
}

// Name returns the rule name.
func (*IndexNotRedundantRule) Name() string {
	return "IndexNotRedundantRule"
}

// OnEnter is called when entering a parse tree node.
func (r *IndexNotRedundantRule) OnEnter(ctx antlr.ParserRuleContext, nodeType string) error {
	if nodeType == NodeTypeCreateIndex {
		r.enterCreateIndex(ctx.(*parser.Create_indexContext))
	}
	return nil
}

// OnExit is called when exiting a parse tree node.
func (*IndexNotRedundantRule) OnExit(_ antlr.ParserRuleContext, _ string) error {
	// This rule doesn't need exit processing
	return nil
}

func (r *IndexNotRedundantRule) enterCreateIndex(ctx *parser.Create_indexContext) {
	idxSchemaName := dftMSSQLSchemaName
	idxTblName := ""

	// Get full table name.
	if fullTblName := ctx.Table_name(); fullTblName != nil {
		// TODO(zp): we only check indexes in the current database due to the lack of necessary metadata.
		// Case sensitive!
		if database := fullTblName.GetDatabase(); database != nil {
			if oriName, _ := tsql.NormalizeTSQLIdentifier(database); oriName != r.curDB {
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
	metaIdxList, ok := (*r.indexMap)[findIdxKey]
	if !ok {
		return
	}

	// Get ordered column list from statement.
	statIdxColList := ctx.Column_name_list_with_order()
	if metaIdxName := containRedundantPrefix(metaIdxList, &statIdxColList); metaIdxName != "" {
		r.AddAdvice(&storepb.Advice{
			Status: r.level,
			Title:  r.title,
			Code:   code.RedundantIndex.Int32(),
			Content: fmt.Sprintf("Redundant indexes with the same prefix ('%s' and '%s') in '%s.%s' is not allowed",
				metaIdxName, statIdxName, findIdxKey.schemaName, findIdxKey.tblName),
			StartPosition: common.ConvertANTLRLineToPosition(ctx.GetStart().GetLine()),
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
		if statColumnList != nil && len((*statColumnList).AllColumn_name_with_order()) != 0 && len(metaIdxList) != 0 {
			statIdxCol, _ := tsql.NormalizeTSQLIdentifier((*statColumnList).AllColumn_name_with_order()[0].Id_())
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
