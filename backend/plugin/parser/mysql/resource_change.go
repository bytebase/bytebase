package mysql

import (
	"github.com/antlr4-go/antlr/v4"
	parser "github.com/bytebase/parser/mysql"
	"github.com/pkg/errors"

	"github.com/bytebase/bytebase/backend/common"
	storepb "github.com/bytebase/bytebase/backend/generated-go/store"
	"github.com/bytebase/bytebase/backend/plugin/parser/base"
	"github.com/bytebase/bytebase/backend/store/model"
)

func init() {
	base.RegisterExtractChangedResourcesFunc(storepb.Engine_MYSQL, extractChangedResources)
	base.RegisterExtractChangedResourcesFunc(storepb.Engine_MARIADB, extractChangedResources)
	base.RegisterExtractChangedResourcesFunc(storepb.Engine_OCEANBASE, extractChangedResources)
	base.RegisterExtractChangedResourcesFunc(storepb.Engine_STARROCKS, extractChangedResources)
	base.RegisterExtractChangedResourcesFunc(storepb.Engine_DORIS, extractChangedResources)
}

func extractChangedResources(currentDatabase string, _ string, dbMetadata *model.DatabaseMetadata, asts []base.AST, statement string) (*base.ChangeSummary, error) {
	changedResources := model.NewChangedResources(dbMetadata)
	l := &resourceChangedListener{
		currentDatabase:  currentDatabase,
		statement:        statement,
		changedResources: changedResources,
	}
	for _, ast := range asts {
		antlrAST, ok := base.GetANTLRAST(ast)
		if !ok {
			return nil, errors.New("expected ANTLR AST for MySQL")
		}
		l.reset()
		antlr.ParseTreeWalkerDefault.Walk(l, antlrAST.Tree)
	}

	return &base.ChangeSummary{
		ChangedResources: changedResources,
		SampleDMLS:       l.sampleDMLs,
		DMLCount:         l.dmlCount,
		InsertCount:      l.insertCount,
	}, nil
}

type resourceChangedListener struct {
	*parser.BaseMySQLParserListener

	currentDatabase string
	statement       string

	changedResources *model.ChangedResources
	sampleDMLs       []string
	dmlCount         int
	insertCount      int

	// Internal data structure used temporarily.
	text string
}

func (l *resourceChangedListener) reset() {
	l.text = ""
}

func (l *resourceChangedListener) EnterQuery(ctx *parser.QueryContext) {
	l.text = ctx.GetParser().GetTokenStream().GetTextFromRuleContext(ctx)
}

// EnterCreateTable is called when production createTable is entered.
func (l *resourceChangedListener) EnterCreateTable(ctx *parser.CreateTableContext) {
	database, table := NormalizeMySQLTableName(ctx.TableName())
	if database == "" {
		database = l.currentDatabase
	}

	l.changedResources.AddTable(
		database,
		"",
		&storepb.ChangedResourceTable{
			Name: table,
		},
		false,
	)
}

// EnterDropTable is called when production dropTable is entered.
func (l *resourceChangedListener) EnterDropTable(ctx *parser.DropTableContext) {
	for _, table := range ctx.TableRefList().AllTableRef() {
		database, table := NormalizeMySQLTableRef(table)
		if database == "" {
			database = l.currentDatabase
		}

		l.changedResources.AddTable(
			database,
			"",
			&storepb.ChangedResourceTable{
				Name: table,
			},
			true,
		)
	}
}

// EnterAlterTable is called when production alterTable is entered.
func (l *resourceChangedListener) EnterAlterTable(ctx *parser.AlterTableContext) {
	database, table := NormalizeMySQLTableRef(ctx.TableRef())
	if database == "" {
		database = l.currentDatabase
	}

	l.changedResources.AddTable(
		database,
		"",
		&storepb.ChangedResourceTable{
			Name: table,
		},
		true,
	)
}

// EnterRenameTableStatement is called when production renameTableStatement is entered.
func (l *resourceChangedListener) EnterRenameTableStatement(ctx *parser.RenameTableStatementContext) {
	for _, pair := range ctx.AllRenamePair() {
		{
			database, table := NormalizeMySQLTableRef(pair.TableRef())
			if database == "" {
				database = l.currentDatabase
			}

			l.changedResources.AddTable(
				database,
				"",
				&storepb.ChangedResourceTable{
					Name: table,
				},
				false,
			)
		}
		{
			database, table := NormalizeMySQLTableName(pair.TableName())
			if database == "" {
				database = l.currentDatabase
			}

			l.changedResources.AddTable(
				database,
				"",
				&storepb.ChangedResourceTable{
					Name: table,
				},
				false,
			)
		}
	}
}

func (l *resourceChangedListener) EnterCreateIndex(ctx *parser.CreateIndexContext) {
	if ctx.CreateIndexTarget() == nil || ctx.CreateIndexTarget().TableRef() == nil {
		return
	}
	database, table := NormalizeMySQLTableRef(ctx.CreateIndexTarget().TableRef())
	if database == "" {
		database = l.currentDatabase
	}

	l.changedResources.AddTable(
		database,
		"",
		&storepb.ChangedResourceTable{
			Name: table,
		},
		false,
	)
}

func (l *resourceChangedListener) EnterDropIndex(ctx *parser.DropIndexContext) {
	if ctx.TableRef() == nil {
		return
	}

	database, table := NormalizeMySQLTableRef(ctx.TableRef())
	if database == "" {
		database = l.currentDatabase
	}

	l.changedResources.AddTable(
		database,
		"",
		&storepb.ChangedResourceTable{
			Name: table,
		},
		false,
	)
}

func (l *resourceChangedListener) EnterInsertStatement(ctx *parser.InsertStatementContext) {
	if !IsTopMySQLRule(&ctx.BaseParserRuleContext) {
		return
	}
	if ctx.TableRef() == nil {
		return
	}

	database, table := NormalizeMySQLTableRef(ctx.TableRef())
	if database == "" {
		database = l.currentDatabase
	}

	l.changedResources.AddTable(
		database,
		"",
		&storepb.ChangedResourceTable{
			Name: table,
		},
		false,
	)

	if ctx.InsertFromConstructor() != nil && ctx.InsertFromConstructor().InsertValues() != nil && ctx.InsertFromConstructor().InsertValues().ValueList() != nil {
		l.insertCount += len(ctx.InsertFromConstructor().InsertValues().ValueList().AllValues())
		return
	}

	// Track DMLs.
	l.dmlCount++
	if len(l.sampleDMLs) < common.MaximumLintExplainSize {
		l.sampleDMLs = append(l.sampleDMLs, l.text)
	}
}

func (l *resourceChangedListener) EnterUpdateStatement(ctx *parser.UpdateStatementContext) {
	if !IsTopMySQLRule(&ctx.BaseParserRuleContext) {
		return
	}
	for _, tableRefCtx := range ctx.TableReferenceList().AllTableReference() {
		resources := l.extractTableReference(tableRefCtx)
		for _, resource := range resources {
			l.changedResources.AddTable(
				resource.Database,
				"",
				&storepb.ChangedResourceTable{
					Name: resource.Table,
				},
				false,
			)
		}
	}

	// Track DMLs.
	l.dmlCount++
	if len(l.sampleDMLs) < common.MaximumLintExplainSize {
		l.sampleDMLs = append(l.sampleDMLs, l.text)
	}
}

func (l *resourceChangedListener) EnterDeleteStatement(ctx *parser.DeleteStatementContext) {
	if !IsTopMySQLRule(&ctx.BaseParserRuleContext) {
		return
	}
	var allResources []base.SchemaResource
	if ctx.TableRef() != nil {
		resources := l.extractTableRef(ctx.TableRef())
		allResources = append(allResources, resources...)
	}
	if ctx.TableReferenceList() != nil {
		resources := l.extractTableReferenceList(ctx.TableReferenceList())
		allResources = append(allResources, resources...)
	}

	for _, resource := range allResources {
		l.changedResources.AddTable(
			resource.Database,
			"",
			&storepb.ChangedResourceTable{
				Name: resource.Table,
			},
			false,
		)
	}

	// Track DMLs.
	l.dmlCount++
	if len(l.sampleDMLs) < common.MaximumLintExplainSize {
		l.sampleDMLs = append(l.sampleDMLs, l.text)
	}
}

func (l *resourceChangedListener) extractTableReference(ctx parser.ITableReferenceContext) []base.SchemaResource {
	if ctx.TableFactor() == nil {
		return nil
	}
	res := l.extractTableFactor(ctx.TableFactor())
	for _, joinedTableCtx := range ctx.AllJoinedTable() {
		tables := l.extractJoinedTable(joinedTableCtx)
		res = append(res, tables...)
	}

	return res
}

func (l *resourceChangedListener) extractTableRef(ctx parser.ITableRefContext) []base.SchemaResource {
	if ctx == nil {
		return nil
	}
	resource := base.SchemaResource{
		Database: l.currentDatabase,
	}
	db, table := NormalizeMySQLTableRef(ctx)
	if db != "" {
		resource.Database = db
	}
	resource.Table = table

	return []base.SchemaResource{resource}
}

func (l *resourceChangedListener) extractTableReferenceList(ctx parser.ITableReferenceListContext) []base.SchemaResource {
	var res []base.SchemaResource
	for _, tableRefCtx := range ctx.AllTableReference() {
		tables := l.extractTableReference(tableRefCtx)
		res = append(res, tables...)
	}
	return res
}

func (l *resourceChangedListener) extractTableReferenceListParens(ctx parser.ITableReferenceListParensContext) []base.SchemaResource {
	if ctx.TableReferenceList() != nil {
		return l.extractTableReferenceList(ctx.TableReferenceList())
	}
	if ctx.TableReferenceListParens() != nil {
		return l.extractTableReferenceListParens(ctx.TableReferenceListParens())
	}
	return nil
}

func (l *resourceChangedListener) extractTableFactor(ctx parser.ITableFactorContext) []base.SchemaResource {
	switch {
	case ctx.SingleTable() != nil:
		return l.extractSingleTable(ctx.SingleTable())
	case ctx.SingleTableParens() != nil:
		return l.extractSingleTableParens(ctx.SingleTableParens())
	case ctx.DerivedTable() != nil:
		return nil
	case ctx.TableReferenceListParens() != nil:
		return l.extractTableReferenceListParens(ctx.TableReferenceListParens())
	case ctx.TableFunction() != nil:
		return nil
	default:
		return nil
	}
}

func (l *resourceChangedListener) extractSingleTable(ctx parser.ISingleTableContext) []base.SchemaResource {
	return l.extractTableRef(ctx.TableRef())
}

func (l *resourceChangedListener) extractSingleTableParens(ctx parser.ISingleTableParensContext) []base.SchemaResource {
	if ctx.SingleTable() != nil {
		return l.extractSingleTable(ctx.SingleTable())
	}
	if ctx.SingleTableParens() != nil {
		return l.extractSingleTableParens(ctx.SingleTableParens())
	}
	return nil
}

func (l *resourceChangedListener) extractJoinedTable(ctx parser.IJoinedTableContext) []base.SchemaResource {
	if ctx.TableFactor() != nil {
		return l.extractTableFactor(ctx.TableFactor())
	}
	if ctx.TableReference() != nil {
		return l.extractTableReference(ctx.TableReference())
	}
	return nil
}
