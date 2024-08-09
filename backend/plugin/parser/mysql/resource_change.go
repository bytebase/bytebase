package mysql

import (
	"sort"

	"github.com/antlr4-go/antlr/v4"
	parser "github.com/bytebase/mysql-parser"
	"github.com/pkg/errors"

	"github.com/bytebase/bytebase/backend/common"
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

func extractChangedResources(currentDatabase string, _ string, asts any, statement string) (*base.ChangeSummary, error) {
	nodes, ok := asts.([]*ParseResult)
	if !ok {
		return nil, errors.Errorf("invalid ast type %T", asts)
	}

	l := &resourceChangedListener{
		currentDatabase:   currentDatabase,
		statement:         statement,
		resourceChangeMap: make(map[string]*base.ResourceChange),
	}
	for _, node := range nodes {
		l.reset()
		antlr.ParseTreeWalkerDefault.Walk(l, node.Tree)
	}

	var resourceChanges []*base.ResourceChange
	for _, change := range l.resourceChangeMap {
		resourceChanges = append(resourceChanges, change)
	}
	sort.Slice(resourceChanges, func(i, j int) bool {
		return resourceChanges[i].String() < resourceChanges[j].String()
	})
	return &base.ChangeSummary{
		ResourceChanges: resourceChanges,
		SampleDMLS:      l.sampleDMLs,
		DMLCount:        l.dmlCount,
	}, nil
}

type resourceChangedListener struct {
	*parser.BaseMySQLParserListener

	currentDatabase string
	statement       string

	resourceChangeMap map[string]*base.ResourceChange
	sampleDMLs        []string
	dmlCount          int

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
	resource := base.SchemaResource{
		Database: l.currentDatabase,
	}
	db, table := NormalizeMySQLTableName(ctx.TableName())
	if db != "" {
		resource.Database = db
	}
	resource.Table = table

	if _, ok := l.resourceChangeMap[resource.String()]; !ok {
		l.resourceChangeMap[resource.String()] = &base.ResourceChange{
			Resource: resource,
		}
	}
	l.resourceChangeMap[resource.String()].Ranges = append(l.resourceChangeMap[resource.String()].Ranges, base.NewRange(l.statement, l.text))
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

		if v, ok := l.resourceChangeMap[resource.String()]; ok {
			v.AffectTable = true
		} else {
			l.resourceChangeMap[resource.String()] = &base.ResourceChange{
				Resource:    resource,
				AffectTable: true,
			}
		}
		l.resourceChangeMap[resource.String()].Ranges = append(l.resourceChangeMap[resource.String()].Ranges, base.NewRange(l.statement, l.text))
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

	if v, ok := l.resourceChangeMap[resource.String()]; ok {
		v.AffectTable = true
	} else {
		l.resourceChangeMap[resource.String()] = &base.ResourceChange{
			Resource:    resource,
			AffectTable: true,
		}
	}
	l.resourceChangeMap[resource.String()].Ranges = append(l.resourceChangeMap[resource.String()].Ranges, base.NewRange(l.statement, l.text))
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
			if _, ok := l.resourceChangeMap[resource.String()]; !ok {
				l.resourceChangeMap[resource.String()] = &base.ResourceChange{
					Resource: resource,
				}
			}
			l.resourceChangeMap[resource.String()].Ranges = append(l.resourceChangeMap[resource.String()].Ranges, base.NewRange(l.statement, l.text))
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
			if _, ok := l.resourceChangeMap[resource.String()]; !ok {
				l.resourceChangeMap[resource.String()] = &base.ResourceChange{
					Resource: resource,
				}
			}
			l.resourceChangeMap[resource.String()].Ranges = append(l.resourceChangeMap[resource.String()].Ranges, base.NewRange(l.statement, l.text))
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

	if _, ok := l.resourceChangeMap[resource.String()]; !ok {
		l.resourceChangeMap[resource.String()] = &base.ResourceChange{
			Resource: resource,
		}
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
			if _, ok := l.resourceChangeMap[resource.String()]; !ok {
				l.resourceChangeMap[resource.String()] = &base.ResourceChange{
					Resource: resource,
				}
			}
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
		if _, ok := l.resourceChangeMap[resource.String()]; !ok {
			l.resourceChangeMap[resource.String()] = &base.ResourceChange{
				Resource: resource,
			}
		}
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
