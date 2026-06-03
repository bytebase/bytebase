package plsql

import (
	"context"
	"strings"

	"github.com/pkg/errors"

	"github.com/bytebase/bytebase/backend/plugin/parser/base"
	"github.com/bytebase/bytebase/backend/store/model"
)

type querySpanExtractor struct {
	ctx             context.Context
	gCtx            base.GetQuerySpanContext
	defaultDatabase string

	ctes []*base.PseudoTable

	outerTableSources []base.TableSource
	tableSourcesFrom  []base.TableSource
}

func newQuerySpanExtractor(connectionDatabase string, gCtx base.GetQuerySpanContext) *querySpanExtractor {
	return &querySpanExtractor{
		defaultDatabase: connectionDatabase,
		gCtx:            gCtx,
	}
}

func (q *querySpanExtractor) getLinkedDatabaseMetadata(linkName string, schema string) (string, string, *model.DatabaseMetadata, error) {
	linkedInstanceID, databaseName, meta, err := q.gCtx.GetLinkedDatabaseMetadataFunc(q.ctx, q.gCtx.InstanceID, linkName, schema)
	if err != nil {
		return "", "", nil, errors.Wrapf(err, "failed to get linked database metadata for schema: %s", schema)
	}
	return linkedInstanceID, databaseName, meta, nil
}

func (q *querySpanExtractor) getDatabaseMetadata(schema string) (*model.DatabaseMetadata, error) {
	_, meta, err := q.gCtx.GetDatabaseMetadataFunc(q.ctx, q.gCtx.InstanceID, schema)
	if err != nil {
		return nil, errors.Wrapf(err, "failed to get database metadata for schema: %s", schema)
	}
	return meta, nil
}

func (q *querySpanExtractor) existsTableMetadata(resource base.SchemaResource) bool {
	if resource.Table == "DUAL" {
		return false
	}
	database := resource.Database
	if database == "" {
		database = q.defaultDatabase
	}
	meta, err := q.getDatabaseMetadata(database)
	if err != nil || meta == nil {
		return false
	}
	schema := meta.GetSchemaMetadata("")
	if schema == nil {
		return false
	}

	return schema.GetTable(resource.Table) != nil ||
		schema.GetView(resource.Table) != nil ||
		schema.GetMaterializedView(resource.Table) != nil ||
		schema.GetExternalTable(resource.Table) != nil
}

func isMixedQuery(m []base.SchemaResource) (bool, bool) {
	hasSystem, hasUser := false, false
	for _, item := range m {
		if systemSchemaMap[item.Database] {
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

var systemSchemaMap = map[string]bool{
	"ANONYMOUS":              true,
	"APPQOSSYS":              true,
	"AUDSYS":                 true,
	"CTXSYS":                 true,
	"DBSFWUSER":              true,
	"DBSNMP":                 true,
	"DGPDB_INT":              true,
	"DIP":                    true,
	"DVF":                    true,
	"DVSYS":                  true,
	"GGSYS":                  true,
	"GSMADMIN_INTERNAL":      true,
	"GSMCATUSER":             true,
	"GSMROOTUSER":            true,
	"GSMUSER":                true,
	"LBACSYS":                true,
	"MDDATA":                 true,
	"MDSYS":                  true,
	"OPS$ORACLE":             true,
	"ORACLE_OCM":             true,
	"OUTLN":                  true,
	"REMOTE_SCHEDULER_AGENT": true,
	"SYS":                    true,
	"SYS$UMF":                true,
	"SYSBACKUP":              true,
	"SYSDG":                  true,
	"SYSKM":                  true,
	"SYSRAC":                 true,
	"SYSTEM":                 true,
	"XDB":                    true,
	"XS$NULL":                true,
	"XS$$NULL":               true,
	"FLOWS_FILES":            true,
	"HR":                     true,
	"EXFSYS":                 true,
	"MGMT_VIEW":              true,
	"OLAPSYS":                true,
	"ORDDATA":                true,
	"ORDPLUGINS":             true,
	"ORDSYS":                 true,
	"OWBSYS":                 true,
	"OWBSYS_AUDIT":           true,
	"SCOTT":                  true,
	"SI_INFORMTN_SCHEMA":     true,
	"SPATIAL_CSW_ADMIN_USR":  true,
	"SPATIAL_WFS_ADMIN_USR":  true,
	"SYSMAN":                 true,
	"WMSYS":                  true,
	"OJVMSYS":                true,
}

func (q *querySpanExtractor) plsqlFindTableSchema(dbLink []string, schemaName, tableName string) (base.TableSource, error) {
	if tableName == "DUAL" {
		return &base.PseudoTable{
			Name:    "DUAL",
			Columns: []base.QuerySpanResult{},
		}, nil
	}
	if len(dbLink) > 0 {
		linkName := strings.Join(dbLink, ".")
		linkedInstanceID, _, linkedMeta, err := q.getLinkedDatabaseMetadata(linkName, schemaName)
		if err != nil {
			return nil, errors.Wrapf(err, "failed to get linked database metadata for: %s", dbLink)
		}
		if linkedMeta == nil {
			return nil, &base.ResourceNotFoundError{
				DatabaseLink: &linkName,
			}
		}
		return q.findTableSchemaInMetadata(linkedInstanceID, linkedMeta, schemaName, tableName)
	}

	if schemaName == q.defaultDatabase {
		for i := len(q.ctes) - 1; i >= 0; i-- {
			table := q.ctes[i]
			if table.Name == tableName {
				return table, nil
			}
		}
	}

	dbMetadata, err := q.getDatabaseMetadata(schemaName)
	if err != nil {
		return nil, errors.Wrapf(err, "failed to get database metadata for: %s", schemaName)
	}
	if dbMetadata == nil {
		return nil, &base.ResourceNotFoundError{
			Database: &schemaName,
		}
	}

	return q.findTableSchemaInMetadata(q.gCtx.InstanceID, dbMetadata, schemaName, tableName)
}

func (q *querySpanExtractor) findTableSchemaInMetadata(instanceID string, dbMetadata *model.DatabaseMetadata, databaseName, tableName string) (base.TableSource, error) {
	schema := dbMetadata.GetSchemaMetadata("")
	if schema == nil {
		return nil, &base.ResourceNotFoundError{
			Database: &databaseName,
		}
	}
	table := schema.GetTable(tableName)
	view := schema.GetView(tableName)
	materializedView := schema.GetMaterializedView(tableName)
	foreignTable := schema.GetExternalTable(tableName)
	if table == nil && view == nil && materializedView == nil && foreignTable == nil {
		return nil, &base.ResourceNotFoundError{
			Database: &databaseName,
			Table:    &tableName,
		}
	}

	if table != nil {
		var columns []string
		for _, column := range table.GetProto().GetColumns() {
			columns = append(columns, column.Name)
		}
		return &base.PhysicalTable{
			Server:   "",
			Database: databaseName,
			Name:     table.GetProto().Name,
			Columns:  columns,
		}, nil
	}

	if foreignTable != nil {
		var columns []string
		for _, column := range foreignTable.GetProto().GetColumns() {
			columns = append(columns, column.Name)
		}
		return &base.PhysicalTable{
			Server:   "",
			Database: databaseName,
			Name:     foreignTable.GetProto().Name,
			Columns:  columns,
		}, nil
	}

	if view != nil && view.Definition != "" {
		columns, err := q.getColumnsForView(instanceID, databaseName, view.Definition)
		if err != nil {
			return nil, err
		}
		return &base.PseudoTable{
			Name:    view.Name,
			Columns: columns,
		}, nil
	}

	if materializedView != nil && materializedView.Definition != "" {
		columns, err := q.getColumnsForMaterializedView(instanceID, databaseName, materializedView.Definition)
		if err != nil {
			return nil, err
		}
		return &base.PseudoTable{
			Name:    materializedView.Name,
			Columns: columns,
		}, nil
	}
	return nil, nil
}

func (q *querySpanExtractor) getColumnsForView(instanceID, defaultDatabase, definition string) ([]base.QuerySpanResult, error) {
	newContext := base.GetQuerySpanContext{
		InstanceID:                    instanceID,
		GetDatabaseMetadataFunc:       q.gCtx.GetDatabaseMetadataFunc,
		ListDatabaseNamesFunc:         q.gCtx.ListDatabaseNamesFunc,
		GetLinkedDatabaseMetadataFunc: q.gCtx.GetLinkedDatabaseMetadataFunc,
	}
	newQ := newOmniQuerySpanExtractor(defaultDatabase, newContext)
	span, err := newQ.getOmniQuerySpan(q.ctx, definition)
	if err != nil {
		return nil, errors.Wrapf(err, "failed to get query span for view definition: %s", definition)
	}
	if span.NotFoundError != nil {
		return nil, span.NotFoundError
	}
	return span.Results, nil
}

func (q *querySpanExtractor) getColumnsForMaterializedView(instanceID, defaultDatabase, definition string) ([]base.QuerySpanResult, error) {
	newContext := base.GetQuerySpanContext{
		InstanceID:                    instanceID,
		GetDatabaseMetadataFunc:       q.gCtx.GetDatabaseMetadataFunc,
		ListDatabaseNamesFunc:         q.gCtx.ListDatabaseNamesFunc,
		GetLinkedDatabaseMetadataFunc: q.gCtx.GetLinkedDatabaseMetadataFunc,
	}
	newQ := newOmniQuerySpanExtractor(defaultDatabase, newContext)
	span, err := newQ.getOmniQuerySpan(q.ctx, definition)
	if err != nil {
		return nil, errors.Wrapf(err, "failed to get query span for materialized view definition: %s", definition)
	}
	if span.NotFoundError != nil {
		return nil, span.NotFoundError
	}
	return span.Results, nil
}
