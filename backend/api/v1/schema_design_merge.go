package v1

import (
	"fmt"
	"reflect"
	"slices"

	"github.com/google/go-cmp/cmp"
	"github.com/pkg/errors"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/testing/protocmp"

	storepb "github.com/bytebase/bytebase/proto/generated-go/store"
)

type diffAction string

const (
	diffActionCreate diffAction = "CREATE"
	diffActionUpdate diffAction = "UPDATE"
	diffActionDrop   diffAction = "DROP"
)

func tryMerge(ancestor, head, base *storepb.DatabaseSchemaMetadata) (*storepb.DatabaseSchemaMetadata, error) {
	ancestor, head, base = proto.Clone(ancestor).(*storepb.DatabaseSchemaMetadata), proto.Clone(head).(*storepb.DatabaseSchemaMetadata), proto.Clone(base).(*storepb.DatabaseSchemaMetadata)

	diffBetweenAncestorAndHead, err := diffMetadata(ancestor, head)
	if err != nil {
		return nil, errors.Wrap(err, "failed to diff between ancestor and head")
	}

	diffBetweenAncestorAndBase, err := diffMetadata(ancestor, base)
	if err != nil {
		return nil, errors.Wrap(err, "failed to diff between ancestor and base")
	}

	if conflict, msg := diffBetweenAncestorAndHead.tryMerge(diffBetweenAncestorAndBase); conflict {
		return nil, errors.Errorf("merge conflict: %s", msg)
	}

	if err := diffBetweenAncestorAndHead.applyDiffTo(ancestor); err != nil {
		return nil, errors.Wrap(err, "failed to apply diff to target")
	}

	return ancestor, nil
}

type metadataDiffBaseNode struct {
	action diffAction
}

type metadataDiffRootNode struct {
	schemas map[string]*metadataDiffSchemaNode
}

// tryMerge merges other root node to current root node, stop and return error if conflict occurs.
func (mr *metadataDiffRootNode) tryMerge(other *metadataDiffRootNode) (bool, string) {
	for _, schema := range mr.schemas {
		otherSchema, in := other.schemas[schema.name]
		if !in {
			continue
		}
		conflict, msg := schema.tryMerge(otherSchema)
		if conflict {
			return true, msg
		}
		delete(other.schemas, schema.name)
	}
	// Append other schema to current root node.
	for _, otherSchema := range other.schemas {
		mr.schemas[otherSchema.name] = otherSchema
	}
	return false, ""
}

func (mr *metadataDiffRootNode) applyDiffTo(target *storepb.DatabaseSchemaMetadata) error {
	sortedSchemaNames := make([]string, 0, len(mr.schemas))
	for schemaName := range mr.schemas {
		sortedSchemaNames = append(sortedSchemaNames, schemaName)
	}
	slices.Sort(sortedSchemaNames)
	for _, schemaName := range sortedSchemaNames {
		schema := mr.schemas[schemaName]
		if err := schema.applyDiffTo(target); err != nil {
			return errors.Wrapf(err, "failed to apply diff to schema %q", schema.name)
		}
	}
	return nil
}

// Schema related.
type metadataDiffSchemaNode struct {
	metadataDiffBaseNode
	name string
	//nolint
	base *storepb.SchemaMetadata
	head *storepb.SchemaMetadata

	tables map[string]*metadataDiffTableNode

	// SchemaMetadata contains other object types, likes function, view and etc, but we do not support them yet.
}

func (n *metadataDiffSchemaNode) tryMerge(other *metadataDiffSchemaNode) (bool, string) {
	if other == nil {
		return true, "other node check conflict with schema node not be nil"
	}

	if n.name != other.name {
		return true, fmt.Sprintf("non-expected schema node pair, one is %s, the other is %s", n.name, other.name)
	}
	if n.action != other.action {
		return true, fmt.Sprintf("conflict schema action, one is %s, the other is %s", n.action, other.action)
	}

	if n.action == diffActionDrop {
		return false, ""
	}

	// if n.action == diffActionCreate {
	// 	// If two actions are CREATE both, we need to check the schema attributes is conflict.
	// 	// XXX: Expanding the schema attributes check if we support more attributes.
	// }

	// if n.action == diffActionUpdate {
	// 	// If two actions are UPDATE both, we need to check the schema attributes is conflict.
	// 	// XXX: Expanding the schema attributes check if we support more attributes.
	// }

	for tableName, tableNode := range n.tables {
		otherTableNode, in := other.tables[tableName]
		if !in {
			continue
		}
		conflict, msg := tableNode.tryMerge(otherTableNode)
		if conflict {
			return true, msg
		}
		delete(other.tables, tableName)
	}

	for _, remainingTable := range other.tables {
		n.tables[remainingTable.name] = remainingTable
	}

	return false, ""
}

func (n *metadataDiffSchemaNode) applyDiffTo(target *storepb.DatabaseSchemaMetadata) error {
	if target == nil {
		return errors.New("target must not be nil")
	}

	sortedTableNames := make([]string, 0, len(n.tables))
	for tableName := range n.tables {
		sortedTableNames = append(sortedTableNames, tableName)
	}
	slices.Sort(sortedTableNames)

	switch n.action {
	case diffActionCreate:
		newSchema := &storepb.SchemaMetadata{
			Name: n.name,
		}
		for _, tableName := range sortedTableNames {
			table := n.tables[tableName]
			if err := table.applyDiffTo(newSchema); err != nil {
				return errors.Wrapf(err, "failed to apply diff to table %q", table.name)
			}
		}
		target.Schemas = append(target.Schemas, newSchema)
	case diffActionDrop:
		for i, schema := range target.Schemas {
			if schema.Name == n.name {
				target.Schemas = append(target.Schemas[:i], target.Schemas[i+1:]...)
				break
			}
		}
	case diffActionUpdate:

		for idx, schema := range target.Schemas {
			if schema.Name == n.name {
				newSchema := &storepb.SchemaMetadata{
					Name:   n.name,
					Tables: schema.Tables,
				}
				// Update schema currently is only contains diff of tables. So we do apply table diff to target schema.
				for _, tableName := range sortedTableNames {
					table := n.tables[tableName]
					if err := table.applyDiffTo(newSchema); err != nil {
						return errors.Wrapf(err, "failed to apply diff to table %q", table.name)
					}
				}
				target.Schemas[idx] = newSchema
			}
		}
	}
	return nil
}

// Table related.
type metadataDiffTableNode struct {
	metadataDiffBaseNode
	name string
	//nolint
	base *storepb.TableMetadata
	head *storepb.TableMetadata

	columns     map[string]*metadataDiffColumnNode
	foreignKeys map[string]*metadataDiffForeignKeyNode
	indexes     map[string]*metadataDiffIndexNode
	// TableMetaData contains other object types, likes trigger, index and etc, but we do not support them yet.
}

func (n *metadataDiffTableNode) tryMerge(other *metadataDiffTableNode) (bool, string) {
	if other == nil {
		return true, "other node check conflict with table node must not be nil"
	}

	if n.name != other.name {
		return true, fmt.Sprintf("non-expected table node pair, one is %s, the other is %s", n.name, other.name)
	}
	if n.action != other.action {
		return true, fmt.Sprintf("conflict table action, one is %s, the other is %s", n.action, other.action)
	}

	if n.action == diffActionDrop {
		return false, ""
	}

	if n.action == diffActionCreate {
		// If two actions are CREATE or UPDATE both, we need to check the table attributes is conflict.
		// XXX: Expanding the table attributes check if we support more attributes.
		if n.head.Engine != other.head.Engine {
			return true, fmt.Sprintf("conflict table engine, one is %s, the other is %s", n.head.Engine, other.head.Engine)
		}
		if n.head.Collation != other.head.Collation {
			return true, fmt.Sprintf("conflict table collation, one is %s, the other is %s", n.head.Collation, other.head.Collation)
		}
		if n.head.Comment != other.head.Comment {
			return true, fmt.Sprintf("conflict table comment, one is %s, the other is %s", n.head.Comment, other.head.Comment)
		}
		if n.head.UserComment != other.head.UserComment {
			return true, fmt.Sprintf("conflict table user comment, one is %s, the other is %s", n.head.UserComment, other.head.UserComment)
		}
		if n.head.Classification != other.head.Classification {
			return true, fmt.Sprintf("conflict table classification, one is %s, the other is %s", n.head.Classification, other.head.Classification)
		}
	}

	if n.action == diffActionUpdate {
		if other.base.Engine != other.head.Engine {
			if n.base.Engine != n.head.Engine {
				if n.head.Engine != other.head.Engine {
					return true, fmt.Sprintf("conflict table engine, one is %s, the other is %s", n.base.Engine, other.base.Engine)
				}
			} else {
				n.head.Engine = other.head.Engine
			}
		}

		if other.base.Collation != other.head.Collation {
			if n.base.Collation != n.head.Collation {
				if n.head.Collation != other.head.Collation {
					return true, fmt.Sprintf("conflict table collation, one is %s, the other is %s", n.base.Collation, other.base.Collation)
				}
			} else {
				n.head.Collation = other.head.Collation
			}
		}

		if other.base.Comment != other.head.Comment {
			if n.base.Comment != n.head.Comment {
				if n.head.Comment != other.head.Comment {
					return true, fmt.Sprintf("conflict table comment, one is %s, the other is %s", n.base.Comment, other.base.Comment)
				}
			} else {
				n.head.Comment = other.head.Comment
			}
		}

		if other.base.UserComment != other.head.UserComment {
			if n.base.UserComment != n.head.UserComment {
				if n.head.UserComment != other.head.UserComment {
					return true, fmt.Sprintf("conflict table user comment, one is %s, the other is %s", n.base.UserComment, other.base.UserComment)
				}
			} else {
				n.head.UserComment = other.head.UserComment
			}
		}

		if other.base.Classification != other.head.Classification {
			if n.base.Classification != n.head.Classification {
				if n.head.Classification != other.head.Classification {
					return true, fmt.Sprintf("conflict table classification, one is %s, the other is %s", n.base.Classification, other.base.Classification)
				}
			} else {
				n.head.Classification = other.head.Classification
			}
		}
	}

	for columnName, columnNode := range n.columns {
		otherColumnNode, in := other.columns[columnName]
		if !in {
			continue
		}
		conflict, msg := columnNode.tryMerge(otherColumnNode)
		if conflict {
			return true, msg
		}
		delete(other.columns, columnName)
	}

	for foreignKeyName, foreignKeyNode := range n.foreignKeys {
		otherForeignKeyNode, in := other.foreignKeys[foreignKeyName]
		if !in {
			continue
		}
		conflict, msg := foreignKeyNode.tryMerge(otherForeignKeyNode)
		if conflict {
			return true, msg
		}
		delete(other.foreignKeys, foreignKeyName)
	}

	for indexName, indexNode := range n.indexes {
		otherIndexNode, in := other.indexes[indexName]
		if !in {
			continue
		}
		conflict, msg := indexNode.tryMerge(otherIndexNode)
		if conflict {
			return true, msg
		}
		delete(other.indexes, indexName)
	}

	for _, remainingColumn := range other.columns {
		n.columns[remainingColumn.name] = remainingColumn
	}

	for _, remainingForeignKey := range other.foreignKeys {
		n.foreignKeys[remainingForeignKey.name] = remainingForeignKey
	}

	for _, remainingIndex := range other.indexes {
		n.indexes[remainingIndex.name] = remainingIndex
	}

	return false, ""
}

func (n *metadataDiffTableNode) applyDiffTo(target *storepb.SchemaMetadata) error {
	if target == nil {
		return errors.New("target must not be nil")
	}

	sortedColumnName := make([]string, 0, len(n.columns))
	for columnName := range n.columns {
		sortedColumnName = append(sortedColumnName, columnName)
	}
	slices.Sort(sortedColumnName)

	sortedForeignKeyName := make([]string, 0, len(n.foreignKeys))
	for foreignKeyName := range n.foreignKeys {
		sortedForeignKeyName = append(sortedForeignKeyName, foreignKeyName)
	}
	slices.Sort(sortedForeignKeyName)

	sortedIndexName := make([]string, 0, len(n.indexes))
	for indexName := range n.indexes {
		sortedIndexName = append(sortedIndexName, indexName)
	}
	slices.Sort(sortedIndexName)

	switch n.action {
	case diffActionCreate:
		columnsInHead := make(map[string]bool)
		for _, column := range n.head.Columns {
			columnsInHead[column.Name] = true
		}
		newTable := &storepb.TableMetadata{
			Name:           n.name,
			Engine:         n.head.Engine,
			Collation:      n.head.Collation,
			Comment:        n.head.Comment,
			UserComment:    n.head.UserComment,
			Classification: n.head.Classification,
		}
		for columnName := range columnsInHead {
			if columnNode, in := n.columns[columnName]; in {
				if err := columnNode.applyDiffTo(newTable); err != nil {
					return errors.Wrapf(err, "failed to apply diff to column %q", columnNode.name)
				}
			}
		}

		for _, columnName := range sortedColumnName {
			if _, in := columnsInHead[columnName]; !in {
				column := n.columns[columnName]
				if err := column.applyDiffTo(newTable); err != nil {
					return errors.Wrapf(err, "failed to apply diff to column %q", column.name)
				}
			}
		}

		// XXX(zp): We need to find a better way to solve the problem of column position.
		// We need to sort the columns by position after applying the diff.
		for idx := range newTable.Columns {
			newTable.Columns[idx].Position = int32(idx + 1)
		}

		for _, foreignKeyName := range sortedForeignKeyName {
			foreignKey := n.foreignKeys[foreignKeyName]
			if err := foreignKey.applyDiffTo(newTable); err != nil {
				return errors.Wrapf(err, "failed to apply diff to foreign key %q", foreignKey.name)
			}
		}
		for _, indexName := range sortedIndexName {
			index := n.indexes[indexName]
			if err := index.applyDiffTo(n); err != nil {
				return errors.Wrapf(err, "failed to apply diff to index %q", index.name)
			}
		}
		target.Tables = append(target.Tables, newTable)
	case diffActionDrop:
		for i, table := range target.Tables {
			if table.Name == n.name {
				target.Tables = append(target.Tables[:i], target.Tables[i+1:]...)
				break
			}
		}
	case diffActionUpdate:
		for idx, table := range target.Tables {
			// Update table currently is only contains diff of columns and foreign keys. So we do apply column and foreign key diff to target table.
			if table.Name == n.name {
				newTable := &storepb.TableMetadata{
					Name:           n.name,
					Engine:         n.head.Engine,
					Collation:      n.head.Collation,
					Comment:        n.head.Comment,
					UserComment:    n.head.UserComment,
					Classification: n.head.Classification,
					Columns:        table.Columns,
					ForeignKeys:    table.ForeignKeys,
					Indexes:        table.Indexes,
				}
				for _, columnName := range sortedColumnName {
					column := n.columns[columnName]
					if err := column.applyDiffTo(newTable); err != nil {
						return errors.Wrapf(err, "failed to apply diff to column %q", column.name)
					}
				}
				for _, foreignKeyName := range sortedForeignKeyName {
					foreignKey := n.foreignKeys[foreignKeyName]
					if err := foreignKey.applyDiffTo(newTable); err != nil {
						return errors.Wrapf(err, "failed to apply diff to foreign key %q", foreignKey.name)
					}
				}
				for _, indexName := range sortedIndexName {
					index := n.indexes[indexName]
					if err := index.applyDiffTo(n); err != nil {
						return errors.Wrapf(err, "failed to apply diff to index %q", index.name)
					}
				}
				// XXX(zp): We need to find a better way to solve the problem of column position.
				// We need to sort the columns by position after applying the diff.
				for idx := range newTable.Columns {
					newTable.Columns[idx].Position = int32(idx + 1)
				}
				target.Tables[idx] = newTable
			}
		}
	}
	return nil
}

// Column related.
type metadataDiffColumnNode struct {
	metadataDiffBaseNode
	name string
	//nolint
	base *storepb.ColumnMetadata
	head *storepb.ColumnMetadata
}

func (n *metadataDiffColumnNode) tryMerge(other *metadataDiffColumnNode) (bool, string) {
	if other == nil {
		return true, "other node check conflict with column node must not be nil"
	}

	if n.name != other.name {
		return true, fmt.Sprintf("non-expected column node pair, one is %s, the other is %s", n.name, other.name)
	}
	if n.action != other.action {
		return true, fmt.Sprintf("conflict column action, one is %s, the other is %s", n.action, other.action)
	}

	if n.action == diffActionDrop {
		return false, ""
	}

	if n.action == diffActionCreate {
		if n.head.Type != other.head.Type {
			return true, fmt.Sprintf("conflict column type, one is %s, the other is %s", n.head.Type, other.head.Type)
		}
		if n.head.DefaultValue != other.head.DefaultValue {
			return true, fmt.Sprintf("conflict column default value, one is %s, the other is %s", n.head.DefaultValue, other.head.DefaultValue)
		}
		if n.head.Nullable != other.head.Nullable {
			return true, fmt.Sprintf("conflict column nullable, one is %t, the other is %t", n.head.Nullable, other.head.Nullable)
		}
		if n.head.Comment != other.head.Comment {
			return true, fmt.Sprintf("conflict column comment, one is %s, the other is %s", n.head.Comment, other.head.Comment)
		}
		if n.head.UserComment != other.head.UserComment {
			return true, fmt.Sprintf("conflict column user comment, one is %s, the other is %s", n.head.UserComment, other.head.UserComment)
		}
		if n.head.Classification != other.head.Classification {
			return true, fmt.Sprintf("conflict column classification, one is %s, the other is %s", n.head.Classification, other.head.Classification)
		}
	}
	if n.action == diffActionUpdate {
		if other.base.Type != other.head.Type {
			if n.base.Type != n.head.Type {
				if n.head.Type != other.head.Type {
					return true, fmt.Sprintf("conflict column type, one is %s, the other is %s", n.base.Type, other.base.Type)
				}
			} else {
				n.head.Type = other.head.Type
			}
		}

		if otherDiff := cmp.Diff(other.base.DefaultValue, other.head.DefaultValue, protocmp.Transform()); otherDiff != "" {
			if nDiff := cmp.Diff(n.base.DefaultValue, n.head.DefaultValue, protocmp.Transform()); nDiff != "" {
				if d := cmp.Diff(n.head.DefaultValue, other.head.DefaultValue, protocmp.Transform()); d != "" {
					return true, fmt.Sprintf("conflict column default value, one is %v, the other is %v", n.base.DefaultValue, other.base.DefaultValue)
				}
			} else {
				n.head.DefaultValue = other.head.DefaultValue
			}
		}

		if other.base.Nullable != other.head.Nullable {
			if n.base.Nullable != n.head.Nullable {
				if n.head.Nullable != other.head.Nullable {
					return true, fmt.Sprintf("conflict column nullable, one is %t, the other is %t", n.base.Nullable, other.base.Nullable)
				}
			} else {
				n.head.Nullable = other.head.Nullable
			}
		}

		if other.base.Comment != other.head.Comment {
			if n.base.Comment != n.head.Comment {
				if n.head.Comment != other.head.Comment {
					return true, fmt.Sprintf("conflict column comment, one is %s, the other is %s", n.base.Comment, other.base.Comment)
				}
			} else {
				n.head.Comment = other.head.Comment
			}
		}

		if other.base.UserComment != other.head.UserComment {
			if n.base.UserComment != n.head.UserComment {
				if n.head.UserComment != other.head.UserComment {
					return true, fmt.Sprintf("conflict column user comment, one is %s, the other is %s", n.base.UserComment, other.base.UserComment)
				}
			} else {
				n.head.UserComment = other.head.UserComment
			}
		}

		if other.base.Classification != other.head.Classification {
			if n.base.Classification != n.head.Classification {
				if n.head.Classification != other.head.Classification {
					return true, fmt.Sprintf("conflict column classification, one is %s, the other is %s", n.base.Classification, other.base.Classification)
				}
			} else {
				n.head.Classification = other.head.Classification
			}
		}
	}

	return false, ""
}

func (n *metadataDiffColumnNode) applyDiffTo(target *storepb.TableMetadata) error {
	// TODO(zp): handle the column position...
	switch n.action {
	case diffActionCreate:
		target.Columns = append(target.Columns, n.head)
	case diffActionDrop:
		for i, column := range target.Columns {
			if column.Name == n.name {
				target.Columns = append(target.Columns[:i], target.Columns[i+1:]...)
				break
			}
		}
	case diffActionUpdate:
		for i, column := range target.Columns {
			if column.Name == n.name {
				target.Columns[i] = n.head
				break
			}
		}
	}
	return nil
}

// Index related.
type metadataDiffIndexNode struct {
	metadataDiffBaseNode
	name string
	//nolint
	base *storepb.IndexMetadata
	head *storepb.IndexMetadata
}

func (n *metadataDiffIndexNode) tryMerge(other *metadataDiffIndexNode) (bool, string) {
	if other == nil {
		return true, "other node check conflict with column node must not be nil"
	}

	if n.name != other.name {
		return true, fmt.Sprintf("non-expected index node pair, one is %s, the other is %s", n.name, other.name)
	}
	if n.action != other.action {
		return true, fmt.Sprintf("conflict index action, one is %s, the other is %s", n.action, other.action)
	}
	if n.action == diffActionDrop {
		return false, ""
	}
	if n.action == diffActionCreate {
		if slices.Equal(n.head.Expressions, other.head.Expressions) {
			return true, fmt.Sprintf("conflict index expressions, one is %v, the other is %v", n.head.Expressions, other.head.Expressions)
		}
		if n.head.Type != other.head.Type {
			return true, fmt.Sprintf("conflict index type, one is %s, the other is %s", n.head.Type, other.head.Type)
		}
		if n.head.Unique != other.head.Unique {
			return true, fmt.Sprintf("conflict index unique, one is %t, the other is %t", n.head.Unique, other.head.Unique)
		}
		if n.head.Primary != other.head.Primary {
			return true, fmt.Sprintf("conflict index primary, one is %t, the other is %t", n.head.Primary, other.head.Primary)
		}
	}

	if n.action == diffActionUpdate {
		if !slices.Equal(other.base.Expressions, other.head.Expressions) {
			if !slices.Equal(n.base.Expressions, n.head.Expressions) {
				if !slices.Equal(n.head.Expressions, other.head.Expressions) {
					return true, fmt.Sprintf("conflict index expressions, one is %v, the other is %v", n.base.Expressions, other.base.Expressions)
				}
			} else {
				n.head.Expressions = other.head.Expressions
			}
		}

		if other.base.Type != other.head.Type {
			if n.base.Type != n.head.Type {
				if n.head.Type != other.head.Type {
					return true, fmt.Sprintf("conflict index type, one is %s, the other is %s", n.base.Type, other.base.Type)
				}
			} else {
				n.head.Type = other.head.Type
			}
		}

		if other.base.Unique != other.head.Unique {
			if n.base.Unique != n.head.Unique {
				if n.head.Unique != other.head.Unique {
					return true, fmt.Sprintf("conflict index unique, one is %t, the other is %t", n.base.Unique, other.base.Unique)
				}
			} else {
				n.head.Unique = other.head.Unique
			}
		}

		if other.base.Primary != other.head.Primary {
			if n.base.Primary != n.head.Primary {
				if n.head.Primary != other.head.Primary {
					return true, fmt.Sprintf("conflict index primary, one is %t, the other is %t", n.base.Primary, other.base.Primary)
				}
			} else {
				n.head.Primary = other.head.Primary
			}
		}
	}

	return false, ""
}

func (n *metadataDiffIndexNode) applyDiffTo(target *metadataDiffTableNode) error {
	switch n.action {
	case diffActionCreate:
		target.head.Indexes = append(target.head.Indexes, n.head)
	case diffActionDrop:
		for i, index := range target.head.Indexes {
			if index.Name == n.name {
				target.head.Indexes = append(target.head.Indexes[:i], target.head.Indexes[i+1:]...)
				break
			}
		}
	case diffActionUpdate:
		for i, index := range target.head.Indexes {
			if index.Name == n.name {
				target.head.Indexes[i] = n.head
				break
			}
		}
	}
	return nil
}

// Foreign Key related.
type metadataDiffForeignKeyNode struct {
	metadataDiffBaseNode
	name string
	//nolint
	base *storepb.ForeignKeyMetadata
	head *storepb.ForeignKeyMetadata
}

func (n *metadataDiffForeignKeyNode) tryMerge(other *metadataDiffForeignKeyNode) (bool, string) {
	if other == nil {
		return true, "other node check conflict with column node must not be nil"
	}

	if n.name != other.name {
		return true, fmt.Sprintf("non-expected foreign key node pair, one is %s, the other is %s", n.name, other.name)
	}
	if n.action != other.action {
		return true, fmt.Sprintf("conflict foreign key action, one is %s, the other is %s", n.action, other.action)
	}
	if n.action == diffActionDrop {
		return false, ""
	}
	if n.action == diffActionCreate {
		if n.head.ReferencedSchema != other.head.ReferencedSchema {
			return true, fmt.Sprintf("conflict foreign key referenced schema, one is %s, the other is %s", n.head.ReferencedSchema, other.head.ReferencedSchema)
		}
		if n.head.ReferencedTable != other.head.ReferencedTable {
			return true, fmt.Sprintf("conflict foreign key referenced table, one is %s, the other is %s", n.head.ReferencedTable, other.head.ReferencedTable)
		}
		if n.head.OnDelete != other.head.OnDelete {
			return true, fmt.Sprintf("conflict foreign key on delete, one is %s, the other is %s", n.head.OnDelete, other.head.OnDelete)
		}
		if n.head.OnUpdate != other.head.OnUpdate {
			return true, fmt.Sprintf("conflict foreign key on update, one is %s, the other is %s", n.head.OnUpdate, other.head.OnUpdate)
		}
		if !slices.Equal(n.head.Columns, other.head.Columns) {
			return true, fmt.Sprintf("conflict foreign key columns, one is %v, the other is %v", n.head.Columns, other.head.Columns)
		}
		if !slices.Equal(n.head.ReferencedColumns, other.head.ReferencedColumns) {
			return true, fmt.Sprintf("conflict foreign key referenced columns, one is %v, the other is %v", n.head.ReferencedColumns, other.head.ReferencedColumns)
		}
	}
	if n.action == diffActionUpdate {
		if other.base.ReferencedSchema != other.head.ReferencedSchema {
			if n.base.ReferencedSchema != n.head.ReferencedSchema {
				if n.head.ReferencedSchema != other.head.ReferencedSchema {
					return true, fmt.Sprintf("conflict foreign key referenced schema, one is %s, the other is %s", n.base.ReferencedSchema, other.base.ReferencedSchema)
				}
			} else {
				n.head.ReferencedSchema = other.head.ReferencedSchema
			}
		}

		if other.base.ReferencedTable != other.head.ReferencedTable {
			if n.base.ReferencedTable != n.head.ReferencedTable {
				if n.head.ReferencedTable != other.head.ReferencedTable {
					return true, fmt.Sprintf("conflict foreign key referenced table, one is %s, the other is %s", n.base.ReferencedTable, other.base.ReferencedTable)
				}
			} else {
				n.head.ReferencedTable = other.head.ReferencedTable
			}
		}

		if other.base.OnDelete != other.head.OnDelete {
			if n.base.OnDelete != n.head.OnDelete {
				if n.head.OnDelete != other.head.OnDelete {
					return true, fmt.Sprintf("conflict foreign key on delete, one is %s, the other is %s", n.base.OnDelete, other.base.OnDelete)
				}
			} else {
				n.head.OnDelete = other.head.OnDelete
			}
		}

		if other.base.OnUpdate != other.head.OnUpdate {
			if n.base.OnUpdate != n.head.OnUpdate {
				if n.head.OnUpdate != other.head.OnUpdate {
					return true, fmt.Sprintf("conflict foreign key on update, one is %s, the other is %s", n.base.OnUpdate, other.base.OnUpdate)
				}
			} else {
				n.head.OnUpdate = other.head.OnUpdate
			}
		}

		if !reflect.DeepEqual(other.base.Columns, other.head.Columns) {
			if !reflect.DeepEqual(n.base.Columns, n.head.Columns) {
				if !reflect.DeepEqual(n.head.Columns, other.head.Columns) {
					return true, fmt.Sprintf("conflict foreign key columns, one is %v, the other is %v", n.base.Columns, other.base.Columns)
				}
			} else {
				n.head.Columns = other.head.Columns
			}
		}
	}

	return false, ""
}

func (n *metadataDiffForeignKeyNode) applyDiffTo(target *storepb.TableMetadata) error {
	switch n.action {
	case diffActionCreate:
		target.ForeignKeys = append(target.ForeignKeys, n.head)
	case diffActionDrop:
		for i, foreignKey := range target.ForeignKeys {
			if foreignKey.Name == n.name {
				target.ForeignKeys = append(target.ForeignKeys[:i], target.ForeignKeys[i+1:]...)
				break
			}
		}
	case diffActionUpdate:
		for i, foreignKey := range target.ForeignKeys {
			if foreignKey.Name == n.name {
				target.ForeignKeys[i] = n.head
				break
			}
		}
	}
	return nil
}

// diffMetadata returns the diff between base and head metadata, it always returns a non-nil root node if no error occurs.
func diffMetadata(base, head *storepb.DatabaseSchemaMetadata) (*metadataDiffRootNode, error) {
	if base == nil && head == nil {
		return nil, errors.New("from and to metadata must not be nil")
	}

	root := &metadataDiffRootNode{
		schemas: make(map[string]*metadataDiffSchemaNode),
	}

	schemaNamesMap := make(map[string]bool)
	baseSchemaMap := make(map[string]*storepb.SchemaMetadata)
	if base != nil {
		for _, schema := range base.Schemas {
			baseSchemaMap[schema.Name] = schema
			schemaNamesMap[schema.Name] = true
		}
	}

	headSchemaMap := make(map[string]*storepb.SchemaMetadata)
	if head != nil {
		for _, schema := range head.Schemas {
			headSchemaMap[schema.Name] = schema
			schemaNamesMap[schema.Name] = true
		}
	}

	for schemaName := range schemaNamesMap {
		baseSchema, headSchema := baseSchemaMap[schemaName], headSchemaMap[schemaName]
		diffNode, err := diffSchemaMetadata(baseSchema, headSchema)
		if err != nil {
			return nil, errors.Wrapf(err, "failed to diff schema %q", schemaName)
		}
		if diffNode != nil {
			root.schemas[schemaName] = diffNode
		}
	}

	return root, nil
}

func diffSchemaMetadata(base, head *storepb.SchemaMetadata) (*metadataDiffSchemaNode, error) {
	if base == nil && head == nil {
		return nil, nil
	}
	var name string
	action := diffActionUpdate
	if base == nil {
		action = diffActionCreate
		name = head.Name
	} else if head == nil {
		action = diffActionDrop
		name = base.Name
	} else {
		name = base.Name
	}

	schemaNode := &metadataDiffSchemaNode{
		metadataDiffBaseNode: metadataDiffBaseNode{
			action: action,
		},
		name:   name,
		base:   base,
		head:   head,
		tables: make(map[string]*metadataDiffTableNode),
	}

	tableNamesMap := make(map[string]bool)

	baseTableMap := make(map[string]*storepb.TableMetadata)
	if base != nil {
		for _, table := range base.Tables {
			baseTableMap[table.Name] = table
			tableNamesMap[table.Name] = true
		}
	}

	headTableMap := make(map[string]*storepb.TableMetadata)
	if head != nil {
		for _, table := range head.Tables {
			headTableMap[table.Name] = table
			tableNamesMap[table.Name] = true
		}
	}

	for tableName := range tableNamesMap {
		baseTable, headTable := baseTableMap[tableName], headTableMap[tableName]
		diffNode, err := diffTableMetadata(baseTable, headTable)
		if err != nil {
			return nil, errors.Wrapf(err, "failed to diff table %q", tableName)
		}
		if diffNode != nil {
			schemaNode.tables[tableName] = diffNode
		}
	}

	if action == diffActionUpdate {
		if len(schemaNode.tables) == 0 {
			return nil, nil
		}
	}

	return schemaNode, nil
}

func diffTableMetadata(base, head *storepb.TableMetadata) (*metadataDiffTableNode, error) {
	if base == nil && head == nil {
		return nil, errors.New("from and to table metadata must not be nil")
	}

	var name string
	action := diffActionUpdate
	if base == nil {
		action = diffActionCreate
		name = head.Name
	} else if head == nil {
		action = diffActionDrop
		name = base.Name
	} else {
		name = base.Name
	}

	tableNode := &metadataDiffTableNode{
		metadataDiffBaseNode: metadataDiffBaseNode{
			action: action,
		},
		name:        name,
		base:        base,
		head:        head,
		columns:     make(map[string]*metadataDiffColumnNode),
		foreignKeys: make(map[string]*metadataDiffForeignKeyNode),
		indexes:     make(map[string]*metadataDiffIndexNode),
	}

	columnNamesMap := make(map[string]bool)

	baseColumnMap := make(map[string]*storepb.ColumnMetadata)
	if base != nil {
		for _, column := range base.Columns {
			baseColumnMap[column.Name] = column
			columnNamesMap[column.Name] = true
		}
	}

	headColumnMap := make(map[string]*storepb.ColumnMetadata)
	if head != nil {
		for _, column := range head.Columns {
			headColumnMap[column.Name] = column
			columnNamesMap[column.Name] = true
		}
	}

	for columnName := range columnNamesMap {
		baseColumn, headColumn := baseColumnMap[columnName], headColumnMap[columnName]
		diffNode, err := diffColumnMetadata(baseColumn, headColumn)
		if err != nil {
			return nil, errors.Wrapf(err, "failed to diff column %q", columnName)
		}
		if diffNode != nil {
			tableNode.columns[columnName] = diffNode
		}
	}

	foreignKeyNamesMap := make(map[string]bool)

	baseForeignKeyMap := make(map[string]*storepb.ForeignKeyMetadata)
	if base != nil {
		for _, foreignKey := range base.ForeignKeys {
			baseForeignKeyMap[foreignKey.Name] = foreignKey
			foreignKeyNamesMap[foreignKey.Name] = true
		}
	}

	headForeignKeyMap := make(map[string]*storepb.ForeignKeyMetadata)
	if head != nil {
		for _, foreignKey := range head.ForeignKeys {
			headForeignKeyMap[foreignKey.Name] = foreignKey
			foreignKeyNamesMap[foreignKey.Name] = true
		}
	}

	for foreignKeyName := range foreignKeyNamesMap {
		baseForeignKey, headForeignKey := baseForeignKeyMap[foreignKeyName], headForeignKeyMap[foreignKeyName]
		diffNode, err := diffForeignKeyMetadata(baseForeignKey, headForeignKey)
		if err != nil {
			return nil, errors.Wrapf(err, "failed to diff foreign key %q", foreignKeyName)
		}
		if diffNode != nil {
			tableNode.foreignKeys[foreignKeyName] = diffNode
		}
	}

	if action == diffActionUpdate {
		if len(tableNode.columns) == 0 && len(tableNode.foreignKeys) == 0 {
			return nil, nil
		}
	}

	return tableNode, nil
}

func diffColumnMetadata(base, head *storepb.ColumnMetadata) (*metadataDiffColumnNode, error) {
	if base == nil && head == nil {
		return nil, errors.New("base and head column metadata cannot be nil both")
	}

	var name string
	action := diffActionUpdate
	if base == nil {
		action = diffActionCreate
		name = head.Name
	} else if head == nil {
		action = diffActionDrop
		name = base.Name
	} else {
		name = base.Name
	}

	columnNode := &metadataDiffColumnNode{
		metadataDiffBaseNode: metadataDiffBaseNode{
			action: action,
		},
		name: name,
		base: base,
		head: head,
	}

	if action == diffActionUpdate {
		if !reflect.DeepEqual(base, head) {
			return columnNode, nil
		}
		return nil, nil
	}
	return columnNode, nil
}

func diffForeignKeyMetadata(base, head *storepb.ForeignKeyMetadata) (*metadataDiffForeignKeyNode, error) {
	if base == nil && head == nil {
		return nil, errors.New("base and head foreign key metadata cannot be nil both")
	}

	var name string
	action := diffActionUpdate
	if base == nil {
		action = diffActionCreate
		name = head.Name
	} else if head == nil {
		action = diffActionDrop
		name = base.Name
	} else {
		name = base.Name
	}

	fkNode := &metadataDiffForeignKeyNode{
		metadataDiffBaseNode: metadataDiffBaseNode{
			action: action,
		},
		name: name,
		base: base,
		head: head,
	}

	if action == diffActionUpdate {
		if !reflect.DeepEqual(base, head) {
			return fkNode, nil
		}
		return nil, nil
	}
	return fkNode, nil
}
