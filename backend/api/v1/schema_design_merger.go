package v1

import (
	"fmt"
	"reflect"

	"github.com/pkg/errors"

	v1pb "github.com/bytebase/bytebase/proto/generated-go/v1"
)

type metadataDiffNode interface {
	isConflictWith(other metadataDiffNode) (bool, string)
}

type diffAction string

const (
	diffActionCreate diffAction = "CREATE"
	diffActionUpdate diffAction = "UPDATE"
	diffActionDrop   diffAction = "DROP"
)

type metadataDiffBaseNode struct {
	action diffAction
}

type metadataDiffRootNode struct {
	schemas map[string]*metadataDiffSchemaNode
}

func (mr *metadataDiffRootNode) isConflictWith(other *metadataDiffRootNode) (bool, string) {
	for _, schema := range mr.schemas {
		otherSchema, in := other.schemas[schema.name]
		if !in {
			continue
		}
		conflict, msg := schema.isConflictWith(otherSchema)
		if conflict {
			return true, msg
		}
	}
	return false, ""
}

/* schema related */
type metadataDiffSchemaNode struct {
	metadataDiffBaseNode
	name string
	//nolint
	from *v1pb.SchemaMetadata
	to   *v1pb.SchemaMetadata

	tables map[string]metadataDiffNode

	// SchemaMetadata contains other object types, likes function, view and etc, but we do not support them yet.
}

func (n *metadataDiffSchemaNode) isConflictWith(other metadataDiffNode) (bool, string) {
	if other == nil {
		return true, "other node check conflict with schema node not be nil"
	}

	switch other := other.(type) {
	case *metadataDiffSchemaNode:
		if !(n.action == diffActionUpdate && other.action == diffActionUpdate) {
			return true, fmt.Sprintf("conflict schema action, one is %s, the other is %s", n.action, other.action)
		}
		if n.name != other.name {
			return true, fmt.Sprintf("conflict schema name, one is %s, the other is %s", n.name, other.name)
		}

		for tableName, tableNode := range n.tables {
			otherTableNode, in := other.tables[tableName]
			if !in {
				continue
			}
			conflict, msg := tableNode.isConflictWith(otherTableNode)
			if conflict {
				return true, msg
			}
		}
		return false, ""
	default:
		return true, fmt.Sprintf("non-expected node type pair, one is %T, the other is %T", n, other)
	}
}

/* table related */
type metadataDiffTableNode struct {
	metadataDiffBaseNode
	name string
	//nolint
	from *v1pb.TableMetadata
	to   *v1pb.TableMetadata

	columns     map[string]metadataDiffNode
	foreignKeys map[string]metadataDiffNode

	// TableMetaData contains other object types, likes trigger, index and etc, but we do not support them yet.
}

func (n *metadataDiffTableNode) isConflictWith(other metadataDiffNode) (bool, string) {
	if other == nil {
		return true, "other node check conflict with table node must not be nil"
	}

	switch other := other.(type) {
	case *metadataDiffTableNode:
		if !(n.action == diffActionUpdate && other.action == diffActionUpdate) {
			return true, fmt.Sprintf("conflict table action, one is %s, the other is %s", n.action, other.action)
		}
		if n.name != other.name {
			return true, fmt.Sprintf("conflict table name, one is %s, the other is %s", n.name, other.name)
		}

		for columnName, columnNode := range n.columns {
			otherColumnNode, in := other.columns[columnName]
			if !in {
				continue
			}
			conflict, msg := columnNode.isConflictWith(otherColumnNode)
			if conflict {
				return true, msg
			}
		}

		for foreignKeyName, foreignKeyNode := range n.foreignKeys {
			otherForeignKeyNode, in := other.foreignKeys[foreignKeyName]
			if !in {
				continue
			}
			conflict, msg := foreignKeyNode.isConflictWith(otherForeignKeyNode)
			if conflict {
				return true, msg
			}
		}
		return false, ""
	default:
		return true, fmt.Sprintf("non-expected node type pair, one is %T, the other is %T", n, other)
	}
}

/* column related */
type metadataDiffColumnNode struct {
	metadataDiffBaseNode
	name string
	//nolint
	from *v1pb.ColumnMetadata
	to   *v1pb.ColumnMetadata
}

func (n *metadataDiffColumnNode) isConflictWith(other metadataDiffNode) (bool, string) {
	if other == nil {
		return true, "other node check conflict with column node must not be nil"
	}

	switch other := other.(type) {
	case *metadataDiffColumnNode:
		if !(n.action == diffActionUpdate && other.action == diffActionUpdate) {
			return true, fmt.Sprintf("conflict column action, one is %s, the other is %s", n.action, other.action)
		}
		// TODO(zp): check update column conflict
		return true, "not implemented yet"
	default:
		return true, fmt.Sprintf("non-expected node type pair, one is %T, the other is %T", n, other)
	}
}

/* foreignKey related */
type metadataDiffForeignKeyNode struct {
	metadataDiffBaseNode
	name string
	//nolint
	from *v1pb.ForeignKeyMetadata
	to   *v1pb.ForeignKeyMetadata
}

func (n *metadataDiffForeignKeyNode) isConflictWith(other metadataDiffNode) (bool, string) {
	if other == nil {
		return true, "other node check conflict with column node must not be nil"
	}

	switch other := other.(type) {
	case *metadataDiffForeignKeyNode:
		if !(n.action == diffActionUpdate && other.action == diffActionUpdate) {
			return true, fmt.Sprintf("conflict column action, one is %s, the other is %s", n.action, other.action)
		}
		// TODO(zp): check update foreign key conflict
		return true, "not implemented yet"
	default:
		return true, fmt.Sprintf("non-expected node type pair, one is %T, the other is %T", n, other)
	}
}

func diffMetadata(from, to *v1pb.DatabaseMetadata) (*metadataDiffRootNode, error) {
	if from == nil || to == nil {
		return nil, errors.New("from and to database metadata must not be nil")
	}

	root := &metadataDiffRootNode{
		schemas: make(map[string]*metadataDiffSchemaNode),
	}

	fromSchemaMap := make(map[string]*v1pb.SchemaMetadata)
	for _, schema := range from.Schemas {
		fromSchemaMap[schema.Name] = schema
	}

	for _, schema := range to.Schemas {
		schemaInFrom, in := fromSchemaMap[schema.Name]
		if !in {
			root.schemas[schema.Name] = &metadataDiffSchemaNode{
				metadataDiffBaseNode: metadataDiffBaseNode{
					action: diffActionCreate,
				},
				name: schema.Name,
				to:   schema,
			}
			continue
		}
		diffNode, err := diffSchemaMetadata(schemaInFrom, schema)
		if err != nil {
			return nil, errors.Wrapf(err, "failed to diff schema %q", schema.Name)
		}
		if diffNode != nil {
			root.schemas[schema.Name] = diffNode
		}
		delete(fromSchemaMap, schema.Name)
	}

	for _, remainingSchema := range fromSchemaMap {
		root.schemas[remainingSchema.Name] = &metadataDiffSchemaNode{
			metadataDiffBaseNode: metadataDiffBaseNode{
				action: diffActionDrop,
			},
			name: remainingSchema.Name,
			from: remainingSchema,
		}
	}
	return root, nil
}

func diffSchemaMetadata(from, to *v1pb.SchemaMetadata) (*metadataDiffSchemaNode, error) {
	if from == nil || to == nil {
		return nil, errors.New("from and to schema metadata must not be nil")
	}

	schemaNode := &metadataDiffSchemaNode{
		name: to.Name,
		from: from,
		to:   to,

		tables: make(map[string]metadataDiffNode),
	}

	fromTableMap := make(map[string]*v1pb.TableMetadata)
	for _, table := range from.Tables {
		fromTableMap[table.Name] = table
	}

	for _, table := range to.Tables {
		tableInFrom, in := fromTableMap[table.Name]
		if !in {
			schemaNode.tables[table.Name] = &metadataDiffTableNode{
				metadataDiffBaseNode: metadataDiffBaseNode{
					action: diffActionCreate,
				},
				name: table.Name,
				to:   table,
			}
			continue
		}
		diffNode, err := diffTableMetadata(tableInFrom, table)
		if err != nil {
			return nil, errors.Wrapf(err, "failed to diff table %q", table.Name)
		}
		if diffNode != nil {
			schemaNode.tables[table.Name] = diffNode
		}
		delete(fromTableMap, table.Name)
	}

	for _, remainingTable := range fromTableMap {
		schemaNode.tables[remainingTable.Name] = &metadataDiffTableNode{
			metadataDiffBaseNode: metadataDiffBaseNode{
				action: diffActionDrop,
			},
			name: remainingTable.Name,
			from: remainingTable,
		}
	}

	if len(schemaNode.tables) > 0 {
		schemaNode.action = diffActionUpdate
		return schemaNode, nil
	}
	return nil, nil
}

func diffTableMetadata(from, to *v1pb.TableMetadata) (*metadataDiffTableNode, error) {
	if from == nil || to == nil {
		return nil, errors.New("from and to table metadata must not be nil")
	}

	tableNode := &metadataDiffTableNode{
		name: to.Name,
		from: from,
		to:   to,

		columns:     make(map[string]metadataDiffNode),
		foreignKeys: make(map[string]metadataDiffNode),
	}

	fromColumnMap := make(map[string]*v1pb.ColumnMetadata)
	for _, column := range from.Columns {
		fromColumnMap[column.Name] = column
	}
	fromForeignKeyMap := make(map[string]*v1pb.ForeignKeyMetadata)
	for _, foreignKey := range from.ForeignKeys {
		fromForeignKeyMap[foreignKey.Name] = foreignKey
	}

	for _, column := range to.Columns {
		columnInFrom, in := fromColumnMap[column.Name]
		if !in {
			tableNode.columns[column.Name] = &metadataDiffColumnNode{
				metadataDiffBaseNode: metadataDiffBaseNode{
					action: diffActionCreate,
				},
				name: column.Name,
				to:   column,
			}
			continue
		}
		diffNodeMeta, err := diffColumnMetadata(columnInFrom, column)
		if err != nil {
			return nil, errors.Wrapf(err, "failed to diff column %q", column.Name)
		}
		if diffNodeMeta != nil {
			tableNode.columns[column.Name] = diffNodeMeta
		}
		delete(fromColumnMap, column.Name)
	}

	for _, foreignKey := range to.ForeignKeys {
		foreignKeyInFrom, in := fromForeignKeyMap[foreignKey.Name]
		if !in {
			tableNode.foreignKeys[foreignKey.Name] = &metadataDiffForeignKeyNode{
				metadataDiffBaseNode: metadataDiffBaseNode{
					action: diffActionCreate,
				},
				name: foreignKey.Name,
				to:   foreignKey,
			}
			continue
		}
		diffNodeMeta, err := diffForeignKeyMetadata(foreignKeyInFrom, foreignKey)
		if err != nil {
			return nil, errors.Wrapf(err, "failed to diff foreign key %q", foreignKey.Name)
		}
		if diffNodeMeta != nil {
			tableNode.foreignKeys[foreignKey.Name] = diffNodeMeta
		}
		delete(fromForeignKeyMap, foreignKey.Name)
	}

	for _, remainingColumn := range fromColumnMap {
		tableNode.columns[remainingColumn.Name] = &metadataDiffColumnNode{
			metadataDiffBaseNode: metadataDiffBaseNode{
				action: diffActionDrop,
			},
			name: remainingColumn.Name,
			from: remainingColumn,
		}
	}

	for _, remainingForeignKey := range fromForeignKeyMap {
		tableNode.foreignKeys[remainingForeignKey.Name] = &metadataDiffForeignKeyNode{
			metadataDiffBaseNode: metadataDiffBaseNode{
				action: diffActionDrop,
			},
			name: remainingForeignKey.Name,
			from: remainingForeignKey,
		}
	}

	if len(tableNode.columns) > 0 && len(tableNode.foreignKeys) > 0 {
		tableNode.action = diffActionUpdate
		return tableNode, nil
	}
	return nil, nil
}

func diffColumnMetadata(from, to *v1pb.ColumnMetadata) (*metadataDiffColumnNode, error) {
	if from == nil || to == nil {
		return nil, errors.New("from and to column metadata must not be nil")
	}

	if !reflect.DeepEqual(from, to) {
		return &metadataDiffColumnNode{
			metadataDiffBaseNode: metadataDiffBaseNode{
				action: diffActionUpdate,
			},
			name: to.Name,
			from: from,
		}, nil
	}
	return nil, nil
}

func diffForeignKeyMetadata(from, to *v1pb.ForeignKeyMetadata) (*metadataDiffForeignKeyNode, error) {
	if from == nil || to == nil {
		return nil, errors.New("from and to foreign key metadata must not be nil")
	}

	if !reflect.DeepEqual(from, to) {
		return &metadataDiffForeignKeyNode{
			metadataDiffBaseNode: metadataDiffBaseNode{
				action: diffActionUpdate,
			},
			name: to.Name,
			from: from,
		}, nil
	}
	return nil, nil
}
