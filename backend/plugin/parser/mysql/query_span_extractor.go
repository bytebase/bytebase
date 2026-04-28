package mysql

import (
	"context"
	"strings"

	"github.com/pkg/errors"

	storepb "github.com/bytebase/bytebase/backend/generated-go/store"
	"github.com/bytebase/bytebase/backend/plugin/parser/base"
	"github.com/bytebase/bytebase/backend/store/model"
)

// querySpanExtractor contains shared query-span state and metadata helpers.
// MySQL query-span extraction itself is omni-backed in query_span_extractor_omni.go.
type querySpanExtractor struct {
	ctx             context.Context
	defaultDatabase string

	gCtx base.GetQuerySpanContext

	ignoreCaseSensitive bool

	// ctes records common table expressions visible to the current select scope.
	ctes []*base.PseudoTable

	// outerTableSources resolves correlated subquery column references.
	outerTableSources []base.TableSource

	// tableSourceFrom is the set of table sources visible from the current FROM clause.
	tableSourceFrom []base.TableSource

	// priorTableInFrom resolves JSON_TABLE document expressions against preceding FROM items.
	priorTableInFrom []base.TableSource
}

func newQuerySpanExtractor(defaultDatabase string, gCtx base.GetQuerySpanContext, ignoreCaseSensitive bool) *querySpanExtractor {
	return &querySpanExtractor{
		defaultDatabase:     defaultDatabase,
		gCtx:                gCtx,
		ignoreCaseSensitive: ignoreCaseSensitive,
	}
}

func (q *querySpanExtractor) getQuerySpan(ctx context.Context, stmt string) (*base.QuerySpan, error) {
	return (&omniQuerySpanExtractor{querySpanExtractor: q}).getOmniQuerySpan(ctx, stmt)
}

type joinType int

const (
	Join joinType = iota
	InnerJoin
	CrossJoin
	StraightJoin
	LeftOuterJoin
	RightOuterJoin
	NaturalInnerJoin
	NaturalLeftOuterJoin
	NaturalRightOuterJoin
)

func joinTableSources(l, r base.TableSource, tp joinType, using []string) base.TableSource {
	switch tp {
	case Join, InnerJoin, CrossJoin, StraightJoin, LeftOuterJoin, RightOuterJoin:
		var columns []base.QuerySpanResult
		rightFieldsMap := make(map[string]bool)
		for _, field := range r.GetQuerySpanResult() {
			rightFieldsMap[strings.ToLower(field.Name)] = true
		}
		lowercaseUsingFields := make(map[string]bool)
		for _, field := range using {
			lowercaseUsingFields[strings.ToLower(field)] = true
		}
		for _, field := range l.GetQuerySpanResult() {
			columns = append(columns, field)
			if _, ok := lowercaseUsingFields[strings.ToLower(field.Name)]; ok {
				delete(rightFieldsMap, strings.ToLower(field.Name))
			}
		}
		for _, field := range r.GetQuerySpanResult() {
			if _, ok := rightFieldsMap[strings.ToLower(field.Name)]; ok {
				columns = append(columns, field)
			}
		}
		return &base.PseudoTable{
			Columns: columns,
		}
	case NaturalInnerJoin, NaturalLeftOuterJoin, NaturalRightOuterJoin:
		rightFieldsMap := make(map[string]bool)
		for _, field := range r.GetQuerySpanResult() {
			rightFieldsMap[strings.ToLower(field.Name)] = true
		}
		var columns []base.QuerySpanResult
		for _, field := range l.GetQuerySpanResult() {
			columns = append(columns, field)
			delete(rightFieldsMap, strings.ToLower(field.Name))
		}
		for _, field := range r.GetQuerySpanResult() {
			if _, ok := rightFieldsMap[strings.ToLower(field.Name)]; ok {
				columns = append(columns, field)
			}
		}
		return &base.PseudoTable{
			Columns: columns,
		}
	default:
		return nil
	}
}

func (q *querySpanExtractor) getAllTableColumnSources(databaseName, tableName string) ([]base.QuerySpanResult, bool) {
	findInTableSource := func(tableSource base.TableSource) ([]base.QuerySpanResult, bool) {
		if q.ignoreCaseSensitive {
			if databaseName != "" && !strings.EqualFold(databaseName, tableSource.GetDatabaseName()) {
				return nil, false
			}
			if tableName != "" && !strings.EqualFold(tableName, tableSource.GetTableName()) {
				return nil, false
			}
		} else {
			if databaseName != "" && databaseName != tableSource.GetDatabaseName() {
				return nil, false
			}
			if tableName != "" && tableName != tableSource.GetTableName() {
				return nil, false
			}
		}
		return tableSource.GetQuerySpanResult(), true
	}

	for i := len(q.outerTableSources) - 1; i >= 0; i-- {
		if querySpanResult, ok := findInTableSource(q.outerTableSources[i]); ok {
			return querySpanResult, true
		}
	}
	for i := len(q.tableSourceFrom) - 1; i >= 0; i-- {
		if querySpanResult, ok := findInTableSource(q.tableSourceFrom[i]); ok {
			return querySpanResult, true
		}
	}
	return nil, false
}

func (q *querySpanExtractor) getFieldColumnSource(databaseName, tableName, fieldName string) (base.SourceColumnSet, error) {
	databaseName = q.filterClusterName(databaseName)
	findInTableSource := func(tableSource base.TableSource) (base.SourceColumnSet, bool) {
		if q.ignoreCaseSensitive {
			if databaseName != "" && !strings.EqualFold(databaseName, tableSource.GetDatabaseName()) {
				return nil, false
			}
			if tableName != "" && !strings.EqualFold(tableName, tableSource.GetTableName()) {
				return nil, false
			}
		} else {
			if databaseName != "" && databaseName != tableSource.GetDatabaseName() {
				return nil, false
			}
			if tableName != "" && tableName != tableSource.GetTableName() {
				return nil, false
			}
		}
		for _, column := range tableSource.GetQuerySpanResult() {
			if strings.EqualFold(column.Name, fieldName) {
				return column.SourceColumns, true
			}
		}
		return nil, false
	}

	for i := len(q.outerTableSources) - 1; i >= 0; i-- {
		if sourceColumnSet, ok := findInTableSource(q.outerTableSources[i]); ok {
			return sourceColumnSet, nil
		}
	}
	for i := len(q.tableSourceFrom) - 1; i >= 0; i-- {
		if sourceColumnSet, ok := findInTableSource(q.tableSourceFrom[i]); ok {
			return sourceColumnSet, nil
		}
	}
	for i := len(q.priorTableInFrom) - 1; i >= 0; i-- {
		if sourceColumnSet, ok := findInTableSource(q.priorTableInFrom[i]); ok {
			return sourceColumnSet, nil
		}
	}

	return nil, &base.ResourceNotFoundError{
		Database: &databaseName,
		Table:    &tableName,
		Column:   &fieldName,
	}
}

func (q *querySpanExtractor) filterClusterName(databaseName string) string {
	if q.gCtx.Engine == storepb.Engine_STARROCKS {
		list := strings.Split(databaseName, ":")
		if len(list) > 1 {
			databaseName = list[len(list)-1]
		}
	}
	return databaseName
}

func (q *querySpanExtractor) findTableSchema(databaseName, tableName string) (base.TableSource, error) {
	if databaseName == "" {
		for i := len(q.ctes) - 1; i >= 0; i-- {
			table := q.ctes[i]
			if table.Name == tableName {
				return table, nil
			}
		}
	}

	if databaseName == "" {
		databaseName = q.defaultDatabase
	}
	databaseName = q.filterClusterName(databaseName)

	var dbMetadata *model.DatabaseMetadata
	allDatabaseNames, err := q.gCtx.ListDatabaseNamesFunc(q.ctx, q.gCtx.InstanceID)
	if err != nil {
		return nil, errors.Wrap(err, "failed to list databases")
	}
	if q.ignoreCaseSensitive {
		for _, db := range allDatabaseNames {
			if strings.EqualFold(db, databaseName) {
				_, dbMetadata, err = q.gCtx.GetDatabaseMetadataFunc(q.ctx, q.gCtx.InstanceID, db)
				if err != nil {
					return nil, errors.Wrapf(err, "failed to get database metadata for database %q", db)
				}
				break
			}
		}
	} else {
		for _, db := range allDatabaseNames {
			if db == databaseName {
				_, dbMetadata, err = q.gCtx.GetDatabaseMetadataFunc(q.ctx, q.gCtx.InstanceID, db)
				if err != nil {
					return nil, errors.Wrapf(err, "failed to get database metadata for database %q", db)
				}
				break
			}
		}
	}
	if dbMetadata == nil {
		return nil, &base.ResourceNotFoundError{
			Database: &databaseName,
		}
	}

	emptySchema := ""
	schema := dbMetadata.GetSchemaMetadata(emptySchema)
	if schema == nil {
		return nil, &base.ResourceNotFoundError{
			Database: &databaseName,
			Schema:   &emptySchema,
		}
	}

	var tableSchema *model.TableMetadata
	if q.ignoreCaseSensitive {
		for _, table := range schema.ListTableNames() {
			if strings.EqualFold(table, tableName) {
				tableSchema = schema.GetTable(table)
				break
			}
		}
	} else {
		tableSchema = schema.GetTable(tableName)
	}
	if tableSchema != nil {
		columnNames := make([]string, 0, len(tableSchema.GetProto().GetColumns()))
		for _, column := range tableSchema.GetProto().GetColumns() {
			columnNames = append(columnNames, column.Name)
		}
		return &base.PhysicalTable{
			Name:     tableSchema.GetProto().Name,
			Schema:   emptySchema,
			Database: dbMetadata.GetProto().GetName(),
			Server:   "",
			Columns:  columnNames,
		}, nil
	}

	var viewSchema *storepb.ViewMetadata
	if q.ignoreCaseSensitive {
		for _, view := range schema.ListViewNames() {
			if strings.EqualFold(view, tableName) {
				viewSchema = schema.GetView(view)
				break
			}
		}
	} else {
		viewSchema = schema.GetView(tableName)
	}
	if viewSchema != nil {
		columns, err := q.getColumnsForView(viewSchema.Definition)
		if err != nil {
			return nil, errors.Wrapf(err, "failed to get columns for view %q", tableName)
		}
		return &base.PhysicalView{
			Name:     viewSchema.Name,
			Schema:   emptySchema,
			Database: dbMetadata.GetProto().GetName(),
			Server:   "",
			Columns:  columns,
		}, nil
	}

	return nil, &base.ResourceNotFoundError{
		Database: &databaseName,
		Schema:   &emptySchema,
		Table:    &tableName,
	}
}

func (q *querySpanExtractor) getColumnsForView(definition string) ([]base.QuerySpanResult, error) {
	span, err := newOmniQuerySpanExtractor(q.defaultDatabase, q.gCtx, q.ignoreCaseSensitive).getOmniQuerySpan(q.ctx, definition)
	if err != nil {
		return nil, errors.Wrap(err, "failed to get query span for view")
	}
	if span.NotFoundError != nil {
		return nil, span.NotFoundError
	}
	return span.Results, nil
}

func isMixedQuery(m base.SourceColumnSet, ignoreCaseSensitive bool) (bool, bool) {
	hasSystem, hasUser := false, false
	for table := range m {
		if isSystemResource(table, ignoreCaseSensitive) {
			hasSystem = true
		} else {
			hasUser = true
		}
	}
	if hasSystem && hasUser {
		return false, true
	}
	return !hasUser && hasSystem, false
}

var reservedSystemDatabases = map[string]bool{
	"information_schema": true,
	"performance_schema": true,
}

var onDiskSystemDatabases = map[string]bool{
	"mysql": true,
}

func isSystemResource(resource base.ColumnResource, ignoreCaseSensitive bool) bool {
	if reservedSystemDatabases[strings.ToLower(resource.Database)] {
		return true
	}
	database := resource.Database
	if ignoreCaseSensitive {
		database = strings.ToLower(database)
	}
	return onDiskSystemDatabases[database]
}
