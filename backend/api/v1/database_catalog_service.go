package v1

import (
	"context"
	"fmt"
	"log/slog"
	"strings"

	"connectrpc.com/connect"
	"github.com/pkg/errors"

	"github.com/bytebase/bytebase/backend/common"
	storepb "github.com/bytebase/bytebase/backend/generated-go/store"
	v1pb "github.com/bytebase/bytebase/backend/generated-go/v1"
	"github.com/bytebase/bytebase/backend/generated-go/v1/v1connect"
	"github.com/bytebase/bytebase/backend/store"
)

// DatabaseCatalogService implements the database catalog service.
type DatabaseCatalogService struct {
	v1connect.UnimplementedDatabaseCatalogServiceHandler
	store *store.Store
}

// NewDatabaseCatalogService creates a new DatabaseCatalogService.
func NewDatabaseCatalogService(store *store.Store) *DatabaseCatalogService {
	return &DatabaseCatalogService{
		store: store,
	}
}

// GetDatabaseCatalog gets a database catalog.
func (s *DatabaseCatalogService) GetDatabaseCatalog(ctx context.Context, req *connect.Request[v1pb.GetDatabaseCatalogRequest]) (*connect.Response[v1pb.DatabaseCatalog], error) {
	databaseResourceName, err := common.TrimSuffix(req.Msg.Name, common.CatalogSuffix)
	if err != nil {
		return nil, connect.NewError(connect.CodeInvalidArgument, err)
	}

	instanceID, databaseName, err := common.GetInstanceDatabaseID(databaseResourceName)
	if err != nil {
		return nil, connect.NewError(connect.CodeInvalidArgument, errors.Wrapf(err, "failed to parse %q", databaseResourceName))
	}
	database, err := s.store.GetDatabase(ctx, &store.FindDatabaseMessage{
		InstanceID:   &instanceID,
		DatabaseName: &databaseName,
		ShowDeleted:  true,
	})
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, errors.Wrapf(err, "failed to get database"))
	}
	if database == nil {
		return nil, connect.NewError(connect.CodeNotFound, errors.Errorf("database %q not found", databaseResourceName))
	}
	dbMetadata, err := s.store.GetDBSchema(ctx, &store.FindDBSchemaMessage{
		InstanceID:   database.InstanceID,
		DatabaseName: database.DatabaseName,
	})
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, err)
	}
	if dbMetadata == nil {
		return connect.NewResponse(&v1pb.DatabaseCatalog{
			Name: fmt.Sprintf("%s%s/%s%s%s", common.InstanceNamePrefix, database.InstanceID, common.DatabaseIDPrefix, database.DatabaseName, common.CatalogSuffix),
		}), nil
	}

	return connect.NewResponse(convertDatabaseConfig(database, dbMetadata.GetConfig())), nil
}

// UpdateDatabaseCatalog updates a database catalog.
func (s *DatabaseCatalogService) UpdateDatabaseCatalog(ctx context.Context, req *connect.Request[v1pb.UpdateDatabaseCatalogRequest]) (*connect.Response[v1pb.DatabaseCatalog], error) {
	databaseResourceName, err := common.TrimSuffix(req.Msg.GetCatalog().GetName(), common.CatalogSuffix)
	if err != nil {
		return nil, connect.NewError(connect.CodeInvalidArgument, err)
	}

	instanceID, databaseName, err := common.GetInstanceDatabaseID(databaseResourceName)
	if err != nil {
		return nil, connect.NewError(connect.CodeInvalidArgument, errors.Wrapf(err, "failed to parse %q", databaseResourceName))
	}
	database, err := s.store.GetDatabase(ctx, &store.FindDatabaseMessage{
		InstanceID:   &instanceID,
		DatabaseName: &databaseName,
	})
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, errors.Wrapf(err, "failed to get database"))
	}
	if database == nil {
		return nil, connect.NewError(connect.CodeNotFound, errors.Errorf("database %q not found", databaseResourceName))
	}

	dbMetadata, err := s.store.GetDBSchema(ctx, &store.FindDBSchemaMessage{
		InstanceID:   database.InstanceID,
		DatabaseName: database.DatabaseName,
	})
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, err)
	}
	if dbMetadata == nil {
		return nil, connect.NewError(connect.CodeFailedPrecondition, errors.New("database schema metadata not found"))
	}

	databaseConfig := convertDatabaseCatalog(req.Msg.GetCatalog())
	databaseConfig = normalizeCatalogSchemaNames(databaseConfig, dbMetadata.GetProto())

	if err := s.store.UpdateDBSchema(ctx, database.InstanceID, database.DatabaseName, &store.UpdateDBSchemaMessage{Config: databaseConfig}); err != nil {
		return nil, connect.NewError(connect.CodeInternal, err)
	}

	return connect.NewResponse(convertDatabaseConfig(database, databaseConfig)), nil
}

func convertDatabaseConfig(database *store.DatabaseMessage, config *storepb.DatabaseConfig) *v1pb.DatabaseCatalog {
	c := &v1pb.DatabaseCatalog{
		Name: fmt.Sprintf("%s%s/%s%s%s", common.InstanceNamePrefix, database.InstanceID, common.DatabaseIDPrefix, database.DatabaseName, common.CatalogSuffix),
	}
	for _, sc := range config.Schemas {
		s := &v1pb.SchemaCatalog{Name: sc.Name}
		for _, tc := range sc.Tables {
			s.Tables = append(s.Tables, convertTableCatalog(tc))
		}
		c.Schemas = append(c.Schemas, s)
	}
	return c
}

func convertTableCatalog(t *storepb.TableCatalog) *v1pb.TableCatalog {
	tc := &v1pb.TableCatalog{
		Name:           t.Name,
		Classification: t.Classification,
	}
	if t.ObjectSchema != nil && len(t.Columns) == 0 {
		tc.Kind = &v1pb.TableCatalog_ObjectSchema{ObjectSchema: convertStoreObjectSchema(t.ObjectSchema)}
	} else {
		var columns []*v1pb.ColumnCatalog
		for _, cc := range t.Columns {
			columns = append(columns, convertColumnCatalog(cc))
		}
		tc.Kind = &v1pb.TableCatalog_Columns_{Columns: &v1pb.TableCatalog_Columns{
			Columns: columns,
		}}
	}
	return tc
}

func convertColumnCatalog(c *storepb.ColumnCatalog) *v1pb.ColumnCatalog {
	return &v1pb.ColumnCatalog{
		Name:           c.Name,
		SemanticType:   c.SemanticType,
		Labels:         c.Labels,
		Classification: c.Classification,
		ObjectSchema:   convertStoreObjectSchema(c.ObjectSchema),
	}
}

func convertStoreObjectSchema(objectSchema *storepb.ObjectSchema) *v1pb.ObjectSchema {
	if objectSchema == nil {
		return nil
	}
	o := &v1pb.ObjectSchema{}
	switch objectSchema.Type {
	case storepb.ObjectSchema_STRING:
		o.Type = v1pb.ObjectSchema_STRING
	case storepb.ObjectSchema_NUMBER:
		o.Type = v1pb.ObjectSchema_NUMBER
	case storepb.ObjectSchema_BOOLEAN:
		o.Type = v1pb.ObjectSchema_BOOLEAN
	case storepb.ObjectSchema_OBJECT:
		o.Type = v1pb.ObjectSchema_OBJECT
	case storepb.ObjectSchema_ARRAY:
		o.Type = v1pb.ObjectSchema_ARRAY
	default:
	}
	switch objectSchema.Kind.(type) {
	case *storepb.ObjectSchema_StructKind_:
		properties := make(map[string]*v1pb.ObjectSchema)
		for k, v := range objectSchema.GetStructKind().Properties {
			properties[k] = convertStoreObjectSchema(v)
		}
		o.Kind = &v1pb.ObjectSchema_StructKind_{
			StructKind: &v1pb.ObjectSchema_StructKind{Properties: properties},
		}
	case *storepb.ObjectSchema_ArrayKind_:
		o.Kind = &v1pb.ObjectSchema_ArrayKind_{
			ArrayKind: &v1pb.ObjectSchema_ArrayKind{
				Kind: convertStoreObjectSchema(objectSchema.GetArrayKind().GetKind()),
			},
		}
	default:
	}
	o.SemanticType = objectSchema.SemanticType
	return o
}

func convertDatabaseCatalog(catalog *v1pb.DatabaseCatalog) *storepb.DatabaseConfig {
	c := &storepb.DatabaseConfig{}
	for _, sc := range catalog.Schemas {
		s := &storepb.SchemaCatalog{Name: sc.Name}
		for _, tc := range sc.Tables {
			s.Tables = append(s.Tables, convertV1TableCatalog(tc))
		}
		c.Schemas = append(c.Schemas, s)
	}
	return c
}

// normalizeCatalogSchemaNames fixes empty schema names in the catalog config
// by resolving them against the actual database metadata. This prevents catalog
// corruption where columns end up under a nameless schema entry.
func normalizeCatalogSchemaNames(config *storepb.DatabaseConfig, metadata *storepb.DatabaseSchemaMetadata) *storepb.DatabaseConfig {
	if metadata == nil || !hasEmptySchemaName(config.Schemas) {
		return config
	}
	// Engines like Cassandra/MySQL legitimately use empty schema names.
	if hasEmptySchemaName(metadata.Schemas) {
		return config
	}

	tableToSchema, ambiguousTables := buildTableToSchemaMap(metadata)
	return resolveEmptySchemaNames(config, tableToSchema, ambiguousTables, defaultSchemaFromSearchPath(metadata))
}

func hasEmptySchemaName[T interface{ GetName() string }](schemas []T) bool {
	for _, s := range schemas {
		if s.GetName() == "" {
			return true
		}
	}
	return false
}

// buildTableToSchemaMap creates a table name to schema name lookup from metadata.
// Returns two maps: unambiguous (table→schema for unique tables) and ambiguous
// (tables that exist in multiple schemas). Callers use the ambiguous set to
// distinguish "table in multiple schemas" from "table not in any schema".
func buildTableToSchemaMap(metadata *storepb.DatabaseSchemaMetadata) (unambiguous map[string]string, ambiguous map[string]bool) {
	m := make(map[string]string)
	amb := make(map[string]bool)
	for _, ms := range metadata.Schemas {
		for _, mt := range ms.Tables {
			if _, exists := m[mt.Name]; exists {
				amb[mt.Name] = true
			}
			m[mt.Name] = ms.Name
		}
	}
	for name := range amb {
		delete(m, name)
	}
	return m, amb
}

// defaultSchemaFromSearchPath returns the first concrete schema name from the
// metadata search path (skipping "$user" and empty entries). This is used as
// a fallback when table-based resolution is ambiguous.
func defaultSchemaFromSearchPath(metadata *storepb.DatabaseSchemaMetadata) string {
	if metadata.SearchPath == "" {
		return ""
	}
	for _, part := range strings.Split(metadata.SearchPath, ",") {
		schema := strings.TrimSpace(part)
		// Strip quotes.
		schema = strings.Trim(schema, "\"'")
		schema = strings.TrimSpace(schema)
		// Skip $user placeholder and empty entries.
		if schema == "" || schema == "$user" {
			continue
		}
		return schema
	}
	return ""
}

// resolveEmptySchemaNames resolves empty schema names by placing each table
// into its correct schema individually. This handles the case where a single
// empty-name schema entry contains tables from different real schemas.
//
// Resolution per table:
//   - Unambiguous match in metadata → assign to that schema
//   - Ambiguous (exists in multiple schemas) → assign to fallbackSchema if available
//   - Unknown (not in any metadata schema) → keep in empty-name schema
func resolveEmptySchemaNames(config *storepb.DatabaseConfig, tableToSchema map[string]string, ambiguousTables map[string]bool, fallbackSchema string) *storepb.DatabaseConfig {
	schemaMap := make(map[string]*storepb.SchemaCatalog)
	var schemaOrder []string

	addTable := func(name string, tc *storepb.TableCatalog) {
		if existing, ok := schemaMap[name]; ok {
			existing.Tables = mergeTableCatalogs(existing.Tables, []*storepb.TableCatalog{tc})
		} else {
			schemaMap[name] = &storepb.SchemaCatalog{Name: name, Tables: []*storepb.TableCatalog{tc}}
			schemaOrder = append(schemaOrder, name)
		}
	}

	for _, sc := range config.Schemas {
		if sc.Name != "" {
			if existing, ok := schemaMap[sc.Name]; ok {
				existing.Tables = mergeTableCatalogs(existing.Tables, sc.Tables)
			} else {
				schemaMap[sc.Name] = &storepb.SchemaCatalog{Name: sc.Name, Tables: sc.Tables}
				schemaOrder = append(schemaOrder, sc.Name)
			}
			continue
		}

		for _, tc := range sc.Tables {
			target := resolveTableSchema(tc.Name, tableToSchema, ambiguousTables, fallbackSchema)
			if target != "" {
				slog.Warn("resolved empty catalog schema table", slog.String("table", tc.Name), slog.String("resolvedSchema", target))
			}
			addTable(target, tc)
		}
	}

	result := &storepb.DatabaseConfig{}
	for _, name := range schemaOrder {
		result.Schemas = append(result.Schemas, schemaMap[name])
	}
	return result
}

// resolveTableSchema determines the target schema for a single table.
// Unknown tables (not in any metadata schema) are left unresolved rather
// than assigned to fallbackSchema — they may be stale catalog entries.
func resolveTableSchema(tableName string, tableToSchema map[string]string, ambiguousTables map[string]bool, fallbackSchema string) string {
	if name, ok := tableToSchema[tableName]; ok {
		return name
	}
	if ambiguousTables[tableName] && fallbackSchema != "" {
		return fallbackSchema
	}
	return ""
}

// mergeTableCatalogs merges two table catalog lists. When both lists contain
// the same table, columns are merged (override wins per column name) to avoid
// losing column-level config from either side.
func mergeTableCatalogs(base, override []*storepb.TableCatalog) []*storepb.TableCatalog {
	tableMap := make(map[string]*storepb.TableCatalog, len(base))
	var order []string
	for _, t := range base {
		tableMap[t.Name] = t
		order = append(order, t.Name)
	}
	for _, t := range override {
		if existing, ok := tableMap[t.Name]; ok {
			mergeColumnCatalogs(existing, t)
		} else {
			tableMap[t.Name] = t
			order = append(order, t.Name)
		}
	}
	result := make([]*storepb.TableCatalog, 0, len(order))
	for _, name := range order {
		result = append(result, tableMap[name])
	}
	return result
}

// mergeColumnCatalogs merges columns from src into dst in-place, mutating dst.
// For duplicate column names, the src entry wins. Non-column fields
// (classification) are taken from src if non-empty.
func mergeColumnCatalogs(dst, src *storepb.TableCatalog) {
	if src.Classification != "" {
		dst.Classification = src.Classification
	}
	if len(src.Columns) == 0 {
		return
	}
	colMap := make(map[string]int, len(dst.Columns))
	for i, c := range dst.Columns {
		colMap[c.Name] = i
	}
	for _, c := range src.Columns {
		if idx, ok := colMap[c.Name]; ok {
			dst.Columns[idx] = c
		} else {
			dst.Columns = append(dst.Columns, c)
		}
	}
}

func convertV1TableCatalog(t *v1pb.TableCatalog) *storepb.TableCatalog {
	tc := &storepb.TableCatalog{
		Name:           t.Name,
		Classification: t.Classification,
	}
	if t.GetObjectSchema() != nil && len(t.GetColumns().GetColumns()) == 0 {
		tc.ObjectSchema = convertV1ObjectSchema(t.GetObjectSchema())
	} else {
		for _, cc := range t.GetColumns().GetColumns() {
			tc.Columns = append(tc.Columns, convertV1ColumnCatalog(cc))
		}
	}
	return tc
}

func convertV1ColumnCatalog(c *v1pb.ColumnCatalog) *storepb.ColumnCatalog {
	return &storepb.ColumnCatalog{
		Name:           c.Name,
		SemanticType:   c.SemanticType,
		Labels:         c.Labels,
		Classification: c.Classification,
		ObjectSchema:   convertV1ObjectSchema(c.GetObjectSchema()),
	}
}

func convertV1ObjectSchema(objectSchema *v1pb.ObjectSchema) *storepb.ObjectSchema {
	if objectSchema == nil {
		return nil
	}
	o := &storepb.ObjectSchema{}
	switch objectSchema.Type {
	case v1pb.ObjectSchema_STRING:
		o.Type = storepb.ObjectSchema_STRING
	case v1pb.ObjectSchema_NUMBER:
		o.Type = storepb.ObjectSchema_NUMBER
	case v1pb.ObjectSchema_BOOLEAN:
		o.Type = storepb.ObjectSchema_BOOLEAN
	case v1pb.ObjectSchema_OBJECT:
		o.Type = storepb.ObjectSchema_OBJECT
	case v1pb.ObjectSchema_ARRAY:
		o.Type = storepb.ObjectSchema_ARRAY
	default:
	}
	switch objectSchema.Kind.(type) {
	case *v1pb.ObjectSchema_StructKind_:
		properties := make(map[string]*storepb.ObjectSchema)
		for k, v := range objectSchema.GetStructKind().Properties {
			properties[k] = convertV1ObjectSchema(v)
		}
		o.Kind = &storepb.ObjectSchema_StructKind_{
			StructKind: &storepb.ObjectSchema_StructKind{Properties: properties},
		}
	case *v1pb.ObjectSchema_ArrayKind_:
		o.Kind = &storepb.ObjectSchema_ArrayKind_{
			ArrayKind: &storepb.ObjectSchema_ArrayKind{
				Kind: convertV1ObjectSchema(objectSchema.GetArrayKind().GetKind()),
			},
		}
	default:
	}
	o.SemanticType = objectSchema.SemanticType
	return o
}
