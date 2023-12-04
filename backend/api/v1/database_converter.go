package v1

import (
	"google.golang.org/protobuf/types/known/wrapperspb"

	storepb "github.com/bytebase/bytebase/proto/generated-go/store"
	v1pb "github.com/bytebase/bytebase/proto/generated-go/v1"
)

func convertStoreDatabaseMetadata(metadata *storepb.DatabaseSchemaMetadata, config *storepb.DatabaseConfig, requestView v1pb.DatabaseMetadataView, filter *metadataFilter) *v1pb.DatabaseMetadata {
	m := &v1pb.DatabaseMetadata{
		CharacterSet: metadata.GetCharacterSet(),
		Collation:    metadata.GetCollation(),
	}
	for _, schema := range metadata.GetSchemas() {
		if schema == nil {
			continue
		}
		if filter != nil && (schema.GetName() != "" && filter.schema != schema.GetName()) {
			continue
		}
		s := &v1pb.SchemaMetadata{
			Name: schema.GetName(),
		}
		for _, table := range schema.GetTables() {
			if table == nil {
				continue
			}
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
		for _, view := range schema.GetViews() {
			if view == nil {
				continue
			}
			v1View := &v1pb.ViewMetadata{
				Name: view.GetName(),
			}
			if requestView == v1pb.DatabaseMetadataView_DATABASE_METADATA_VIEW_FULL {
				var dependentColumnList []*v1pb.DependentColumn
				for _, dependentColumn := range view.GetDependentColumns() {
					dependentColumnList = append(dependentColumnList, &v1pb.DependentColumn{
						Schema: dependentColumn.GetSchema(),
						Table:  dependentColumn.GetTable(),
						Column: dependentColumn.GetColumn(),
					})
				}
				v1View.Definition = view.GetDefinition()
				v1View.Comment = view.GetComment()
				v1View.DependentColumns = dependentColumnList
			}

			s.Views = append(s.Views, v1View)
		}
		for _, function := range schema.Functions {
			if function == nil {
				continue
			}
			v1Func := &v1pb.FunctionMetadata{
				Name: function.GetName(),
			}
			if requestView == v1pb.DatabaseMetadataView_DATABASE_METADATA_VIEW_FULL {
				v1Func.Definition = function.GetDefinition()
			}
			s.Functions = append(s.Functions, v1Func)
		}
		for _, task := range schema.GetTasks() {
			if task == nil {
				continue
			}
			v1Task := &v1pb.TaskMetadata{
				Name: task.GetName(),
			}
			if requestView == v1pb.DatabaseMetadataView_DATABASE_METADATA_VIEW_FULL {
				v1Task.Id = task.GetId()
				v1Task.Owner = task.GetOwner()
				v1Task.Comment = task.GetComment()
				v1Task.Warehouse = task.GetWarehouse()
				v1Task.Schedule = task.GetSchedule()
				v1Task.Predecessors = task.GetPredecessors()
				v1Task.State = v1pb.TaskMetadata_State(task.GetState())
				v1Task.Condition = task.GetCondition()
				v1Task.Definition = task.GetDefinition()
			}
			s.Tasks = append(s.Tasks, v1Task)
		}
		for _, stream := range schema.Streams {
			if stream == nil {
				continue
			}
			v1Stream := &v1pb.StreamMetadata{
				Name: stream.GetName(),
			}
			if requestView == v1pb.DatabaseMetadataView_DATABASE_METADATA_VIEW_FULL {
				v1Stream.TableName = stream.GetTableName()
				v1Stream.Owner = stream.GetOwner()
				v1Stream.Comment = stream.GetComment()
				v1Stream.Type = v1pb.StreamMetadata_Type(stream.GetType())
				v1Stream.Stale = stream.GetStale()
				v1Stream.Mode = v1pb.StreamMetadata_Mode(stream.GetMode())
				v1Stream.Definition = stream.GetDefinition()
			}
			s.Streams = append(s.Streams, v1Stream)
		}
		m.Schemas = append(m.Schemas, s)
	}
	for _, extension := range metadata.GetExtensions() {
		if extension == nil {
			continue
		}
		m.Extensions = append(m.Extensions, &v1pb.ExtensionMetadata{
			Name:        extension.GetName(),
			Schema:      extension.GetSchema(),
			Version:     extension.GetVersion(),
			Description: extension.GetDescription(),
		})
	}

	if requestView == v1pb.DatabaseMetadataView_DATABASE_METADATA_VIEW_FULL {
		databaseConfig := convertStoreDatabaseConfig(config, filter)
		if databaseConfig != nil {
			m.SchemaConfigs = databaseConfig.GetSchemaConfigs()
		}
	}
	return m
}

func convertStoreTableMetadata(table *storepb.TableMetadata, view v1pb.DatabaseMetadataView) *v1pb.TableMetadata {
	t := &v1pb.TableMetadata{
		Name:           table.GetName(),
		Engine:         table.GetEngine(),
		Collation:      table.GetCollation(),
		RowCount:       table.GetRowCount(),
		DataSize:       table.GetDataSize(),
		IndexSize:      table.GetIndexSize(),
		DataFree:       table.GetDataFree(),
		CreateOptions:  table.GetCreateOptions(),
		Comment:        table.GetComment(),
		Classification: table.GetClassification(),
		UserComment:    table.GetUserComment(),
	}
	for _, partition := range table.Partitions {
		if partition == nil {
			continue
		}
		t.Partitions = append(t.Partitions, convertStoreTablePartitionMetadata(partition))
	}
	// We only return the table info for basic view.
	if view != v1pb.DatabaseMetadataView_DATABASE_METADATA_VIEW_FULL {
		return t
	}

	for _, column := range table.GetColumns() {
		if column == nil {
			continue
		}
		t.Columns = append(t.Columns, convertStoreColumnMetadata(column))
	}
	for _, index := range table.GetIndexes() {
		if index == nil {
			continue
		}
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
	for _, foreignKey := range table.GetForeignKeys() {
		if foreignKey == nil {
			continue
		}
		t.ForeignKeys = append(t.ForeignKeys, &v1pb.ForeignKeyMetadata{
			Name:              foreignKey.GetName(),
			Columns:           foreignKey.GetColumns(),
			ReferencedSchema:  foreignKey.GetReferencedSchema(),
			ReferencedTable:   foreignKey.GetReferencedTable(),
			ReferencedColumns: foreignKey.GetReferencedColumns(),
			OnDelete:          foreignKey.GetOnDelete(),
			OnUpdate:          foreignKey.GetOnUpdate(),
			MatchType:         foreignKey.GetMatchType(),
		})
	}
	return t
}

func convertStoreTablePartitionMetadata(partition *storepb.TablePartitionMetadata) *v1pb.TablePartitionMetadata {
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
	for _, subpartition := range partition.GetSubpartitions() {
		if subpartition == nil {
			continue
		}
		metadata.Subpartitions = append(metadata.Subpartitions, convertStoreTablePartitionMetadata(subpartition))
	}
	return metadata
}

func convertStoreColumnMetadata(column *storepb.ColumnMetadata) *v1pb.ColumnMetadata {
	metadata := &v1pb.ColumnMetadata{
		Name:           column.GetName(),
		Position:       column.GetPosition(),
		HasDefault:     column.GetDefaultValue() != nil,
		Nullable:       column.GetNullable(),
		Type:           column.GetType(),
		CharacterSet:   column.GetCharacterSet(),
		Collation:      column.GetCollation(),
		Comment:        column.GetComment(),
		Classification: column.GetClassification(),
		UserComment:    column.GetUserComment(),
	}
	if metadata.HasDefault {
		switch value := column.GetDefaultValue().(type) {
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
	databaseConfig := &v1pb.DatabaseConfig{
		Name: config.GetName(),
	}
	for _, schema := range config.GetSchemaConfigs() {
		if schema == nil {
			continue
		}
		if filter != nil && filter.schema != schema.GetName() {
			continue
		}
		s := &v1pb.SchemaConfig{
			Name: schema.GetName(),
		}
		for _, table := range schema.GetTableConfigs() {
			if table == nil {
				continue
			}
			if filter != nil && filter.table != table.GetName() {
				continue
			}
			s.TableConfigs = append(s.TableConfigs, convertStoreTableConfig(table))
		}
		databaseConfig.SchemaConfigs = append(databaseConfig.SchemaConfigs, s)
	}
	return databaseConfig
}

func convertStoreTableConfig(table *storepb.TableConfig) *v1pb.TableConfig {
	t := &v1pb.TableConfig{
		Name: table.GetName(),
	}
	for _, column := range table.GetColumnConfigs() {
		if column == nil {
			continue
		}
		t.ColumnConfigs = append(t.ColumnConfigs, convertStoreColumnConfig(column))
	}
	return t
}

func convertStoreColumnConfig(column *storepb.ColumnConfig) *v1pb.ColumnConfig {
	return &v1pb.ColumnConfig{
		Name:           column.GetName(),
		SemanticTypeId: column.GetSemanticTypeId(),
		Labels:         column.GetLabels(),
	}
}

func convertV1TableMetadata(table *v1pb.TableMetadata) *storepb.TableMetadata {
	t := &storepb.TableMetadata{
		Name:           table.GetName(),
		Engine:         table.GetEngine(),
		Collation:      table.GetCollation(),
		RowCount:       table.GetRowCount(),
		DataSize:       table.GetDataSize(),
		IndexSize:      table.GetIndexSize(),
		DataFree:       table.GetDataFree(),
		CreateOptions:  table.GetCreateOptions(),
		Comment:        table.GetComment(),
		Classification: table.GetClassification(),
		UserComment:    table.GetUserComment(),
	}
	for _, column := range table.Columns {
		if column == nil {
			continue
		}
		t.Columns = append(t.Columns, convertV1ColumnMetadata(column))
	}
	for _, index := range table.GetIndexes() {
		if index == nil {
			continue
		}
		t.Indexes = append(t.Indexes, &storepb.IndexMetadata{
			Name:        index.GetName(),
			Expressions: index.GetExpressions(),
			Type:        index.GetType(),
			Unique:      index.GetUnique(),
			Primary:     index.GetPrimary(),
			Visible:     index.GetVisible(),
			Comment:     index.GetComment(),
		})
	}
	for _, foreignKey := range table.ForeignKeys {
		if foreignKey == nil {
			continue
		}
		t.ForeignKeys = append(t.ForeignKeys, &storepb.ForeignKeyMetadata{
			Name:              foreignKey.GetName(),
			Columns:           foreignKey.GetColumns(),
			ReferencedSchema:  foreignKey.GetReferencedSchema(),
			ReferencedTable:   foreignKey.GetReferencedTable(),
			ReferencedColumns: foreignKey.GetReferencedColumns(),
			OnDelete:          foreignKey.GetOnDelete(),
			OnUpdate:          foreignKey.GetOnUpdate(),
			MatchType:         foreignKey.GetMatchType(),
		})
	}
	return t
}

func convertV1ColumnMetadata(column *v1pb.ColumnMetadata) *storepb.ColumnMetadata {
	metadata := &storepb.ColumnMetadata{
		Name:           column.GetName(),
		Position:       column.GetPosition(),
		Nullable:       column.GetNullable(),
		Type:           column.GetType(),
		CharacterSet:   column.GetCharacterSet(),
		Collation:      column.GetCollation(),
		Comment:        column.GetComment(),
		Classification: column.GetClassification(),
		UserComment:    column.GetUserComment(),
	}

	if column.HasDefault {
		switch value := column.GetDefault().(type) {
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
	config := &storepb.DatabaseConfig{
		Name: databaseConfig.GetName(),
	}
	for _, schema := range databaseConfig.GetSchemaConfigs() {
		if schema == nil {
			continue
		}
		s := &storepb.SchemaConfig{
			Name: schema.GetName(),
		}
		for _, table := range schema.GetTableConfigs() {
			if table == nil {
				continue
			}
			t := &storepb.TableConfig{
				Name: table.GetName(),
			}
			for _, column := range table.GetColumnConfigs() {
				if column == nil {
					continue
				}
				t.ColumnConfigs = append(t.ColumnConfigs, &storepb.ColumnConfig{
					Name:           column.GetName(),
					SemanticTypeId: column.GetSemanticTypeId(),
					Labels:         column.GetLabels(),
				})
			}
			s.TableConfigs = append(s.TableConfigs, t)
		}
		config.SchemaConfigs = append(config.SchemaConfigs, s)
	}
	return config
}

func convertV1TableConfig(table *v1pb.TableConfig) *storepb.TableConfig {
	t := &storepb.TableConfig{
		Name: table.GetName(),
	}
	for _, column := range table.GetColumnConfigs() {
		if column == nil {
			continue
		}
		t.ColumnConfigs = append(t.ColumnConfigs, convertV1ColumnConfig(column))
	}
	return t
}

func convertV1ColumnConfig(column *v1pb.ColumnConfig) *storepb.ColumnConfig {
	return &storepb.ColumnConfig{
		Name:           column.GetName(),
		SemanticTypeId: column.GetSemanticTypeId(),
		Labels:         column.GetLabels(),
	}
}
