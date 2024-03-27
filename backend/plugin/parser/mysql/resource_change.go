package mysql

import (
	"sort"

	"github.com/antlr4-go/antlr/v4"
	parser "github.com/bytebase/mysql-parser"

	"github.com/bytebase/bytebase/backend/plugin/parser/base"
	storepb "github.com/bytebase/bytebase/proto/generated-go/store"
)

func init() {
	base.RegisterExtractChangedResourcesFunc(storepb.Engine_MYSQL, extractChangedResources)
	base.RegisterExtractChangedResourcesFunc(storepb.Engine_MARIADB, extractChangedResources)
	base.RegisterExtractChangedResourcesFunc(storepb.Engine_OCEANBASE, extractChangedResources)
	base.RegisterExtractChangedResourcesFunc(storepb.Engine_STARROCKS, extractChangedResources)
	base.RegisterExtractChangedResourcesFunc(storepb.Engine_DORIS, extractChangedResources)
}

func extractChangedResources(currentDatabase string, _, statement string) ([]base.SchemaResource, error) {
	treeList, err := ParseMySQL(statement)
	if err != nil {
		return nil, err
	}

	l := &resourceChangedListener{
		currentDatabase: currentDatabase,
		resourceMap:     make(map[string]base.SchemaResource),
	}

	var result []base.SchemaResource
	for _, tree := range treeList {
		if tree.Tree == nil {
			continue
		}
		antlr.ParseTreeWalkerDefault.Walk(l, tree.Tree)
	}

	for _, resource := range l.resourceMap {
		result = append(result, resource)
	}

	sort.Slice(result, func(i, j int) bool {
		return result[i].String() < result[j].String()
	})
	return result, nil
}

type resourceChangedListener struct {
	*parser.BaseMySQLParserListener

	currentDatabase string
	resourceMap     map[string]base.SchemaResource
}

// EnterCreateTable is called when production createTable is entered.
func (l *resourceChangedListener) EnterCreateTable(ctx *parser.CreateTableContext) {
	resource := base.SchemaResource{
		Database: l.currentDatabase,
	}
	db, table := NormalizeMySQLTableName(ctx.TableName())
	if db != "" {
		resource.Database = db
	}
	resource.Table = table
	l.resourceMap[resource.String()] = resource
}

// EnterDropTable is called when production dropTable is entered.
func (l *resourceChangedListener) EnterDropTable(ctx *parser.DropTableContext) {
	for _, table := range ctx.TableRefList().AllTableRef() {
		resource := base.SchemaResource{
			Database: l.currentDatabase,
		}
		db, table := NormalizeMySQLTableRef(table)
		if db != "" {
			resource.Database = db
		}
		resource.Table = table
		l.resourceMap[resource.String()] = resource
	}
}

// EnterAlterTable is called when production alterTable is entered.
func (l *resourceChangedListener) EnterAlterTable(ctx *parser.AlterTableContext) {
	resource := base.SchemaResource{
		Database: l.currentDatabase,
	}
	db, table := NormalizeMySQLTableRef(ctx.TableRef())
	if db != "" {
		resource.Database = db
	}
	resource.Table = table
	l.resourceMap[resource.String()] = resource
}

// EnterRenameTableStatement is called when production renameTableStatement is entered.
func (l *resourceChangedListener) EnterRenameTableStatement(ctx *parser.RenameTableStatementContext) {
	for _, pair := range ctx.AllRenamePair() {
		{
			resource := base.SchemaResource{
				Database: l.currentDatabase,
			}
			db, table := NormalizeMySQLTableRef(pair.TableRef())
			if db != "" {
				resource.Database = db
			}
			resource.Table = table
			l.resourceMap[resource.String()] = resource
		}
		{
			resource := base.SchemaResource{
				Database: l.currentDatabase,
			}
			db, table := NormalizeMySQLTableName(pair.TableName())
			if db != "" {
				resource.Database = db
			}
			resource.Table = table
			l.resourceMap[resource.String()] = resource
		}
	}
}

func (l *resourceChangedListener) EnterInsertStatement(ctx *parser.InsertStatementContext) {
	if !IsTopMySQLRule(&ctx.BaseParserRuleContext) {
		return
	}
	if ctx.TableRef() == nil {
		return
	}
	resource := base.SchemaResource{
		Database: l.currentDatabase,
	}
	db, table := NormalizeMySQLTableRef(ctx.TableRef())
	if db != "" {
		resource.Database = db
	}
	resource.Table = table
	l.resourceMap[resource.String()] = resource
}

func (l *resourceChangedListener) EnterUpdateStatement(ctx *parser.UpdateStatementContext) {
	if !IsTopMySQLRule(&ctx.BaseParserRuleContext) {
		return
	}
	for _, tableRefCtx := range ctx.TableReferenceList().AllTableReference() {
		resources := l.extractTableReference(tableRefCtx)
		for _, resource := range resources {
			l.resourceMap[resource.String()] = resource
		}
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
		l.resourceMap[resource.String()] = resource
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
