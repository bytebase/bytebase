package v1

import (
	"fmt"

	"google.golang.org/protobuf/types/known/wrapperspb"

	"github.com/bytebase/bytebase/backend/common"
	"github.com/bytebase/bytebase/backend/store"
	storepb "github.com/bytebase/bytebase/proto/generated-go/store"
	v1pb "github.com/bytebase/bytebase/proto/generated-go/v1"
)

func convertStoreDatabaseMetadata(database *store.DatabaseMessage, metadata *storepb.DatabaseSchemaMetadata, config *storepb.DatabaseConfig, requestView v1pb.DatabaseMetadataView, filter *metadataFilter) *v1pb.DatabaseMetadata {
	if metadata == nil {
		return nil
	}
	m := &v1pb.DatabaseMetadata{
		CharacterSet: metadata.CharacterSet,
		Collation:    metadata.Collation,
	}
	if database != nil {
		m.Name = fmt.Sprintf("%s%s/%s%s%s", common.InstanceNamePrefix, database.InstanceID, common.DatabaseIDPrefix, database.DatabaseName, common.MetadataSuffix)
	}
	for _, schema := range metadata.Schemas {
		if filter != nil && (schema.Name != "" && filter.schema != schema.Name) {
			continue
		}
		s := &v1pb.SchemaMetadata{
			Name: schema.Name,
		}
		for _, table := range schema.Tables {
			if filter != nil && filter.table != table.Name {
				continue
			}
			s.Tables = append(s.Tables, convertStoreTableMetadata(table, requestView))
		}
		// Only return table for request with a filter.
		if filter != nil {
			m.Schemas = append(m.Schemas, s)
			continue
		}
		for _, view := range schema.Views {
			v1View := &v1pb.ViewMetadata{
				Name: view.Name,
			}
			if requestView == v1pb.DatabaseMetadataView_DATABASE_METADATA_VIEW_FULL {
				var dependentColumnList []*v1pb.DependentColumn
				for _, dependentColumn := range view.DependentColumns {
					dependentColumnList = append(dependentColumnList, &v1pb.DependentColumn{
						Schema: dependentColumn.Schema,
						Table:  dependentColumn.Table,
						Column: dependentColumn.Column,
					})
				}
				v1View.Definition = view.Definition
				v1View.Comment = view.Comment
				v1View.DependentColumns = dependentColumnList
			}

			s.Views = append(s.Views, v1View)
		}
		for _, function := range schema.Functions {
			v1Func := &v1pb.FunctionMetadata{
				Name: function.Name,
			}
			if requestView == v1pb.DatabaseMetadataView_DATABASE_METADATA_VIEW_FULL {
				v1Func.Definition = function.Definition
			}
			s.Functions = append(s.Functions, v1Func)
		}
		for _, task := range schema.Tasks {
			v1Task := &v1pb.TaskMetadata{
				Name: task.Name,
			}
			if requestView == v1pb.DatabaseMetadataView_DATABASE_METADATA_VIEW_FULL {
				v1Task.Id = task.Id
				v1Task.Owner = task.Owner
				v1Task.Comment = task.Comment
				v1Task.Warehouse = task.Warehouse
				v1Task.Schedule = task.Schedule
				v1Task.Predecessors = task.Predecessors
				v1Task.State = v1pb.TaskMetadata_State(task.State)
				v1Task.Condition = task.Condition
				v1Task.Definition = task.Definition
			}
			s.Tasks = append(s.Tasks, v1Task)
		}
		for _, stream := range schema.Streams {
			v1Stream := &v1pb.StreamMetadata{
				Name: stream.Name,
			}
			if requestView == v1pb.DatabaseMetadataView_DATABASE_METADATA_VIEW_FULL {
				v1Stream.TableName = stream.TableName
				v1Stream.Owner = stream.Owner
				v1Stream.Comment = stream.Comment
				v1Stream.Type = v1pb.StreamMetadata_Type(stream.Type)
				v1Stream.Stale = stream.Stale
				v1Stream.Mode = v1pb.StreamMetadata_Mode(stream.Mode)
				v1Stream.Definition = stream.Definition
			}
			s.Streams = append(s.Streams, v1Stream)
		}
		m.Schemas = append(m.Schemas, s)
	}
	for _, extension := range metadata.Extensions {
		m.Extensions = append(m.Extensions, &v1pb.ExtensionMetadata{
			Name:        extension.Name,
			Schema:      extension.Schema,
			Version:     extension.Version,
			Description: extension.Description,
		})
	}

	if requestView == v1pb.DatabaseMetadataView_DATABASE_METADATA_VIEW_FULL {
		databaseConfig := convertStoreDatabaseConfig(config, filter)
		if databaseConfig != nil {
			m.SchemaConfigs = databaseConfig.SchemaConfigs
		}
	}
	return m
}

func convertStoreTableMetadata(table *storepb.TableMetadata, view v1pb.DatabaseMetadataView) *v1pb.TableMetadata {
	if table == nil {
		return nil
	}
	t := &v1pb.TableMetadata{
		Name:           table.Name,
		Engine:         table.Engine,
		Collation:      table.Collation,
		RowCount:       table.RowCount,
		DataSize:       table.DataSize,
		IndexSize:      table.IndexSize,
		DataFree:       table.DataFree,
		CreateOptions:  table.CreateOptions,
		Comment:        table.Comment,
		Classification: table.Classification,
		UserComment:    table.UserComment,
	}
	for _, partition := range table.Partitions {
		t.Partitions = append(t.Partitions, convertStoreTablePartitionMetadata(partition))
	}
	// We only return the table info for basic view.
	if view != v1pb.DatabaseMetadataView_DATABASE_METADATA_VIEW_FULL {
		return t
	}

	for _, column := range table.Columns {
		t.Columns = append(t.Columns, convertStoreColumnMetadata(column))
	}
	for _, index := range table.Indexes {
		t.Indexes = append(t.Indexes, &v1pb.IndexMetadata{
			Name:        index.Name,
			Expressions: index.Expressions,
			Type:        index.Type,
			Unique:      index.Unique,
			Primary:     index.Primary,
			Visible:     index.Visible,
			Comment:     index.Comment,
		})
	}
	for _, foreignKey := range table.ForeignKeys {
		t.ForeignKeys = append(t.ForeignKeys, &v1pb.ForeignKeyMetadata{
			Name:              foreignKey.Name,
			Columns:           foreignKey.Columns,
			ReferencedSchema:  foreignKey.ReferencedSchema,
			ReferencedTable:   foreignKey.ReferencedTable,
			ReferencedColumns: foreignKey.ReferencedColumns,
			OnDelete:          foreignKey.OnDelete,
			OnUpdate:          foreignKey.OnUpdate,
			MatchType:         foreignKey.MatchType,
		})
	}
	return t
}

func convertStoreTablePartitionMetadata(partition *storepb.TablePartitionMetadata) *v1pb.TablePartitionMetadata {
	if partition == nil {
		return nil
	}
	metadata := &v1pb.TablePartitionMetadata{
		Name:       partition.Name,
		Expression: partition.Expression,
	}
	switch partition.Type {
	case storepb.TablePartitionMetadata_RANGE:
		metadata.Type = v1pb.TablePartitionMetadata_RANGE
	case storepb.TablePartitionMetadata_LIST:
		metadata.Type = v1pb.TablePartitionMetadata_LIST
	case storepb.TablePartitionMetadata_HASH:
		metadata.Type = v1pb.TablePartitionMetadata_HASH
	default:
		metadata.Type = v1pb.TablePartitionMetadata_TYPE_UNSPECIFIED
	}
	for _, subpartition := range partition.Subpartitions {
		metadata.Subpartitions = append(metadata.Subpartitions, convertStoreTablePartitionMetadata(subpartition))
	}
	return metadata
}

func convertStoreColumnMetadata(column *storepb.ColumnMetadata) *v1pb.ColumnMetadata {
	if column == nil {
		return nil
	}
	metadata := &v1pb.ColumnMetadata{
		Name:           column.Name,
		Position:       column.Position,
		HasDefault:     column.DefaultValue != nil,
		Nullable:       column.Nullable,
		Type:           column.Type,
		CharacterSet:   column.CharacterSet,
		Collation:      column.Collation,
		Comment:        column.Comment,
		Classification: column.Classification,
		UserComment:    column.UserComment,
	}
	if metadata.HasDefault {
		switch value := column.DefaultValue.(type) {
		case *storepb.ColumnMetadata_Default:
			if value.Default == nil {
				metadata.Default = &v1pb.ColumnMetadata_DefaultNull{DefaultNull: true}
			} else {
				metadata.Default = &v1pb.ColumnMetadata_DefaultString{DefaultString: value.Default.Value}
			}
		case *storepb.ColumnMetadata_DefaultNull:
			metadata.Default = &v1pb.ColumnMetadata_DefaultNull{DefaultNull: true}
		case *storepb.ColumnMetadata_DefaultExpression:
			metadata.Default = &v1pb.ColumnMetadata_DefaultExpression{DefaultExpression: value.DefaultExpression}
		}
	}
	return metadata
}

func convertStoreDatabaseConfig(config *storepb.DatabaseConfig, filter *metadataFilter) *v1pb.DatabaseConfig {
	if config == nil {
		return nil
	}
	databaseConfig := &v1pb.DatabaseConfig{
		Name: config.Name,
	}
	for _, schema := range config.SchemaConfigs {
		if filter != nil && filter.schema != schema.Name {
			continue
		}
		s := &v1pb.SchemaConfig{
			Name: schema.Name,
		}
		for _, table := range schema.TableConfigs {
			if filter != nil && filter.table != table.Name {
				continue
			}
			s.TableConfigs = append(s.TableConfigs, convertStoreTableConfig(table))
		}
		databaseConfig.SchemaConfigs = append(databaseConfig.SchemaConfigs, s)
	}
	return databaseConfig
}

func convertStoreTableConfig(table *storepb.TableConfig) *v1pb.TableConfig {
	if table == nil {
		return nil
	}
	t := &v1pb.TableConfig{
		Name: table.Name,
	}
	for _, column := range table.ColumnConfigs {
		t.ColumnConfigs = append(t.ColumnConfigs, convertStoreColumnConfig(column))
	}
	return t
}

func convertStoreColumnConfig(column *storepb.ColumnConfig) *v1pb.ColumnConfig {
	if column == nil {
		return nil
	}
	return &v1pb.ColumnConfig{
		Name:           column.Name,
		SemanticTypeId: column.SemanticTypeId,
		Labels:         column.Labels,
	}
}

func convertV1TableMetadata(table *v1pb.TableMetadata) *storepb.TableMetadata {
	t := &storepb.TableMetadata{
		Name:           table.Name,
		Engine:         table.Engine,
		Collation:      table.Collation,
		RowCount:       table.RowCount,
		DataSize:       table.DataSize,
		IndexSize:      table.IndexSize,
		DataFree:       table.DataFree,
		CreateOptions:  table.CreateOptions,
		Comment:        table.Comment,
		Classification: table.Classification,
		UserComment:    table.UserComment,
	}
	for _, column := range table.Columns {
		t.Columns = append(t.Columns, convertV1ColumnMetadata(column))
	}
	for _, index := range table.Indexes {
		t.Indexes = append(t.Indexes, &storepb.IndexMetadata{
			Name:        index.Name,
			Expressions: index.Expressions,
			Type:        index.Type,
			Unique:      index.Unique,
			Primary:     index.Primary,
			Visible:     index.Visible,
			Comment:     index.Comment,
		})
	}
	for _, foreignKey := range table.ForeignKeys {
		t.ForeignKeys = append(t.ForeignKeys, &storepb.ForeignKeyMetadata{
			Name:              foreignKey.Name,
			Columns:           foreignKey.Columns,
			ReferencedSchema:  foreignKey.ReferencedSchema,
			ReferencedTable:   foreignKey.ReferencedTable,
			ReferencedColumns: foreignKey.ReferencedColumns,
			OnDelete:          foreignKey.OnDelete,
			OnUpdate:          foreignKey.OnUpdate,
			MatchType:         foreignKey.MatchType,
		})
	}
	return t
}

func convertV1ColumnMetadata(column *v1pb.ColumnMetadata) *storepb.ColumnMetadata {
	metadata := &storepb.ColumnMetadata{
		Name:           column.Name,
		Position:       column.Position,
		Nullable:       column.Nullable,
		Type:           column.Type,
		CharacterSet:   column.CharacterSet,
		Collation:      column.Collation,
		Comment:        column.Comment,
		Classification: column.Classification,
		UserComment:    column.UserComment,
	}

	if column.HasDefault {
		switch value := column.Default.(type) {
		case *v1pb.ColumnMetadata_DefaultString:
			metadata.DefaultValue = &storepb.ColumnMetadata_Default{Default: wrapperspb.String(value.DefaultString)}
		case *v1pb.ColumnMetadata_DefaultNull:
			metadata.DefaultValue = &storepb.ColumnMetadata_DefaultNull{DefaultNull: true}
		case *v1pb.ColumnMetadata_DefaultExpression:
			metadata.DefaultValue = &storepb.ColumnMetadata_DefaultExpression{DefaultExpression: value.DefaultExpression}
		}
	}
	return metadata
}

func convertV1DatabaseConfig(databaseConfig *v1pb.DatabaseConfig) *storepb.DatabaseConfig {
	if databaseConfig == nil {
		return nil
	}

	config := &storepb.DatabaseConfig{
		Name: databaseConfig.Name,
	}
	for _, schema := range databaseConfig.SchemaConfigs {
		s := &storepb.SchemaConfig{
			Name: schema.Name,
		}
		for _, table := range schema.TableConfigs {
			t := &storepb.TableConfig{
				Name: table.Name,
			}
			for _, column := range table.ColumnConfigs {
				t.ColumnConfigs = append(t.ColumnConfigs, &storepb.ColumnConfig{
					Name:           column.Name,
					SemanticTypeId: column.SemanticTypeId,
					Labels:         column.Labels,
				})
			}
			s.TableConfigs = append(s.TableConfigs, t)
		}
		config.SchemaConfigs = append(config.SchemaConfigs, s)
	}
	return config
}

func convertV1TableConfig(table *v1pb.TableConfig) *storepb.TableConfig {
	if table == nil {
		return nil
	}

	t := &storepb.TableConfig{
		Name: table.Name,
	}
	for _, column := range table.ColumnConfigs {
		t.ColumnConfigs = append(t.ColumnConfigs, convertV1ColumnConfig(column))
	}
	return t
}

func convertV1ColumnConfig(column *v1pb.ColumnConfig) *storepb.ColumnConfig {
	if column == nil {
		return nil
	}

	return &storepb.ColumnConfig{
		Name:           column.Name,
		SemanticTypeId: column.SemanticTypeId,
		Labels:         column.Labels,
	}
}
