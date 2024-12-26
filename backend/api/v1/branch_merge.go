package v1

import (
	"fmt"
	"reflect"
	"regexp"
	"slices"
	"strconv"
	"strings"

	"github.com/google/go-cmp/cmp"
	"github.com/pkg/errors"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/testing/protocmp"
	"google.golang.org/protobuf/types/known/timestamppb"

	"github.com/bytebase/bytebase/backend/plugin/db/mysql"
	"github.com/bytebase/bytebase/backend/plugin/db/tidb"
	storepb "github.com/bytebase/bytebase/proto/generated-go/store"
)

var (
	currentTimestampRegexp = regexp.MustCompile(`(?mi)CURRENT_TIMESTAMP(\((?P<fsp>(\d+))?\))?`)
	nowRegexp              = regexp.MustCompile(`(?mi)NOW(\((?P<fsp>(\d+))?\))?`)
)

type timestampDefaultValue interface {
	isTimeStampDefaultValue()
	getFsp() int
}
type currentTimestamp struct {
	// fsp is the fractional seconds precision.
	fsp int
}

func (*currentTimestamp) isTimeStampDefaultValue() {}

func (c *currentTimestamp) getFsp() int { return c.fsp }

type now struct {
	fsp int
}

func (*now) isTimeStampDefaultValue() {}

func (n *now) getFsp() int { return n.fsp }

func buildTimestampDefaultValue(input string) timestampDefaultValue {
	matches := currentTimestampRegexp.FindStringSubmatch(input)
	if matches != nil {
		for i, name := range currentTimestampRegexp.SubexpNames() {
			if name == "fsp" {
				if matches[i] != "" {
					fsp, err := strconv.ParseInt(matches[i], 10, 64)
					if err != nil {
						return nil
					}
					return &currentTimestamp{int(fsp)}
				}
			}
		}
		return &currentTimestamp{}
	}

	matches = nowRegexp.FindStringSubmatch(input)
	if matches != nil {
		for i, name := range nowRegexp.SubexpNames() {
			if name == "fsp" {
				if matches[i] != "" {
					fsp, err := strconv.ParseInt(matches[i], 10, 64)
					if err != nil {
						return nil
					}
					return &now{int(fsp)}
				}
			}
		}
		return &now{}
	}

	return nil
}

var (
	autoRandomRegexp = regexp.MustCompile(`(?mis)^AUTO_RANDOM(\(\s*(?P<shardBit>(\d+))?\s*(,\s*(?P<allocationRange>(\d+)))?\s*\))?`)
)

// https://docs.pingcap.com/tidb/stable/auto-random
type autoRandomDefaultValue struct {
	shardBit        int
	allocationRange int
}

func buildAutoRandomDefaultValue(input string) *autoRandomDefaultValue {
	matches := autoRandomRegexp.FindStringSubmatch(input)
	if matches != nil {
		result := &autoRandomDefaultValue{}
		for i, name := range autoRandomRegexp.SubexpNames() {
			if name == "shardBit" {
				if matches[i] != "" {
					shardBit, err := strconv.ParseInt(matches[i], 10, 64)
					if err != nil {
						return nil
					}
					result.shardBit = int(shardBit)
				}
			} else if name == "allocationRange" {
				if matches[i] != "" {
					allocationRange, err := strconv.ParseInt(matches[i], 10, 64)
					if err != nil {
						return nil
					}
					result.allocationRange = int(allocationRange)
				}
			}
		}
		return result
	}

	return nil
}

func isAutoRandomEquivalent(a, b *autoRandomDefaultValue) bool {
	if a == nil || b == nil {
		return (a == nil) == (b == nil)
	}
	// https://docs.pingcap.com/tidb/stable/auto-random
	canonicalShardBitA, canonicalShardBitB := a.shardBit, b.shardBit
	if canonicalShardBitA == 0 {
		canonicalShardBitA = 5
	}
	if canonicalShardBitB == 0 {
		canonicalShardBitB = 5
	}

	canonicalAllocationRangeA, canonicalAllocationRangeB := a.allocationRange, b.allocationRange
	if canonicalAllocationRangeA == 0 {
		canonicalAllocationRangeA = 64
	}
	if canonicalAllocationRangeB == 0 {
		canonicalAllocationRangeB = 64
	}

	return (canonicalShardBitA == canonicalShardBitB) && (canonicalAllocationRangeA == canonicalAllocationRangeB)
}

type diffAction string

const (
	diffActionCreate diffAction = "CREATE"
	diffActionUpdate diffAction = "UPDATE"
	diffActionDrop   diffAction = "DROP"
)

// tryMerge merges other metadata to current metadata, always returns a non-nil metadata if no error occurs.
func tryMerge(ancestor, head, base *storepb.DatabaseSchemaMetadata, ancestorConfig *storepb.DatabaseConfig, headConfig, baseConfig *storepb.DatabaseConfig, engine storepb.Engine) (*storepb.DatabaseSchemaMetadata, *storepb.DatabaseConfig, error) {
	ancestor, head, base = proto.Clone(ancestor).(*storepb.DatabaseSchemaMetadata), proto.Clone(head).(*storepb.DatabaseSchemaMetadata), proto.Clone(base).(*storepb.DatabaseSchemaMetadata)

	if ancestor == nil {
		ancestor = &storepb.DatabaseSchemaMetadata{}
	}

	diffBetweenAncestorAndHead, err := diffMetadata(ancestor, head)
	if err != nil {
		return nil, nil, errors.Wrap(err, "failed to diff between ancestor and head")
	}
	configDiffBetweenAncestorAndHead := deriveUpdateInfoFromMetadataDiff(diffBetweenAncestorAndHead, headConfig)

	diffBetweenAncestorAndBase, err := diffMetadata(ancestor, base)
	if err != nil {
		return nil, nil, errors.Wrap(err, "failed to diff between ancestor and base")
	}
	configDiffBetweenAncestorAndBase := deriveUpdateInfoFromMetadataDiff(diffBetweenAncestorAndBase, baseConfig)

	if conflict, msg := diffBetweenAncestorAndBase.tryMerge(diffBetweenAncestorAndHead, engine); conflict {
		return nil, nil, errors.Errorf("merge conflict: %s", msg)
	}
	mergedUpdateInfoDiff, err := mergeUpdateInfoDiffRooNode(configDiffBetweenAncestorAndBase, configDiffBetweenAncestorAndHead)
	if err != nil {
		return nil, nil, errors.Wrap(err, "failed to merge update info diff")
	}

	if err := diffBetweenAncestorAndBase.applyDiffTo(ancestor); err != nil {
		return nil, nil, errors.Wrap(err, "failed to apply diff to target")
	}

	config := applyUpdateInfoDiffRootNode(mergedUpdateInfoDiff, ancestorConfig)
	// The updateInfoDiff do not contain the column config, align database config to add the column config.
	config = alignDatabaseConfig(ancestor, config)
	return ancestor, config, nil
}

type diffBaseNode struct {
	action diffAction
}

type metadataDiffRootNode struct {
	schemas map[string]*metadataDiffSchemaNode
}

// tryMerge merges other root node to current root node, stop and return error if conflict occurs.
func (mr *metadataDiffRootNode) tryMerge(other *metadataDiffRootNode, engine storepb.Engine) (bool, string) {
	for _, schema := range mr.schemas {
		otherSchema, in := other.schemas[schema.name]
		if !in {
			continue
		}
		conflict, msg := schema.tryMerge(otherSchema, engine)
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
	diffBaseNode
	name string
	//nolint
	base *storepb.SchemaMetadata
	head *storepb.SchemaMetadata

	tables     map[string]*metadataDiffTableNode
	views      map[string]*metadataDiffViewNode
	functions  map[string]*metadataDiffFunctioNnode
	procedures map[string]*metadataDiffProcedureNode

	// SchemaMetadata contains other object types, likes function, view etc. But we do not support them yet.
}

func (n *metadataDiffSchemaNode) tryMerge(other *metadataDiffSchemaNode, engine storepb.Engine) (bool, string) {
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
		conflict, msg := tableNode.tryMerge(otherTableNode, engine)
		if conflict {
			return true, msg
		}
		delete(other.tables, tableName)
	}

	for _, remainingTable := range other.tables {
		n.tables[remainingTable.name] = remainingTable
	}

	for viewName, viewNode := range n.views {
		otherViewNode, in := other.views[viewName]
		if !in {
			continue
		}
		conflict, msg := viewNode.tryMerge(otherViewNode)
		if conflict {
			return true, msg
		}
		delete(other.views, viewName)
	}

	for _, remainingView := range other.views {
		n.views[remainingView.name] = remainingView
	}

	for functionName, functionNode := range n.functions {
		otherFunctionNode, in := other.functions[functionName]
		if !in {
			continue
		}
		conflict, msg := functionNode.tryMerge(otherFunctionNode)
		if conflict {
			return true, msg
		}
		delete(other.functions, functionName)
	}

	for _, remainingFunction := range other.functions {
		n.functions[remainingFunction.name] = remainingFunction
	}

	for procedureName, procedureNode := range n.procedures {
		otherProcedureNode, in := other.procedures[procedureName]
		if !in {
			continue
		}
		conflict, msg := procedureNode.tryMerge(otherProcedureNode)
		if conflict {
			return true, msg
		}
		delete(other.procedures, procedureName)
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

	sortedViewNames := make([]string, 0, len(n.views))
	for viewName := range n.views {
		sortedViewNames = append(sortedViewNames, viewName)
	}

	sortedFunctionNames := make([]string, 0, len(n.functions))
	for functionName := range n.functions {
		sortedFunctionNames = append(sortedFunctionNames, functionName)
	}

	sortedProcedureNames := make([]string, 0, len(n.procedures))
	for procedureName := range n.procedures {
		sortedProcedureNames = append(sortedProcedureNames, procedureName)
	}

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
		for _, viewName := range sortedViewNames {
			view := n.views[viewName]
			if err := view.applyDiffTo(newSchema); err != nil {
				return errors.Wrapf(err, "failed to apply diff to view %q", view.name)
			}
		}
		for _, functionName := range sortedFunctionNames {
			function := n.functions[functionName]
			if err := function.applyDiffTo(newSchema); err != nil {
				return errors.Wrapf(err, "failed to apply diff to function %q", function.name)
			}
		}
		for _, procedureName := range sortedProcedureNames {
			procedure := n.procedures[procedureName]
			if err := procedure.applyDiffTo(newSchema); err != nil {
				return errors.Wrapf(err, "failed to apply diff to procedure %q", procedure.name)
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
					Name:       n.name,
					Tables:     schema.Tables,
					Views:      schema.Views,
					Functions:  schema.Functions,
					Procedures: schema.Procedures,
				}
				for _, tableName := range sortedTableNames {
					table := n.tables[tableName]
					if err := table.applyDiffTo(newSchema); err != nil {
						return errors.Wrapf(err, "failed to apply diff to table %q", table.name)
					}
				}
				for _, viewName := range sortedViewNames {
					view := n.views[viewName]
					if err := view.applyDiffTo(newSchema); err != nil {
						return errors.Wrapf(err, "failed to apply diff to view %q", view.name)
					}
				}
				for _, functionName := range sortedFunctionNames {
					function := n.functions[functionName]
					if err := function.applyDiffTo(newSchema); err != nil {
						return errors.Wrapf(err, "failed to apply diff to function %q", function.name)
					}
				}
				for _, procedureName := range sortedProcedureNames {
					procedure := n.procedures[procedureName]
					if err := procedure.applyDiffTo(newSchema); err != nil {
						return errors.Wrapf(err, "failed to apply diff to procedure %q", procedure.name)
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
	diffBaseNode
	name string
	//nolint
	base *storepb.TableMetadata
	head *storepb.TableMetadata

	// columnNames is designed to help to handle the column orders.
	// The size of columnNames is always equal to the size of columnsMap,
	// and all the value appeared in columnNames is also the key of columnsMap.
	columnNames []string
	columnsMap  map[string]*metadataDiffColumnNode

	foreignKeys map[string]*metadataDiffForeignKeyNode
	indexes     map[string]*metadataDiffIndexNode
	// TableMetaData contains other object types, likes trigger, index etc. But we do not support them yet.

	// partitionNames is designed to help to handle the partition orders.
	// The size of partitionNames is always equal to the size of partitionsMap,
	// and all the value appeared in partitionNames is also the key of partitionsMap.
	partitionNames []string
	partitionsMap  map[string]*metadataDiffPartitionNode
}

func (n *metadataDiffTableNode) tryMerge(other *metadataDiffTableNode, engine storepb.Engine) (bool, string) {
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
		// Engine and Collation is the attributes are used to display only, would not
		// affect the table schema in schema design.
		if n.head.Comment != other.head.Comment {
			return true, fmt.Sprintf("conflict table comment, one is %s, the other is %s", n.head.Comment, other.head.Comment)
		}
		if n.head.UserComment != other.head.UserComment {
			return true, fmt.Sprintf("conflict table user comment, one is %s, the other is %s", n.head.UserComment, other.head.UserComment)
		}
	}

	if n.action == diffActionUpdate {
		// Engine and Collation is the attributes are used to display only, would not
		// affect the table schema in schema design.
		if other.base.Comment != other.head.Comment {
			if n.base.Comment != n.head.Comment {
				if n.head.Comment != other.head.Comment {
					return true, fmt.Sprintf("conflict table comment, one is %s, the other is %s", n.head.Comment, other.head.Comment)
				}
			} else {
				n.head.Comment = other.head.Comment
			}
		}

		if other.base.UserComment != other.head.UserComment {
			if n.base.UserComment != n.head.UserComment {
				if n.head.UserComment != other.head.UserComment {
					return true, fmt.Sprintf("conflict table user comment, one is %s, the other is %s", n.head.UserComment, other.head.UserComment)
				}
			} else {
				n.head.UserComment = other.head.UserComment
			}
		}
	}

	for _, columnName := range n.columnNames {
		columnNode := n.columnsMap[columnName]
		otherColumnNode, in := other.columnsMap[columnName]
		if !in {
			continue
		}
		conflict, msg := columnNode.tryMerge(otherColumnNode, engine)
		if conflict {
			return true, msg
		}
		delete(other.columnsMap, columnName)
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

	for _, partitionName := range n.partitionNames {
		partitionNode := n.partitionsMap[partitionName]
		otherPartitionNode, in := other.partitionsMap[partitionName]
		if !in {
			continue
		}
		conflict, msg := partitionNode.tryMerge(otherPartitionNode)
		if conflict {
			return true, msg
		}
		delete(other.partitionsMap, partitionName)
	}

	for _, columnName := range other.columnNames {
		// We had deleted the column node which appeared in both table nodes.
		if columnNode, in := other.columnsMap[columnName]; in {
			n.columnsMap[columnName] = columnNode
			n.columnNames = append(n.columnNames, columnName)
			continue
		}
	}

	for _, partition := range other.partitionNames {
		// We had deleted the partition node which appeared in both table nodes.
		if partitionNode, in := other.partitionsMap[partition]; in {
			n.partitionsMap[partition] = partitionNode
			n.partitionNames = append(n.partitionNames, partition)
			continue
		}
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
		newTable := &storepb.TableMetadata{
			Name:        n.name,
			Engine:      n.head.Engine,
			Collation:   n.head.Collation,
			Comment:     n.head.Comment,
			UserComment: n.head.UserComment,
		}
		for _, columnName := range n.columnNames {
			if columnNode, in := n.columnsMap[columnName]; in {
				if err := columnNode.applyDiffTo(newTable); err != nil {
					return errors.Wrapf(err, "failed to apply diff to column %q", columnNode.name)
				}
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
			if err := index.applyDiffTo(newTable); err != nil {
				return errors.Wrapf(err, "failed to apply diff to index %q", index.name)
			}
		}

		for _, partitionName := range n.partitionNames {
			partition := n.partitionsMap[partitionName]
			if err := partition.applyDiffTo(newTable); err != nil {
				return errors.Wrapf(err, "failed to apply diff to partition %q", partition.name)
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
			// Update table currently is only contains diff of columns, foreign keys and indexes.
			// So we do apply column and foreign key diff to target table.
			if table.Name == n.name {
				newTable := &storepb.TableMetadata{
					Name:        n.name,
					Engine:      n.head.Engine,
					Collation:   n.head.Collation,
					Comment:     n.head.Comment,
					UserComment: n.head.UserComment,
					Columns:     table.Columns,
					ForeignKeys: table.ForeignKeys,
					Indexes:     table.Indexes,
					Partitions:  table.Partitions,
				}
				for _, columnName := range n.columnNames {
					if columnNode, in := n.columnsMap[columnName]; in {
						if err := columnNode.applyDiffTo(newTable); err != nil {
							return errors.Wrapf(err, "failed to apply diff to column %q", columnNode.name)
						}
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
					if err := index.applyDiffTo(newTable); err != nil {
						return errors.Wrapf(err, "failed to apply diff to index %q", index.name)
					}
				}
				for _, partitionName := range n.partitionNames {
					partition := n.partitionsMap[partitionName]
					if err := partition.applyDiffTo(newTable); err != nil {
						return errors.Wrapf(err, "failed to apply diff to partition %q", partition.name)
					}
				}
				target.Tables[idx] = newTable
			}
		}
	}
	return nil
}

// Column related.
type metadataDiffColumnNode struct {
	diffBaseNode
	name string
	//nolint
	base *storepb.ColumnMetadata
	head *storepb.ColumnMetadata
}

func (n *metadataDiffColumnNode) tryMerge(other *metadataDiffColumnNode, engine storepb.Engine) (bool, string) {
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
		if !isColumnTypeEqual(n.head.Type, other.head.Type, engine) {
			return true, fmt.Sprintf("conflict column type, one is %s, the other is %s", n.head.Type, other.head.Type)
		}
		if !compareColumnDefaultValue(n.head.DefaultValue, other.head.DefaultValue) {
			return true, fmt.Sprintf("conflict column default value, one is %+v, the other is %+v", n.head.DefaultValue, other.head.DefaultValue)
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
	}
	if n.action == diffActionUpdate {
		if !isColumnTypeEqual(other.base.Type, other.head.Type, engine) {
			if !isColumnTypeEqual(n.base.Type, n.head.Type, engine) {
				if !isColumnTypeEqual(n.head.Type, other.head.Type, engine) {
					return true, fmt.Sprintf("conflict column type, one is %s, the other is %s", n.head.Type, other.head.Type)
				}
			} else {
				n.head.Type = other.head.Type
			}
		}

		if !compareColumnDefaultValue(n.head.DefaultValue, other.head.DefaultValue) {
			if !compareColumnDefaultValue(n.base.DefaultValue, n.head.DefaultValue) {
				if !compareColumnDefaultValue(n.head.DefaultValue, other.head.DefaultValue) {
					return true, fmt.Sprintf("conflict column default value, one is %+v, the other is %+v", n.head.DefaultValue, other.head.DefaultValue)
				}
			} else {
				n.head.DefaultValue = other.head.DefaultValue
			}
		}

		if !compareColumnOnUpdateValue(n.head.OnUpdate, other.head.OnUpdate) {
			if !compareColumnOnUpdateValue(n.base.OnUpdate, n.head.OnUpdate) {
				if !compareColumnOnUpdateValue(n.head.OnUpdate, other.head.OnUpdate) {
					return true, fmt.Sprintf("conflict column onUpdate value, one is %+v, the other is %+v", n.head.OnUpdate, other.head.OnUpdate)
				}
			} else {
				n.head.OnUpdate = other.head.OnUpdate
			}
		}

		if other.base.Nullable != other.head.Nullable {
			if n.base.Nullable != n.head.Nullable {
				if n.head.Nullable != other.head.Nullable {
					return true, fmt.Sprintf("conflict column nullable, one is %t, the other is %t", n.head.Nullable, other.head.Nullable)
				}
			} else {
				n.head.Nullable = other.head.Nullable
			}
		}

		if other.base.Comment != other.head.Comment {
			if n.base.Comment != n.head.Comment {
				if n.head.Comment != other.head.Comment {
					return true, fmt.Sprintf("conflict column comment, one is %s, the other is %s", n.head.Comment, other.head.Comment)
				}
			} else {
				n.head.Comment = other.head.Comment
			}
		}

		if other.base.UserComment != other.head.UserComment {
			if n.base.UserComment != n.head.UserComment {
				if n.head.UserComment != other.head.UserComment {
					return true, fmt.Sprintf("conflict column user comment, one is %s, the other is %s", n.head.UserComment, other.head.UserComment)
				}
			} else {
				n.head.UserComment = other.head.UserComment
			}
		}
	}

	return false, ""
}

func compareColumnOnUpdateValue(a, b string) bool {
	// TODO(zp): The special case should be assosiacted with the engine type.
	aTimestampDefaultValue := buildTimestampDefaultValue(a)
	bTimestampDefaultValue := buildTimestampDefaultValue(b)
	if aTimestampDefaultValue != nil && bTimestampDefaultValue != nil {
		return aTimestampDefaultValue.getFsp() == bTimestampDefaultValue.getFsp()
	}
	return a == b
}

// To avoid getting caught up in the case struggle, we handle some special cases first.
// compareColumnDefaultValue compares the default value of two columns.
// The type of a, b should can be assigned to storepb.isColumnMetadata_DefaultValue.
// return true if the default value of a and b are equal, otherwise return false.
func compareColumnDefaultValue(a, b any) bool {
	// TODO(zp): The special case should be assosiacted with the engine type.
	aExpr, aOK := a.(*storepb.ColumnMetadata_DefaultExpression)
	bExpr, bOK := b.(*storepb.ColumnMetadata_DefaultExpression)
	if aOK && bOK {
		if strings.EqualFold(aExpr.DefaultExpression, "AUTO_INCREMENT") {
			return strings.EqualFold(bExpr.DefaultExpression, "AUTO_INCREMENT")
		}
		aTimestampDefaultValue := buildTimestampDefaultValue(aExpr.DefaultExpression)
		bTimestampDefaultValue := buildTimestampDefaultValue(bExpr.DefaultExpression)
		if aTimestampDefaultValue != nil && bTimestampDefaultValue != nil {
			return aTimestampDefaultValue.getFsp() == bTimestampDefaultValue.getFsp()
		}

		aAutoRandomDefaultValue := buildAutoRandomDefaultValue(aExpr.DefaultExpression)
		bAutoRandomDefaultValue := buildAutoRandomDefaultValue(bExpr.DefaultExpression)
		if aAutoRandomDefaultValue != nil && bAutoRandomDefaultValue != nil {
			return isAutoRandomEquivalent(aAutoRandomDefaultValue, bAutoRandomDefaultValue)
		}
	}

	r := cmp.Diff(a, b, protocmp.Transform())

	return r == ""
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
	diffBaseNode
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
		if !slices.Equal(n.head.Expressions, other.head.Expressions) {
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
					return true, fmt.Sprintf("conflict index expressions, one is %v, the other is %v", n.head.Expressions, other.head.Expressions)
				}
			} else {
				n.head.Expressions = other.head.Expressions
			}
		}

		if other.base.Type != other.head.Type {
			if n.base.Type != n.head.Type {
				if n.head.Type != other.head.Type {
					return true, fmt.Sprintf("conflict index type, one is %s, the other is %s", n.head.Type, other.head.Type)
				}
			} else {
				n.head.Type = other.head.Type
			}
		}

		if other.base.Unique != other.head.Unique {
			if n.base.Unique != n.head.Unique {
				if n.head.Unique != other.head.Unique {
					return true, fmt.Sprintf("conflict index unique, one is %t, the other is %t", n.head.Unique, other.head.Unique)
				}
			} else {
				n.head.Unique = other.head.Unique
			}
		}

		if other.base.Primary != other.head.Primary {
			if n.base.Primary != n.head.Primary {
				if n.head.Primary != other.head.Primary {
					return true, fmt.Sprintf("conflict index primary, one is %t, the other is %t", n.head.Primary, other.head.Primary)
				}
			} else {
				n.head.Primary = other.head.Primary
			}
		}
	}

	return false, ""
}

func (n *metadataDiffIndexNode) applyDiffTo(target *storepb.TableMetadata) error {
	switch n.action {
	case diffActionCreate:
		target.Indexes = append(target.Indexes, n.head)
	case diffActionDrop:
		for i, index := range target.Indexes {
			if index.Name == n.name {
				target.Indexes = append(target.Indexes[:i], target.Indexes[i+1:]...)
				break
			}
		}
	case diffActionUpdate:
		for i, index := range target.Indexes {
			if index.Name == n.name {
				target.Indexes[i] = n.head
				break
			}
		}
	}
	return nil
}

type metadataDiffPartitionNode struct {
	diffBaseNode
	name string
	// nolint
	base *storepb.TablePartitionMetadata
	head *storepb.TablePartitionMetadata

	// subpartitionNames is designed to help to handle the subpartition orders.
	subpartitionNames []string
	subpartitions     map[string]*metadataDiffPartitionNode
}

func (n *metadataDiffPartitionNode) tryMerge(other *metadataDiffPartitionNode) (bool, string) {
	if other == nil {
		return true, "other node check conflict with partition node must not be nil"
	}

	if n.name != other.name {
		return true, fmt.Sprintf("non-expected partition node pair, one is %s, the other is %s", n.name, other.name)
	}
	if n.action != other.action {
		return true, fmt.Sprintf("conflict partition action, one is %s, the other is %s", n.action, other.action)
	}
	if n.action == diffActionDrop {
		return false, ""
	}
	if n.action == diffActionCreate {
		if n.head.Type != other.head.Type {
			return true, fmt.Sprintf("conflict partition type, one is %s, the other is %s", n.head.Type, other.head.Type)
		}
		if !isPartitionExprEqual(n.head.Expression, other.head.Expression) {
			return true, fmt.Sprintf("conflict partition expression, one is %s, the other is %s", n.head.Expression, other.head.Expression)
		}
		if n.head.Value != other.head.Value {
			return true, fmt.Sprintf("conflict partition value, one is %s, the other is %s", n.head.Value, other.head.Value)
		}
	}

	if n.action == diffActionUpdate {
		if other.base.Type != other.head.Type {
			if n.base.Type != n.head.Type {
				if n.head.Type != other.head.Type {
					return true, fmt.Sprintf("conflict partition type, one is %s, the other is %s", n.head.Type, other.head.Type)
				}
			} else {
				n.head.Type = other.head.Type
			}
		}

		if !isPartitionExprEqual(other.base.Expression, other.head.Expression) {
			if !isPartitionExprEqual(n.base.Expression, n.head.Expression) {
				if !isPartitionExprEqual(n.head.Expression, other.head.Expression) {
					return true, fmt.Sprintf("conflict partition expression, one is %s, the other is %s", n.head.Expression, other.head.Expression)
				}
			} else {
				n.head.Expression = other.head.Expression
			}
		}

		if other.base.Value != other.head.Value {
			if n.base.Value != n.head.Value {
				if n.head.Value != other.head.Value {
					return true, fmt.Sprintf("conflict partition value, one is %s, the other is %s", n.head.Value, other.head.Value)
				}
			} else {
				n.head.Value = other.head.Value
			}
		}
	}

	for _, subpartitionName := range n.subpartitionNames {
		subpartitionNode := n.subpartitions[subpartitionName]
		otherSubpartitionNode, in := other.subpartitions[subpartitionName]
		if !in {
			continue
		}
		conflict, msg := subpartitionNode.tryMerge(otherSubpartitionNode)
		if conflict {
			return true, msg
		}
		delete(other.subpartitions, subpartitionName)
	}

	for _, remainingSubpartition := range other.subpartitions {
		n.subpartitions[remainingSubpartition.name] = remainingSubpartition
	}

	return false, ""
}

func isPartitionExprEqual(a, b string) bool {
	ignoreTokens := []string{
		"`", " ", "\t", "\n", "\r", "(", ")",
	}
	for _, token := range ignoreTokens {
		a = strings.ReplaceAll(a, token, "")
		b = strings.ReplaceAll(b, token, "")
	}
	return strings.EqualFold(a, b)
}

func (n *metadataDiffPartitionNode) applyDiffTo(target *storepb.TableMetadata) error {
	switch n.action {
	case diffActionCreate:
		target.Partitions = append(target.Partitions, n.head)
	case diffActionDrop:
		for i, partition := range target.Partitions {
			if partition.Name == n.name {
				target.Partitions = append(target.Partitions[:i], target.Partitions[i+1:]...)
				break
			}
		}
	case diffActionUpdate:
		for i, partition := range target.Partitions {
			if partition.Name == n.name {
				target.Partitions[i] = n.head
				break
			}
		}
	}
	return nil
}

type metadataDiffFunctioNnode struct {
	diffBaseNode
	name string
	//nolint
	base *storepb.FunctionMetadata
	head *storepb.FunctionMetadata
}

func (n *metadataDiffFunctioNnode) tryMerge(other *metadataDiffFunctioNnode) (bool, string) {
	if other == nil {
		return true, "other node check conflict with function node must not be nil"
	}

	if n.name != other.name {
		return true, fmt.Sprintf("non-expected function node pair, one is %s, the other is %s", n.name, other.name)
	}
	if n.action != other.action {
		return true, fmt.Sprintf("conflict function action, one is %s, the other is %s", n.action, other.action)
	}
	if n.action == diffActionDrop {
		return false, ""
	}
	if n.action == diffActionCreate {
		nHeadDefinition := strings.TrimRight(n.head.Definition, ";")
		otherHeadDefinition := strings.TrimRight(other.head.Definition, ";")
		if !equalRoutineDefinition(nHeadDefinition, otherHeadDefinition) {
			return true, fmt.Sprintf("conflict function definition, one is %s, the other is %s", nHeadDefinition, otherHeadDefinition)
		}
	}

	if n.action == diffActionUpdate {
		otherBaseDefinition := strings.TrimRight(other.base.Definition, ";")
		otherHeadDefinition := strings.TrimRight(other.head.Definition, ";")
		if !equalRoutineDefinition(otherBaseDefinition, otherHeadDefinition) {
			nHeadDefinition := strings.TrimRight(n.head.Definition, ";")
			nBaseDefinition := strings.TrimRight(n.base.Definition, ";")
			if !equalRoutineDefinition(nHeadDefinition, nBaseDefinition) {
				if !equalRoutineDefinition(nHeadDefinition, otherHeadDefinition) {
					return true, fmt.Sprintf("conflict function definition, one is %s, the other is %s", nHeadDefinition, otherHeadDefinition)
				}
			} else {
				n.head.Definition = other.head.Definition
			}
		}
	}

	return false, ""
}

func (n *metadataDiffFunctioNnode) applyDiffTo(target *storepb.SchemaMetadata) error {
	if target == nil {
		return errors.New("target must not be nil")
	}

	switch n.action {
	case diffActionCreate:
		newFunction := &storepb.FunctionMetadata{
			Name:       n.name,
			Definition: n.head.Definition,
		}
		target.Functions = append(target.Functions, newFunction)
	case diffActionUpdate:
		for i, function := range target.Functions {
			if function.Name == n.name {
				target.Functions[i] = n.head
				break
			}
		}
	case diffActionDrop:
		for i, function := range target.Functions {
			if function.Name == n.name {
				target.Functions = append(target.Functions[:i], target.Functions[i+1:]...)
				break
			}
		}
		return nil
	}
	return nil
}

type metadataDiffProcedureNode struct {
	diffBaseNode
	name string
	//nolint
	base *storepb.ProcedureMetadata
	head *storepb.ProcedureMetadata
}

func (n *metadataDiffProcedureNode) tryMerge(other *metadataDiffProcedureNode) (bool, string) {
	if other == nil {
		return true, "other node check conflict with procedure node must not be nil"
	}

	if n.name != other.name {
		return true, fmt.Sprintf("non-expected procedure node pair, one is %s, the other is %s", n.name, other.name)
	}
	if n.action != other.action {
		return true, fmt.Sprintf("conflict procedure action, one is %s, the other is %s", n.action, other.action)
	}
	if n.action == diffActionDrop {
		return false, ""
	}
	if n.action == diffActionCreate {
		nHeadDefinition := strings.TrimRight(n.head.Definition, ";")
		otherHeadDefinition := strings.TrimRight(other.head.Definition, ";")
		if !equalRoutineDefinition(nHeadDefinition, otherHeadDefinition) {
			return true, fmt.Sprintf("conflict procedure definition, one is %s, the other is %s", nHeadDefinition, otherHeadDefinition)
		}
	}

	if n.action == diffActionUpdate {
		otherBaseDefinition := strings.TrimRight(other.base.Definition, ";")
		otherHeadDefinition := strings.TrimRight(other.head.Definition, ";")
		if !equalRoutineDefinition(otherBaseDefinition, otherHeadDefinition) {
			nHeadDefinition := strings.TrimRight(n.head.Definition, ";")
			nBaseDefinition := strings.TrimRight(n.base.Definition, ";")
			if !equalRoutineDefinition(nHeadDefinition, nBaseDefinition) {
				if !equalRoutineDefinition(nHeadDefinition, otherHeadDefinition) {
					return true, fmt.Sprintf("conflict procedure definition, one is %s, the other is %s", nHeadDefinition, otherHeadDefinition)
				}
			} else {
				n.head.Definition = other.head.Definition
			}
		}
	}

	return false, ""
}

func (n *metadataDiffProcedureNode) applyDiffTo(target *storepb.SchemaMetadata) error {
	if target == nil {
		return errors.New("target must not be nil")
	}

	switch n.action {
	case diffActionCreate:
		newProcedure := &storepb.ProcedureMetadata{
			Name:       n.name,
			Definition: n.head.Definition,
		}
		target.Procedures = append(target.Procedures, newProcedure)
	case diffActionDrop:
		for i, view := range target.Procedures {
			if view.Name == n.name {
				target.Procedures = append(target.Procedures[:i], target.Procedures[i+1:]...)
				break
			}
		}
		return nil
	case diffActionUpdate:
		for i, procedure := range target.Procedures {
			if procedure.Name == n.name {
				target.Procedures[i] = n.head
				break
			}
		}
	}
	return nil
}

type metadataDiffViewNode struct {
	diffBaseNode
	name string
	//nolint
	base *storepb.ViewMetadata
	head *storepb.ViewMetadata
}

func (n *metadataDiffViewNode) tryMerge(other *metadataDiffViewNode) (bool, string) {
	if other == nil {
		return true, "other node check conflict with view node must not be nil"
	}

	if n.name != other.name {
		return true, fmt.Sprintf("non-expected view node pair, one is %s, the other is %s", n.name, other.name)
	}
	if n.action != other.action {
		return true, fmt.Sprintf("conflict view action, one is %s, the other is %s", n.action, other.action)
	}
	if n.action == diffActionDrop {
		return false, ""
	}
	if n.action == diffActionCreate {
		if !equalViewDefinition(n.head.Definition, other.head.Definition) {
			return true, fmt.Sprintf("conflict view definition, one is %s, the other is %s", n.head.Definition, other.head.Definition)
		}
	}

	if n.action == diffActionUpdate {
		if !equalViewDefinition(other.base.Definition, other.head.Definition) {
			if !equalViewDefinition(n.base.Definition, n.head.Definition) {
				if !equalViewDefinition(n.head.Definition, other.head.Definition) {
					return true, fmt.Sprintf("conflict view definition, one is %s, the other is %s", n.head.Definition, other.head.Definition)
				}
			} else {
				n.head.Definition = other.head.Definition
			}
		}
	}

	return false, ""
}

func equalRoutineDefinition(a, b string) bool {
	ignoreTokens := []string{
		"`", " ", "\t", "\n", "\r",
	}
	for _, token := range ignoreTokens {
		a = strings.ReplaceAll(a, token, "")
		b = strings.ReplaceAll(b, token, "")
	}
	return strings.EqualFold(a, b)
}

func equalViewDefinition(a, b string) bool {
	a = normalizeMySQLViewDefinition(a)
	b = normalizeMySQLViewDefinition(b)
	ignoreTokens := []string{
		"`", " ", "(", ")", "\t", "\n", "\r",
	}
	for _, token := range ignoreTokens {
		a = strings.ReplaceAll(a, token, "")
		b = strings.ReplaceAll(b, token, "")
	}
	return strings.EqualFold(a, b)
}

var qualifiedRe = regexp.MustCompile("`" + `[^` + "`" + `]+` + "`" + `\.` + "`")

func normalizeMySQLViewDefinition(query string) string {
	query = strings.TrimSpace(query)
	if !strings.HasSuffix(query, ";") {
		query += ";"
	}
	trailSymbols := []string{"` ", "`,", "`;"}
	for {
		asIdx := strings.Index(query, " AS ")
		if asIdx == -1 {
			break
		}
		var endCandidates []int
		for _, symbol := range trailSymbols {
			i := strings.Index(query[asIdx:], symbol)
			if i >= 0 {
				endCandidates = append(endCandidates, i)
			}
		}
		if len(endCandidates) == 0 {
			break
		}
		endIdx := endCandidates[0]
		for _, v := range endCandidates {
			if v < endIdx {
				endIdx = v
			}
		}
		query = query[:asIdx] + query[(asIdx+endIdx+1):]
	}
	return qualifiedRe.ReplaceAllString(query, "`")
}

func (n *metadataDiffViewNode) applyDiffTo(target *storepb.SchemaMetadata) error {
	if target == nil {
		return errors.New("target must not be nil")
	}

	switch n.action {
	case diffActionCreate:
		newView := &storepb.ViewMetadata{
			Name:       n.name,
			Definition: n.head.Definition,
		}
		target.Views = append(target.Views, newView)
	case diffActionUpdate:
		for i, view := range target.Views {
			if view.Name == n.name {
				target.Views[i] = n.head
				break
			}
		}
	case diffActionDrop:
		for i, view := range target.Views {
			if view.Name == n.name {
				target.Views = append(target.Views[:i], target.Views[i+1:]...)
				break
			}
		}
		return nil
	}
	return nil
}

// Foreign Key related.
type metadataDiffForeignKeyNode struct {
	diffBaseNode
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
					return true, fmt.Sprintf("conflict foreign key referenced schema, one is %s, the other is %s", n.head.ReferencedSchema, other.head.ReferencedSchema)
				}
			} else {
				n.head.ReferencedSchema = other.head.ReferencedSchema
			}
		}

		if other.base.ReferencedTable != other.head.ReferencedTable {
			if n.base.ReferencedTable != n.head.ReferencedTable {
				if n.head.ReferencedTable != other.head.ReferencedTable {
					return true, fmt.Sprintf("conflict foreign key referenced table, one is %s, the other is %s", n.head.ReferencedTable, other.head.ReferencedTable)
				}
			} else {
				n.head.ReferencedTable = other.head.ReferencedTable
			}
		}

		if other.base.OnDelete != other.head.OnDelete {
			if n.base.OnDelete != n.head.OnDelete {
				if n.head.OnDelete != other.head.OnDelete {
					return true, fmt.Sprintf("conflict foreign key on delete, one is %s, the other is %s", n.head.OnDelete, other.head.OnDelete)
				}
			} else {
				n.head.OnDelete = other.head.OnDelete
			}
		}

		if other.base.OnUpdate != other.head.OnUpdate {
			if n.base.OnUpdate != n.head.OnUpdate {
				if n.head.OnUpdate != other.head.OnUpdate {
					return true, fmt.Sprintf("conflict foreign key on update, one is %s, the other is %s", n.head.OnUpdate, other.head.OnUpdate)
				}
			} else {
				n.head.OnUpdate = other.head.OnUpdate
			}
		}

		if !reflect.DeepEqual(other.base.Columns, other.head.Columns) {
			if !reflect.DeepEqual(n.base.Columns, n.head.Columns) {
				if !reflect.DeepEqual(n.head.Columns, other.head.Columns) {
					return true, fmt.Sprintf("conflict foreign key columns, one is %v, the other is %v", n.head.Columns, other.head.Columns)
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
		diffBaseNode: diffBaseNode{
			action: action,
		},
		name:       name,
		base:       base,
		head:       head,
		tables:     make(map[string]*metadataDiffTableNode),
		views:      make(map[string]*metadataDiffViewNode),
		functions:  make(map[string]*metadataDiffFunctioNnode),
		procedures: make(map[string]*metadataDiffProcedureNode),
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

	viewNamesMap := make(map[string]bool)
	baseViewMap := make(map[string]*storepb.ViewMetadata)
	if base != nil {
		for _, view := range base.Views {
			baseViewMap[view.Name] = view
			viewNamesMap[view.Name] = true
		}
	}

	headViewMap := make(map[string]*storepb.ViewMetadata)
	if head != nil {
		for _, view := range head.Views {
			headViewMap[view.Name] = view
			viewNamesMap[view.Name] = true
		}
	}

	for viewName := range viewNamesMap {
		baseView, headView := baseViewMap[viewName], headViewMap[viewName]
		diffNode, err := diffViewMetadata(baseView, headView)
		if err != nil {
			return nil, errors.Wrapf(err, "failed to diff view %q", viewName)
		}
		if diffNode != nil {
			schemaNode.views[viewName] = diffNode
		}
	}

	functionNamesMap := make(map[string]bool)
	baseFunctionMap := make(map[string]*storepb.FunctionMetadata)
	if base != nil {
		for _, function := range base.Functions {
			baseFunctionMap[function.Name] = function
			functionNamesMap[function.Name] = true
		}
	}

	headFunctionMap := make(map[string]*storepb.FunctionMetadata)
	if head != nil {
		for _, function := range head.Functions {
			headFunctionMap[function.Name] = function
			functionNamesMap[function.Name] = true
		}
	}

	for functionName := range functionNamesMap {
		baseFunction, headFunction := baseFunctionMap[functionName], headFunctionMap[functionName]
		diffNode, err := diffFunctionMetadata(baseFunction, headFunction)
		if err != nil {
			return nil, errors.Wrapf(err, "failed to diff function %q", functionName)
		}
		if diffNode != nil {
			schemaNode.functions[functionName] = diffNode
		}
	}

	procedureNamesMap := make(map[string]bool)
	baseProcedureMap := make(map[string]*storepb.ProcedureMetadata)
	if base != nil {
		for _, procedure := range base.Procedures {
			baseProcedureMap[procedure.Name] = procedure
			procedureNamesMap[procedure.Name] = true
		}
	}

	headProcedureMap := make(map[string]*storepb.ProcedureMetadata)
	if head != nil {
		for _, procedure := range head.Procedures {
			headProcedureMap[procedure.Name] = procedure
			procedureNamesMap[procedure.Name] = true
		}
	}

	for procedureName := range procedureNamesMap {
		baseProcedure, headProcedure := baseProcedureMap[procedureName], headProcedureMap[procedureName]
		diffNode, err := diffProcedureMetadata(baseProcedure, headProcedure)
		if err != nil {
			return nil, errors.Wrapf(err, "failed to diff procedure %q", procedureName)
		}
		if diffNode != nil {
			schemaNode.procedures[procedureName] = diffNode
		}
	}

	if action == diffActionUpdate {
		if len(schemaNode.tables) == 0 && len(schemaNode.views) == 0 && len(schemaNode.functions) == 0 && len(schemaNode.procedures) == 0 {
			return nil, nil
		}
	}

	return schemaNode, nil
}

func diffViewMetadata(base, head *storepb.ViewMetadata) (*metadataDiffViewNode, error) {
	if base == nil && head == nil {
		return nil, errors.New("from and to view metadata must not be nil")
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

	viewNode := &metadataDiffViewNode{
		diffBaseNode: diffBaseNode{
			action: action,
		},
		name: name,
		base: base,
		head: head,
	}

	if action == diffActionUpdate {
		if base.Definition == head.Definition {
			return nil, nil
		}
	}

	return viewNode, nil
}

func diffProcedureMetadata(base, head *storepb.ProcedureMetadata) (*metadataDiffProcedureNode, error) {
	if base == nil && head == nil {
		return nil, errors.New("from and to view metadata must not be nil")
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

	procedureNode := &metadataDiffProcedureNode{
		diffBaseNode: diffBaseNode{
			action: action,
		},
		name: name,
		base: base,
		head: head,
	}

	if action == diffActionUpdate {
		if base.Definition == head.Definition {
			return nil, nil
		}
	}

	return procedureNode, nil
}

func diffFunctionMetadata(base, head *storepb.FunctionMetadata) (*metadataDiffFunctioNnode, error) {
	if base == nil && head == nil {
		return nil, errors.New("from and to view metadata must not be nil")
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

	functionNode := &metadataDiffFunctioNnode{
		diffBaseNode: diffBaseNode{
			action: action,
		},
		name: name,
		base: base,
		head: head,
	}

	if action == diffActionUpdate {
		if base.Definition == head.Definition {
			return nil, nil
		}
	}

	return functionNode, nil
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
		diffBaseNode: diffBaseNode{
			action: action,
		},
		name:          name,
		base:          base,
		head:          head,
		columnsMap:    make(map[string]*metadataDiffColumnNode),
		foreignKeys:   make(map[string]*metadataDiffForeignKeyNode),
		indexes:       make(map[string]*metadataDiffIndexNode),
		partitionsMap: make(map[string]*metadataDiffPartitionNode),
	}

	columnNamesMap := make(map[string]bool)
	var columnNameSlice []string

	baseColumnMap := make(map[string]int)
	if base != nil {
		for idx, column := range base.Columns {
			baseColumnMap[column.Name] = idx
			if _, ok := columnNamesMap[column.Name]; !ok {
				columnNameSlice = append(columnNameSlice, column.Name)
			}
			columnNamesMap[column.Name] = true
		}
	}

	headColumnMap := make(map[string]int)
	if head != nil {
		for idx, column := range head.Columns {
			headColumnMap[column.Name] = idx
			if _, ok := columnNamesMap[column.Name]; !ok {
				columnNameSlice = append(columnNameSlice, column.Name)
			}
			columnNamesMap[column.Name] = true
		}
	}

	for _, columnName := range columnNameSlice {
		baseColumnIdx, baseColumnOk := baseColumnMap[columnName]
		headColumnIdx, headColumnOk := headColumnMap[columnName]
		var baseColumn, headColumn *storepb.ColumnMetadata
		if baseColumnOk {
			baseColumn = base.Columns[baseColumnIdx]
		}
		if headColumnOk {
			headColumn = head.Columns[headColumnIdx]
		}
		if baseColumnIdx != 0 {
			baseColumn = base.Columns[baseColumnIdx]
		}
		diffNode, err := diffColumnMetadata(baseColumn, headColumn)
		if err != nil {
			return nil, errors.Wrapf(err, "failed to diff column %q", columnName)
		}
		if diffNode != nil {
			tableNode.columnsMap[columnName] = diffNode
			tableNode.columnNames = append(tableNode.columnNames, columnName)
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

	indexNamesMap := make(map[string]bool)

	baseIndexKeyMap := make(map[string]*storepb.IndexMetadata)
	if base != nil {
		for _, index := range base.Indexes {
			baseIndexKeyMap[index.Name] = index
			indexNamesMap[index.Name] = true
		}
	}

	headIndexKeyMap := make(map[string]*storepb.IndexMetadata)
	if head != nil {
		for _, index := range head.Indexes {
			headIndexKeyMap[index.Name] = index
			indexNamesMap[index.Name] = true
		}
	}

	for indexName := range indexNamesMap {
		baseIndex, headIndex := baseIndexKeyMap[indexName], headIndexKeyMap[indexName]
		diffNode, err := diffIndexMetadata(baseIndex, headIndex)
		if err != nil {
			return nil, errors.Wrapf(err, "failed to diff index %q", indexName)
		}
		if diffNode != nil {
			tableNode.indexes[indexName] = diffNode
		}
	}

	partitionNamesMap := make(map[string]bool)
	var partitionNameSlice []string

	basePartitionMap := make(map[string]int)
	if base != nil {
		for idx, partition := range base.Partitions {
			basePartitionMap[partition.Name] = idx
			if _, ok := partitionNamesMap[partition.Name]; !ok {
				partitionNameSlice = append(partitionNameSlice, partition.Name)
			}
			partitionNamesMap[partition.Name] = true
		}
	}

	headPartitionMap := make(map[string]int)
	if head != nil {
		for idx, partition := range head.Partitions {
			headPartitionMap[partition.Name] = idx
			if _, ok := partitionNamesMap[partition.Name]; !ok {
				partitionNameSlice = append(partitionNameSlice, partition.Name)
			}
			partitionNamesMap[partition.Name] = true
		}
	}

	for _, partitionName := range partitionNameSlice {
		basePartitionIdx, basePartitionOk := basePartitionMap[partitionName]
		headPartitionIdx, headPartitionOk := headPartitionMap[partitionName]
		var basePartition, headPartition *storepb.TablePartitionMetadata
		if basePartitionOk {
			basePartition = base.Partitions[basePartitionIdx]
		}
		if headPartitionOk {
			headPartition = head.Partitions[headPartitionIdx]
		}
		if basePartitionIdx != 0 {
			basePartition = base.Partitions[basePartitionIdx]
		}
		diffNode, err := diffPartitionMetadata(basePartition, headPartition)
		if err != nil {
			return nil, errors.Wrapf(err, "failed to diff partition %q", partitionName)
		}
		if diffNode != nil {
			tableNode.partitionsMap[partitionName] = diffNode
			tableNode.partitionNames = append(tableNode.partitionNames, partitionName)
		}
	}

	if action == diffActionUpdate {
		if len(tableNode.columnsMap) == 0 &&
			len(tableNode.foreignKeys) == 0 &&
			len(tableNode.indexes) == 0 &&
			len(tableNode.partitionsMap) == 0 &&
			!(tableNode.base.GetComment() != tableNode.head.GetComment()) &&
			!(tableNode.base.GetUserComment() != tableNode.head.GetUserComment()) {
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
		diffBaseNode: diffBaseNode{
			action: action,
		},
		name: name,
		base: base,
		head: head,
	}

	if action == diffActionUpdate {
		if !proto.Equal(base, head) {
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
		diffBaseNode: diffBaseNode{
			action: action,
		},
		name: name,
		base: base,
		head: head,
	}

	if action == diffActionUpdate {
		if !proto.Equal(base, head) {
			return fkNode, nil
		}
		return nil, nil
	}
	return fkNode, nil
}

func diffIndexMetadata(base, head *storepb.IndexMetadata) (*metadataDiffIndexNode, error) {
	if base == nil && head == nil {
		return nil, errors.New("base and head index metadata cannot be nil both")
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

	indexNode := &metadataDiffIndexNode{
		diffBaseNode: diffBaseNode{
			action: action,
		},
		name: name,
		base: base,
		head: head,
	}

	if action == diffActionUpdate {
		if !proto.Equal(base, head) {
			return indexNode, nil
		}
		return nil, nil
	}
	return indexNode, nil
}

func diffPartitionMetadata(base, head *storepb.TablePartitionMetadata) (*metadataDiffPartitionNode, error) {
	if base == nil && head == nil {
		return nil, errors.New("base and head partition metadata cannot be nil both")
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

	partitionNode := &metadataDiffPartitionNode{
		diffBaseNode: diffBaseNode{
			action: action,
		},
		name:          name,
		base:          base,
		head:          head,
		subpartitions: make(map[string]*metadataDiffPartitionNode),
	}

	baseSubpartitionMap := make(map[string]*storepb.TablePartitionMetadata)
	if base != nil {
		for _, partition := range base.Subpartitions {
			baseSubpartitionMap[partition.Name] = partition
		}
	}

	headSubpartitionMap := make(map[string]*storepb.TablePartitionMetadata)
	if head != nil {
		for _, partition := range head.Subpartitions {
			headSubpartitionMap[partition.Name] = partition
		}
	}

	subpartitionNamesMap := make(map[string]bool)
	var subpartitionNamesSlice []string

	for subpartitionName := range baseSubpartitionMap {
		subpartitionNamesMap[subpartitionName] = true
		subpartitionNamesSlice = append(subpartitionNamesSlice, subpartitionName)
	}

	for subpartitionName := range headSubpartitionMap {
		if _, ok := subpartitionNamesMap[subpartitionName]; !ok {
			subpartitionNamesSlice = append(subpartitionNamesSlice, subpartitionName)
		}
		subpartitionNamesMap[subpartitionName] = true
	}

	for _, subpartitionName := range subpartitionNamesSlice {
		baseSubpartition, headSubpartition := baseSubpartitionMap[subpartitionName], headSubpartitionMap[subpartitionName]
		diffNode, err := diffPartitionMetadata(baseSubpartition, headSubpartition)
		if err != nil {
			return nil, errors.Wrapf(err, "failed to diff subpartition %q", subpartitionName)
		}
		if diffNode != nil {
			partitionNode.subpartitions[subpartitionName] = diffNode
			partitionNode.subpartitionNames = append(partitionNode.subpartitionNames, subpartitionName)
		}
	}

	if action == diffActionUpdate {
		if len(partitionNode.subpartitions) > 0 || !proto.Equal(base, head) {
			return partitionNode, nil
		}
		return nil, nil
	}

	return partitionNode, nil
}

func isColumnTypeEqual(a, b string, engine storepb.Engine) bool {
	switch engine {
	case storepb.Engine_MYSQL:
		canonicalA := mysql.GetColumnTypeCanonicalSynonym(a)
		canonicalB := mysql.GetColumnTypeCanonicalSynonym(b)
		return strings.EqualFold(canonicalA, canonicalB)
	case storepb.Engine_TIDB:
		canonicalA := tidb.GetColumnTypeCanonicalSynonym(a)
		canonicalB := tidb.GetColumnTypeCanonicalSynonym(b)
		return strings.EqualFold(canonicalA, canonicalB)
	default:
		return strings.EqualFold(a, b)
	}
}

type updateInfo struct {
	lastUpdatedTime *timestamppb.Timestamp
	lastUpdater     string
	sourceBranch    string
}

func (u *updateInfo) clone() *updateInfo {
	if u == nil {
		return nil
	}
	return &updateInfo{
		//nolint
		lastUpdatedTime: proto.Clone(u.lastUpdatedTime).(*timestamppb.Timestamp),
		lastUpdater:     u.lastUpdater,
		sourceBranch:    u.sourceBranch,
	}
}

type updateInfoDiffRootNode struct {
	schemas map[string]*updateInfoDiffSchemaNode
}

type updateInfoDiffSchemaNode struct {
	diffBaseNode
	name       string
	tables     map[string]*updateInfoDiffTableNode
	views      map[string]*updateInfoDiffViewNode
	functions  map[string]*updateInfoDiffFunctionNode
	procedures map[string]*updateInfoDiffProcedureNode
}

type updateInfoDiffTableNode struct {
	diffBaseNode
	name string
	// If the action is diffActionDelete, the updateInfo should be nil.
	updateInfo *updateInfo
}

type updateInfoDiffViewNode struct {
	diffBaseNode
	name string
	// If the action is diffActionDelete, the updateInfo should be nil.
	updateInfo *updateInfo
}

type updateInfoDiffFunctionNode struct {
	diffBaseNode
	name string
	// If the action is diffActionDelete, the updateInfo should be nil.
	updateInfo *updateInfo
}

type updateInfoDiffProcedureNode struct {
	diffBaseNode
	name string
	// If the action is diffActionDelete, the updateInfo should be nil.
	updateInfo *updateInfo
}

// deriveUpdateInfoFromMetadataDiff derives the update info diff from the metadata diff.
func deriveUpdateInfoFromMetadataDiff(metadataDiff *metadataDiffRootNode, headDatabaseConfig *storepb.DatabaseConfig) *updateInfoDiffRootNode {
	if metadataDiff == nil {
		return nil
	}
	rootNode := &updateInfoDiffRootNode{
		schemas: make(map[string]*updateInfoDiffSchemaNode),
	}

	schemaConfigMap := buildMap(headDatabaseConfig.GetSchemas(), func(schemaConfig *storepb.SchemaCatalog) string {
		return schemaConfig.GetName()
	})
	for _, metadataSchemaDiff := range metadataDiff.schemas {
		action := metadataSchemaDiff.action
		updateInfoSchemaDiff := &updateInfoDiffSchemaNode{
			diffBaseNode: diffBaseNode{
				action: action,
			},
			name:       metadataSchemaDiff.name,
			tables:     make(map[string]*updateInfoDiffTableNode),
			views:      make(map[string]*updateInfoDiffViewNode),
			functions:  make(map[string]*updateInfoDiffFunctionNode),
			procedures: make(map[string]*updateInfoDiffProcedureNode),
		}
		schemaConfig := schemaConfigMap[metadataSchemaDiff.name]
		tableConfigMap := buildMap(schemaConfig.GetTables(), func(tableConfig *storepb.TableCatalog) string {
			return tableConfig.GetName()
		})
		for tableName, table := range metadataSchemaDiff.tables {
			action := table.action
			updateInfoSchemaDiff.tables[tableName] = &updateInfoDiffTableNode{
				diffBaseNode: diffBaseNode{
					action: action,
				},
				name: tableName,
			}
			if action != diffActionDrop {
				tableConfig := tableConfigMap[tableName]
				updateInfoSchemaDiff.tables[tableName].updateInfo = &updateInfo{
					lastUpdatedTime: tableConfig.GetUpdateTime(),
					lastUpdater:     tableConfig.GetUpdater(),
					sourceBranch:    tableConfig.GetSourceBranch(),
				}
			}
		}

		// Views
		viewConfigMap := buildMap(schemaConfig.GetViewConfigs(), func(viewConfig *storepb.ViewConfig) string {
			return viewConfig.GetName()
		})
		for viewName, view := range metadataSchemaDiff.views {
			action := view.action
			updateInfoSchemaDiff.views[viewName] = &updateInfoDiffViewNode{
				diffBaseNode: diffBaseNode{
					action: action,
				},
				name: viewName,
			}
			if action != diffActionDrop {
				viewConfig := viewConfigMap[viewName]
				updateInfoSchemaDiff.views[viewName].updateInfo = &updateInfo{
					lastUpdatedTime: viewConfig.GetUpdateTime(),
					lastUpdater:     viewConfig.GetUpdater(),
					sourceBranch:    viewConfig.GetSourceBranch(),
				}
			}
		}

		// Functions
		functionConfigMap := buildMap(schemaConfig.GetFunctionConfigs(), func(functionConfig *storepb.FunctionConfig) string {
			return functionConfig.GetName()
		})
		for functionName, function := range metadataSchemaDiff.functions {
			action := function.action
			updateInfoSchemaDiff.functions[functionName] = &updateInfoDiffFunctionNode{
				diffBaseNode: diffBaseNode{
					action: action,
				},
				name: functionName,
			}
			if action != diffActionDrop {
				functionConfig := functionConfigMap[functionName]
				updateInfoSchemaDiff.functions[functionName].updateInfo = &updateInfo{
					lastUpdatedTime: functionConfig.GetUpdateTime(),
					lastUpdater:     functionConfig.GetUpdater(),
					sourceBranch:    functionConfig.GetSourceBranch(),
				}
			}
		}

		// Procedures
		procedureConfigMap := buildMap(schemaConfig.GetProcedureConfigs(), func(procedureConfig *storepb.ProcedureConfig) string {
			return procedureConfig.GetName()
		})
		for procedureName, procedure := range metadataSchemaDiff.procedures {
			action := procedure.action
			updateInfoSchemaDiff.procedures[procedureName] = &updateInfoDiffProcedureNode{
				diffBaseNode: diffBaseNode{
					action: action,
				},
				name: procedureName,
			}
			if action != diffActionDrop {
				procedureConfig := procedureConfigMap[procedureName]
				updateInfoSchemaDiff.procedures[procedureName].updateInfo = &updateInfo{
					lastUpdatedTime: procedureConfig.GetUpdateTime(),
					lastUpdater:     procedureConfig.GetUpdater(),
					sourceBranch:    procedureConfig.GetSourceBranch(),
				}
			}
		}
		rootNode.schemas[metadataSchemaDiff.name] = updateInfoSchemaDiff
	}

	return rootNode
}

func mergeUpdateInfoDiffRooNode(a, b *updateInfoDiffRootNode) (*updateInfoDiffRootNode, error) {
	if a == nil && b == nil {
		return nil, nil
	}
	if a == nil {
		return b, nil
	}
	if b == nil {
		return a, nil
	}
	rootNode := &updateInfoDiffRootNode{
		schemas: make(map[string]*updateInfoDiffSchemaNode),
	}

	schemaNamesMap := make(map[string]bool)
	for schemaName := range a.schemas {
		schemaNamesMap[schemaName] = true
	}
	for schemaName := range b.schemas {
		schemaNamesMap[schemaName] = true
	}

	for schemaName := range schemaNamesMap {
		aSchema, aOk := a.schemas[schemaName]
		bSchema, bOk := b.schemas[schemaName]
		if aOk && bOk {
			mergedSchema, err := mergeUpdateInfoDiffSchemaNode(aSchema, bSchema)
			if err != nil {
				return nil, err
			}
			rootNode.schemas[schemaName] = mergedSchema
			continue
		}
		if aOk {
			rootNode.schemas[schemaName] = aSchema
			continue
		}
		if bOk {
			rootNode.schemas[schemaName] = bSchema
			continue
		}
	}

	return rootNode, nil
}

func mergeUpdateInfoDiffSchemaNode(a, b *updateInfoDiffSchemaNode) (*updateInfoDiffSchemaNode, error) {
	if a == nil && b == nil {
		return nil, nil
	}
	if a == nil {
		return b, nil
	}
	if b == nil {
		return a, nil
	}

	if a.action != b.action {
		return nil, errors.Errorf("conflict schema config action, one is %s, the other is %s", a.action, b.action)
	}
	schemaNode := &updateInfoDiffSchemaNode{
		diffBaseNode: diffBaseNode{
			action: b.action,
		},
		name:       b.name,
		tables:     make(map[string]*updateInfoDiffTableNode),
		views:      make(map[string]*updateInfoDiffViewNode),
		functions:  make(map[string]*updateInfoDiffFunctionNode),
		procedures: make(map[string]*updateInfoDiffProcedureNode),
	}

	tableNameMap := make(map[string]bool)
	for tableName := range a.tables {
		tableNameMap[tableName] = true
	}
	for tableName := range b.tables {
		tableNameMap[tableName] = true
	}
	for tableName := range tableNameMap {
		aTable, aOk := a.tables[tableName]
		bTable, bOk := b.tables[tableName]
		if aOk && bOk {
			mergedTable, err := mergeUpdateInfoDiffTableNode(aTable, bTable)
			if err != nil {
				return nil, err
			}
			schemaNode.tables[tableName] = mergedTable
			continue
		}
		if aOk {
			schemaNode.tables[tableName] = aTable
			continue
		}
		if bOk {
			schemaNode.tables[tableName] = bTable
			continue
		}
	}

	viewNameMap := make(map[string]bool)
	for viewName := range a.views {
		viewNameMap[viewName] = true
	}
	for viewName := range b.views {
		viewNameMap[viewName] = true
	}
	for viewName := range viewNameMap {
		aView, aOk := a.views[viewName]
		bView, bOk := b.views[viewName]
		if aOk && bOk {
			mergedView, err := mergeUpdateInfoDiffViewNode(aView, bView)
			if err != nil {
				return nil, err
			}
			schemaNode.views[viewName] = mergedView
			continue
		}
		if aOk {
			schemaNode.views[viewName] = aView
			continue
		}
		if bOk {
			schemaNode.views[viewName] = bView
			continue
		}
	}

	procedureNameMap := make(map[string]bool)
	for procedureName := range a.procedures {
		procedureNameMap[procedureName] = true
	}
	for procedureName := range b.procedures {
		procedureNameMap[procedureName] = true
	}
	for procedureName := range procedureNameMap {
		aProcedure, aOk := a.procedures[procedureName]
		bProcedure, bOk := b.procedures[procedureName]
		if aOk && bOk {
			mergedProcedure, err := mergeUpdateInfoDiffProcedureNode(aProcedure, bProcedure)
			if err != nil {
				return nil, err
			}
			schemaNode.procedures[procedureName] = mergedProcedure
			continue
		}
		if aOk {
			schemaNode.procedures[procedureName] = aProcedure
			continue
		}
		if bOk {
			schemaNode.procedures[procedureName] = bProcedure
			continue
		}
	}

	functionNameMap := make(map[string]bool)
	for functionName := range a.functions {
		functionNameMap[functionName] = true
	}
	for functionName := range b.functions {
		functionNameMap[functionName] = true
	}
	for functionName := range functionNameMap {
		aFunction, aOk := a.functions[functionName]
		bFunction, bOk := b.functions[functionName]
		if aOk && bOk {
			mergedFunction, err := mergeUpdateInfoDiffFunctionNode(aFunction, bFunction)
			if err != nil {
				return nil, err
			}
			schemaNode.functions[functionName] = mergedFunction
			continue
		}
		if aOk {
			schemaNode.functions[functionName] = aFunction
			continue
		}
		if bOk {
			schemaNode.functions[functionName] = bFunction
			continue
		}
	}

	return schemaNode, nil
}

func mergeUpdateInfoDiffTableNode(a, b *updateInfoDiffTableNode) (*updateInfoDiffTableNode, error) {
	if a == nil && b == nil {
		return nil, nil
	}
	if a == nil {
		return b, nil
	}
	if b == nil {
		return a, nil
	}

	if a.action != b.action {
		return nil, errors.Errorf("conflict table config action, one is %s, the other is %s", a.action, b.action)
	}

	return &updateInfoDiffTableNode{
		diffBaseNode: diffBaseNode{
			action: b.action,
		},
		name:       b.name,
		updateInfo: b.updateInfo.clone(),
	}, nil
}

func mergeUpdateInfoDiffViewNode(a, b *updateInfoDiffViewNode) (*updateInfoDiffViewNode, error) {
	if a == nil && b == nil {
		return nil, nil
	}
	if a == nil {
		return b, nil
	}
	if b == nil {
		return a, nil
	}

	if a.action != b.action {
		return nil, errors.Errorf("conflict view config action, one is %s, the other is %s", a.action, b.action)
	}

	return &updateInfoDiffViewNode{
		diffBaseNode: diffBaseNode{
			action: b.action,
		},
		name:       b.name,
		updateInfo: b.updateInfo.clone(),
	}, nil
}

func mergeUpdateInfoDiffProcedureNode(a, b *updateInfoDiffProcedureNode) (*updateInfoDiffProcedureNode, error) {
	if a == nil && b == nil {
		return nil, nil
	}
	if a == nil {
		return b, nil
	}
	if b == nil {
		return a, nil
	}

	if a.action != b.action {
		return nil, errors.Errorf("conflict procedure config action, one is %s, the other is %s", a.action, b.action)
	}

	return &updateInfoDiffProcedureNode{
		diffBaseNode: diffBaseNode{
			action: b.action,
		},
		name:       b.name,
		updateInfo: b.updateInfo.clone(),
	}, nil
}

func mergeUpdateInfoDiffFunctionNode(a, b *updateInfoDiffFunctionNode) (*updateInfoDiffFunctionNode, error) {
	if a == nil && b == nil {
		return nil, nil
	}
	if a == nil {
		return b, nil
	}
	if b == nil {
		return a, nil
	}

	if a.action != b.action {
		return nil, errors.Errorf("conflict function config action, one is %s, the other is %s", a.action, b.action)
	}

	return &updateInfoDiffFunctionNode{
		diffBaseNode: diffBaseNode{
			action: b.action,
		},
		name:       b.name,
		updateInfo: b.updateInfo.clone(),
	}, nil
}

func applyUpdateInfoDiffRootNode(a *updateInfoDiffRootNode, target *storepb.DatabaseConfig) *storepb.DatabaseConfig {
	if a == nil {
		//nolint
		return proto.Clone(target).(*storepb.DatabaseConfig)
	}

	schemaConfigMap := buildMap(target.GetSchemas(), func(schemaConfig *storepb.SchemaCatalog) string {
		return schemaConfig.GetName()
	})

	var schemaConfigs []*storepb.SchemaCatalog
	for schemaName, schema := range a.schemas {
		switch schema.action {
		case diffActionDrop:
			continue
		default:
			schemaConfig := schemaConfigMap[schemaName]
			if schemaConfig == nil {
				schemaConfig = &storepb.SchemaCatalog{
					Name: schema.name,
				}
			}
			appliedSchema := applyUpdateInfoDiffSchemaNode(schema, schemaConfig)
			schemaConfigs = append(schemaConfigs, appliedSchema)
		}
	}

	return &storepb.DatabaseConfig{
		Name:    target.GetName(),
		Schemas: schemaConfigs,
	}
}

func applyUpdateInfoDiffSchemaNode(a *updateInfoDiffSchemaNode, target *storepb.SchemaCatalog) *storepb.SchemaCatalog {
	if a == nil {
		//nolint
		return proto.Clone(target).(*storepb.SchemaCatalog)
	}

	tableCatalogMap := buildMap(target.GetTables(), func(tableConfig *storepb.TableCatalog) string {
		return tableConfig.GetName()
	})
	var tableCatalogs []*storepb.TableCatalog
	for tableName, table := range a.tables {
		switch table.action {
		case diffActionDrop:
			delete(tableCatalogMap, tableName)
			continue
		default:
			tableCatalog := tableCatalogMap[tableName]
			if tableCatalog == nil {
				tableCatalog = &storepb.TableCatalog{
					Name: tableName,
				}
			}
			tableCatalog.UpdateTime = table.updateInfo.lastUpdatedTime
			tableCatalog.Updater = table.updateInfo.lastUpdater
			tableCatalog.SourceBranch = table.updateInfo.sourceBranch
			tableCatalogs = append(tableCatalogs, tableCatalog)
			delete(tableCatalogMap, tableName)
		}
	}
	// Add the remaining table configs in the target schema config.
	for _, tableCatalog := range tableCatalogMap {
		tableCatalogs = append(tableCatalogs, tableCatalog)
	}

	viewConfigMap := buildMap(target.GetViewConfigs(), func(viewConfig *storepb.ViewConfig) string {
		return viewConfig.GetName()
	})
	var viewConfigs []*storepb.ViewConfig
	for viewName, view := range a.views {
		switch view.action {
		case diffActionDrop:
			delete(viewConfigMap, viewName)
			continue
		default:
			viewConfig := viewConfigMap[viewName]
			if viewConfig == nil {
				viewConfig = &storepb.ViewConfig{
					Name: viewName,
				}
			}
			viewConfig.UpdateTime = view.updateInfo.lastUpdatedTime
			viewConfig.Updater = view.updateInfo.lastUpdater
			viewConfig.SourceBranch = view.updateInfo.sourceBranch
			viewConfigs = append(viewConfigs, viewConfig)
			delete(viewConfigMap, viewName)
		}
	}
	// Add the remaining view configs in the target schema config.
	for _, viewConfig := range viewConfigMap {
		viewConfigs = append(viewConfigs, viewConfig)
	}

	procedureConfigMap := buildMap(target.GetProcedureConfigs(), func(procedureConfig *storepb.ProcedureConfig) string {
		return procedureConfig.GetName()
	})
	var procedureConfigs []*storepb.ProcedureConfig
	for procedureName, procedure := range a.procedures {
		switch procedure.action {
		case diffActionDrop:
			delete(procedureConfigMap, procedureName)
			continue
		default:
			procedureConfig := procedureConfigMap[procedureName]
			if procedureConfig == nil {
				procedureConfig = &storepb.ProcedureConfig{
					Name: procedureName,
				}
			}
			procedureConfig.UpdateTime = procedure.updateInfo.lastUpdatedTime
			procedureConfig.Updater = procedure.updateInfo.lastUpdater
			procedureConfig.SourceBranch = procedure.updateInfo.sourceBranch
			procedureConfigs = append(procedureConfigs, procedureConfig)
			delete(procedureConfigMap, procedureName)
		}
	}
	// Add the remaining procedure configs in the target schema config.
	for _, procedureConfig := range procedureConfigMap {
		procedureConfigs = append(procedureConfigs, procedureConfig)
	}

	functionConfigMap := buildMap(target.GetFunctionConfigs(), func(functionConfig *storepb.FunctionConfig) string {
		return functionConfig.GetName()
	})
	var functionConfigs []*storepb.FunctionConfig
	for functionName, function := range a.functions {
		switch function.action {
		case diffActionDrop:
			delete(functionConfigMap, functionName)
			continue
		default:
			functionConfig := functionConfigMap[functionName]
			if functionConfig == nil {
				functionConfig = &storepb.FunctionConfig{
					Name: functionName,
				}
			}
			functionConfig.UpdateTime = function.updateInfo.lastUpdatedTime
			functionConfig.Updater = function.updateInfo.lastUpdater
			functionConfig.SourceBranch = function.updateInfo.sourceBranch
			functionConfigs = append(functionConfigs, functionConfig)
			delete(functionConfigMap, functionName)
		}
	}
	// Add the remaining function configs in the target schema config.
	for _, functionConfig := range functionConfigMap {
		functionConfigs = append(functionConfigs, functionConfig)
	}

	return &storepb.SchemaCatalog{
		Name:             target.GetName(),
		Tables:           tableCatalogs,
		ViewConfigs:      viewConfigs,
		FunctionConfigs:  functionConfigs,
		ProcedureConfigs: procedureConfigs,
	}
}
