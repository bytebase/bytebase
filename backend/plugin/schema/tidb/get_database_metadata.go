package tidb

import (
	"fmt"
	"slices"
	"strings"

	metadatapb "github.com/bytebase/omni/metadata"
	"github.com/bytebase/omni/tidb/catalog"
	"github.com/pkg/errors"

	storepb "github.com/bytebase/bytebase/backend/generated-go/store"
	"github.com/bytebase/bytebase/backend/plugin/schema"
)

func init() {
	schema.RegisterGetDatabaseMetadata(storepb.Engine_TIDB, GetDatabaseMetadata)
}

// catalogDBName is the scratch database the schema DDL is simulated in when it
// selects none of its own. The name is deliberately unlikely to appear as a
// real table, alias or literal, so stripping its view-definition qualifier
// (tidbViewDefinition) is unambiguous.
const catalogDBName = "_bytebase_schema_scratch"

// GetDatabaseMetadata parses TiDB schema DDL text and returns database metadata
// using the omni TiDB catalog.
func GetDatabaseMetadata(schemaText string) (*metadatapb.DatabaseSchemaMetadata, error) {
	if strings.TrimSpace(schemaText) == "" {
		return &metadatapb.DatabaseSchemaMetadata{}, nil
	}

	c := catalog.New()
	initSQL := fmt.Sprintf("SET foreign_key_checks = 0;\nCREATE DATABASE IF NOT EXISTS `%s`;\nUSE `%s`;", catalogDBName, catalogDBName)
	if _, err := c.Exec(initSQL, nil); err != nil {
		return nil, errors.Wrap(err, "failed to initialize catalog")
	}

	results, err := c.Exec(schemaText, &catalog.ExecOptions{ContinueOnError: true})
	if err != nil {
		return nil, errors.Wrap(err, "failed to parse schema DDL")
	}
	for _, r := range results {
		if r.Error != nil {
			return nil, errors.Wrap(r.Error, "failed to execute schema DDL")
		}
	}

	// The DDL may select its own database (CREATE DATABASE app; USE app; ...),
	// in which case its objects live there rather than in the scratch database.
	// Read back whichever database the DDL ended in.
	return tidbCatalogToProto(c, c.CurrentDatabase()), nil
}

// tidbCatalogToProto converts the omni TiDB catalog state to a
// DatabaseSchemaMetadata, mirroring the MySQL schema package: the database is
// the scratch database the DDL was simulated in, and the single empty-name
// schema holds the objects.
func tidbCatalogToProto(c *catalog.Catalog, dbName string) *metadatapb.DatabaseSchemaMetadata {
	// When the DDL selected no database of its own, the objects sit in the
	// scratch database; the returned metadata is then schema-relative (empty
	// database name), as the old extractor produced it.
	name := dbName
	if dbName == catalogDBName {
		name = ""
	}
	dbMeta := &metadatapb.DatabaseSchemaMetadata{Name: name}
	db := c.GetDatabase(dbName)
	if db == nil {
		return dbMeta
	}
	dbMeta.CharacterSet = db.Charset
	dbMeta.Collation = db.Collation

	schemaMeta := &metadatapb.SchemaMetadata{Name: ""}

	for _, name := range sortedKeys(db.Tables) {
		schemaMeta.Tables = append(schemaMeta.Tables, tidbTableToProto(db.Tables[name]))
	}
	for _, name := range sortedKeys(db.Views) {
		v := db.Views[name]
		schemaMeta.Views = append(schemaMeta.Views, &metadatapb.ViewMetadata{
			Name:       v.Name,
			Definition: tidbViewDefinition(v.Definition, dbName),
		})
	}
	for _, name := range sortedKeys(db.Functions) {
		f := db.Functions[name]
		schemaMeta.Functions = append(schemaMeta.Functions, &metadatapb.FunctionMetadata{Name: f.Name, Definition: f.Body})
	}
	for _, name := range sortedKeys(db.Procedures) {
		p := db.Procedures[name]
		schemaMeta.Procedures = append(schemaMeta.Procedures, &metadatapb.ProcedureMetadata{Name: p.Name, Definition: p.Body})
	}

	dbMeta.Schemas = append(dbMeta.Schemas, schemaMeta)
	return dbMeta
}

// tidbViewDefinition drops the qualifier omni adds to a view body's
// same-database references ONLY when that database is the throwaway scratch one,
// so the definition does not mention it. When the DDL selected a real database,
// its qualifier is left intact — live TiDB qualifies same-database references in
// information_schema.VIEWS the same way, so the stored definition matches sync.
// The scratch name cannot occur as a real alias or literal, so this replacement
// is unambiguous.
func tidbViewDefinition(definition, dbName string) string {
	if dbName != catalogDBName {
		return definition
	}
	return strings.ReplaceAll(definition, fmt.Sprintf("`%s`.", dbName), "")
}

func tidbTableToProto(t *catalog.Table) *metadatapb.TableMetadata {
	table := &metadatapb.TableMetadata{
		Name:      t.Name,
		Engine:    t.Engine,
		Collation: t.Collation,
		Comment:   t.Comment,
	}

	for _, col := range t.Columns {
		colMeta := &metadatapb.ColumnMetadata{
			Name:         col.Name,
			Position:     int32(col.Position),
			Type:         col.ColumnType,
			Nullable:     col.Nullable,
			CharacterSet: col.Charset,
			Collation:    col.Collation,
			Comment:      col.Comment,
		}
		switch {
		case col.AutoRandom:
			// TiDB reports an AUTO_RANDOM column's default as AUTO_RANDOM(<shard bits>).
			colMeta.Default = fmt.Sprintf("AUTO_RANDOM(%d)", col.AutoRandomShardBits)
		case col.AutoIncrement:
			colMeta.Default = "AUTO_INCREMENT"
		case col.Default != nil:
			colMeta.Default = *col.Default
		default:
		}
		if col.OnUpdate != "" {
			colMeta.OnUpdate = col.OnUpdate
		}
		if col.Generated != nil {
			genType := metadatapb.GenerationMetadata_TYPE_VIRTUAL
			if col.Generated.Stored {
				genType = metadatapb.GenerationMetadata_TYPE_STORED
			}
			colMeta.Generation = &metadatapb.GenerationMetadata{
				Type:       genType,
				Expression: col.Generated.Expr,
			}
		}
		colMeta.IsInvisible = col.Invisible
		table.Columns = append(table.Columns, colMeta)
	}

	for _, idx := range t.Indexes {
		indexType := strings.ToUpper(idx.IndexType)
		if indexType == "" {
			indexType = "BTREE"
		}
		idxMeta := &metadatapb.IndexMetadata{
			Name:    idx.Name,
			Type:    indexType,
			Unique:  idx.Unique,
			Primary: idx.Primary,
			Visible: idx.Visible,
			Comment: idx.Comment,
		}
		for _, col := range idx.Columns {
			if col.Expr != "" {
				idxMeta.Expressions = append(idxMeta.Expressions, col.Expr)
			} else {
				idxMeta.Expressions = append(idxMeta.Expressions, col.Name)
			}
			keyLength := int64(col.Length)
			if keyLength == 0 {
				keyLength = -1
			}
			idxMeta.KeyLength = append(idxMeta.KeyLength, keyLength)
			idxMeta.Descending = append(idxMeta.Descending, col.Descending)
		}
		table.Indexes = append(table.Indexes, idxMeta)
	}

	for _, con := range t.Constraints {
		switch con.Type {
		case catalog.ConForeignKey:
			fk := &metadatapb.ForeignKeyMetadata{
				Name:              con.Name,
				Columns:           con.Columns,
				ReferencedTable:   con.RefTable,
				ReferencedColumns: con.RefColumns,
				OnUpdate:          con.OnUpdate,
				OnDelete:          con.OnDelete,
				MatchType:         con.MatchType,
			}
			if con.RefDatabase != "" {
				fk.ReferencedSchema = con.RefDatabase
			}
			table.ForeignKeys = append(table.ForeignKeys, fk)
		case catalog.ConCheck:
			table.CheckConstraints = append(table.CheckConstraints, &metadatapb.CheckConstraintMetadata{
				Name:       con.Name,
				Expression: con.CheckExpr,
			})
		default:
		}
	}

	if t.Partitioning != nil {
		table.Partitions = tidbPartitionsToProto(t.Partitioning)
	}

	return table
}

func tidbPartitionsToProto(p *catalog.PartitionInfo) []*metadatapb.TablePartitionMetadata {
	expr := p.Expr
	if expr == "" && len(p.Columns) > 0 {
		expr = strings.Join(p.Columns, ",")
	}
	var result []*metadatapb.TablePartitionMetadata
	for _, pd := range p.Partitions {
		result = append(result, &metadatapb.TablePartitionMetadata{
			Name:       pd.Name,
			Type:       tidbPartitionTypeToProto(p.Type),
			Expression: expr,
			Value:      pd.ValueExpr,
		})
	}
	return result
}

func tidbPartitionTypeToProto(t string) metadatapb.TablePartitionMetadata_Type {
	switch strings.ToUpper(t) {
	case "RANGE":
		return metadatapb.TablePartitionMetadata_RANGE
	case "RANGE COLUMNS":
		return metadatapb.TablePartitionMetadata_RANGE_COLUMNS
	case "LIST":
		return metadatapb.TablePartitionMetadata_LIST
	case "LIST COLUMNS":
		return metadatapb.TablePartitionMetadata_LIST_COLUMNS
	case "HASH":
		return metadatapb.TablePartitionMetadata_HASH
	case "KEY":
		return metadatapb.TablePartitionMetadata_KEY
	default:
		return metadatapb.TablePartitionMetadata_TYPE_UNSPECIFIED
	}
}

func sortedKeys[V any](m map[string]V) []string {
	keys := make([]string, 0, len(m))
	for k := range m {
		keys = append(keys, k)
	}
	slices.Sort(keys)
	return keys
}
