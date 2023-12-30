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
