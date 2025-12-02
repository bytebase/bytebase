package v1

import (
	"strings"

	storepb "github.com/bytebase/bytebase/backend/generated-go/store"
	v1pb "github.com/bytebase/bytebase/backend/generated-go/v1"
)

func convertStoreDatabaseMetadata(metadata *storepb.DatabaseSchemaMetadata, filter *metadataFilter, limit int) *v1pb.DatabaseMetadata {
	m := &v1pb.DatabaseMetadata{
		CharacterSet: metadata.CharacterSet,
		Collation:    metadata.Collation,
		Owner:        metadata.Owner,
	}

	for _, schema := range metadata.Schemas {
		if schema == nil {
			continue
		}
		if filter != nil && filter.schema != nil {
			if schema.Name != *filter.schema {
				continue
			}
		}
		s := &v1pb.SchemaMetadata{
			Name:     schema.Name,
			Owner:    schema.Owner,
			Comment:  schema.Comment,
			SkipDump: schema.SkipDump,
		}
		for _, table := range schema.Tables {
			if table == nil {
				continue
			}
			if filter != nil && filter.table != nil {
				if !filter.table.wildcard {
					if table.Name != filter.table.name {
						continue
					}
				} else {
					if !strings.Contains(strings.ToLower(table.Name), filter.table.name) {
						continue
					}
				}
			}
			s.Tables = append(s.Tables, convertStoreTableMetadata(table))
			if limit > 0 && len(s.Tables) >= limit {
				break
			}
		}
		for _, externalTable := range schema.ExternalTables {
			if externalTable == nil {
				continue
			}
			s.ExternalTables = append(s.ExternalTables, convertStoreExternalTableMetadata(externalTable))
		}
		// Only return table for request with a filter.
		if filter != nil {
			m.Schemas = append(m.Schemas, s)
			continue
		}

		for _, view := range schema.Views {
			if view == nil {
				continue
			}
			v1View := &v1pb.ViewMetadata{
				Name:       view.Name,
				Definition: view.Definition,
				Comment:    view.Comment,
				SkipDump:   view.SkipDump,
			}

			for _, column := range view.Columns {
				if column == nil {
					continue
				}
				v1View.Columns = append(v1View.Columns, convertStoreColumnMetadata(column))
			}

			for _, dependencyColumn := range view.DependencyColumns {
				if dependencyColumn == nil {
					continue
				}
				v1View.DependencyColumns = append(v1View.DependencyColumns, &v1pb.DependencyColumn{
					Schema: dependencyColumn.Schema,
					Table:  dependencyColumn.Table,
					Column: dependencyColumn.Column,
				})
			}

			for _, trigger := range view.Triggers {
				if trigger == nil {
					continue
				}

				v1View.Triggers = append(v1View.Triggers, convertStoreTriggerMetadata(trigger))
			}

			s.Views = append(s.Views, v1View)
		}
		for _, function := range schema.Functions {
			if function == nil {
				continue
			}
			v1Func := &v1pb.FunctionMetadata{
				Name:                function.Name,
				Definition:          function.Definition,
				Signature:           function.Signature,
				CharacterSetClient:  function.CharacterSetClient,
				CollationConnection: function.CollationConnection,
				DatabaseCollation:   function.DatabaseCollation,
				SqlMode:             function.SqlMode,
				Comment:             function.Comment,
				SkipDump:            function.SkipDump,
			}
			for _, dep := range function.DependencyTables {
				v1Func.DependencyTables = append(v1Func.DependencyTables, &v1pb.DependencyTable{
					Schema: dep.Schema,
					Table:  dep.Table,
				})
			}
			s.Functions = append(s.Functions, v1Func)
		}
		for _, procedure := range schema.Procedures {
			if procedure == nil {
				continue
			}
			v1Procedure := &v1pb.ProcedureMetadata{
				Name:                procedure.Name,
				Definition:          procedure.Definition,
				Signature:           procedure.Signature,
				CharacterSetClient:  procedure.CharacterSetClient,
				CollationConnection: procedure.CollationConnection,
				DatabaseCollation:   procedure.DatabaseCollation,
				SqlMode:             procedure.SqlMode,
				Comment:             procedure.Comment,
				SkipDump:            procedure.SkipDump,
			}
			s.Procedures = append(s.Procedures, v1Procedure)
		}
		for _, p := range schema.Packages {
			if p == nil {
				continue
			}
			v1Package := &v1pb.PackageMetadata{
				Name:       p.Name,
				Definition: p.Definition,
			}
			s.Packages = append(s.Packages, v1Package)
		}
		for _, task := range schema.Tasks {
			if task == nil {
				continue
			}
			v1Task := &v1pb.TaskMetadata{
				Name:         task.Name,
				Id:           task.Id,
				Owner:        task.Owner,
				Comment:      task.Comment,
				Warehouse:    task.Warehouse,
				Schedule:     task.Schedule,
				Predecessors: task.Predecessors,
				State:        v1pb.TaskMetadata_State(task.State),
				Condition:    task.Condition,
				Definition:   task.Definition,
			}
			s.Tasks = append(s.Tasks, v1Task)
		}
		for _, stream := range schema.Streams {
			if stream == nil {
				continue
			}
			v1Stream := &v1pb.StreamMetadata{
				Name:       stream.Name,
				TableName:  stream.TableName,
				Owner:      stream.Owner,
				Comment:    stream.Comment,
				Type:       v1pb.StreamMetadata_Type(stream.Type),
				Stale:      stream.Stale,
				Mode:       v1pb.StreamMetadata_Mode(stream.Mode),
				Definition: stream.Definition,
			}
			s.Streams = append(s.Streams, v1Stream)
		}

		for _, sequence := range schema.Sequences {
			if sequence == nil {
				continue
			}
			v1Sequence := &v1pb.SequenceMetadata{
				Name:        sequence.Name,
				DataType:    sequence.DataType,
				Start:       sequence.Start,
				MinValue:    sequence.MinValue,
				MaxValue:    sequence.MaxValue,
				Increment:   sequence.Increment,
				Cycle:       sequence.Cycle,
				CacheSize:   sequence.CacheSize,
				LastValue:   sequence.LastValue,
				OwnerTable:  sequence.OwnerTable,
				OwnerColumn: sequence.OwnerColumn,
				Comment:     sequence.Comment,
				SkipDump:    sequence.SkipDump,
			}
			s.Sequences = append(s.Sequences, v1Sequence)
		}

		for _, event := range schema.Events {
			if event == nil {
				continue
			}
			v1Event := &v1pb.EventMetadata{
				Name:                event.Name,
				TimeZone:            event.TimeZone,
				Definition:          event.Definition,
				SqlMode:             event.SqlMode,
				CharacterSetClient:  event.CharacterSetClient,
				CollationConnection: event.CollationConnection,
				Comment:             event.Comment,
			}
			s.Events = append(s.Events, v1Event)
		}

		for _, enum := range schema.EnumTypes {
			if enum == nil {
				continue
			}
			v1Enum := &v1pb.EnumTypeMetadata{
				Name:     enum.Name,
				Values:   enum.Values,
				Comment:  enum.Comment,
				SkipDump: enum.SkipDump,
			}
			s.EnumTypes = append(s.EnumTypes, v1Enum)
		}

		for _, matview := range schema.MaterializedViews {
			if matview == nil {
				continue
			}
			v1Matview := &v1pb.MaterializedViewMetadata{
				Name:       matview.Name,
				Definition: matview.Definition,
				Comment:    matview.Comment,
				SkipDump:   matview.SkipDump,
			}

			for _, dependencyColumn := range matview.DependencyColumns {
				if dependencyColumn == nil {
					continue
				}
				v1Matview.DependencyColumns = append(v1Matview.DependencyColumns,
					&v1pb.DependencyColumn{
						Schema: dependencyColumn.Schema,
						Table:  dependencyColumn.Table,
						Column: dependencyColumn.Column,
					})
			}

			for _, index := range matview.Indexes {
				if index == nil {
					continue
				}
				v1Matview.Indexes = append(v1Matview.Indexes, convertStoreIndexMetadata(index))
			}

			for _, trigger := range matview.Triggers {
				if trigger == nil {
					continue
				}
				v1Matview.Triggers = append(v1Matview.Triggers, convertStoreTriggerMetadata(trigger))
			}

			s.MaterializedViews = append(s.MaterializedViews, v1Matview)
		}

		m.Schemas = append(m.Schemas, s)
	}
	for _, extension := range metadata.Extensions {
		if extension == nil {
			continue
		}
		m.Extensions = append(m.Extensions, &v1pb.ExtensionMetadata{
			Name:        extension.Name,
			Schema:      extension.Schema,
			Version:     extension.Version,
			Description: extension.Description,
		})
	}
	return m
}

func convertStoreIndexMetadata(index *storepb.IndexMetadata) *v1pb.IndexMetadata {
	return &v1pb.IndexMetadata{
		Name:              index.Name,
		Expressions:       index.Expressions,
		KeyLength:         index.KeyLength,
		Descending:        index.Descending,
		Type:              index.Type,
		Unique:            index.Unique,
		Primary:           index.Primary,
		Visible:           index.Visible,
		Comment:           index.Comment,
		Definition:        index.Definition,
		ParentIndexSchema: index.ParentIndexSchema,
		ParentIndexName:   index.ParentIndexName,
		Granularity:       index.Granularity,
		IsConstraint:      index.IsConstraint,
		SpatialConfig:     convertStoreSpatialIndexConfig(index.SpatialConfig),
		OpclassNames:      index.OpclassNames,
		OpclassDefaults:   index.OpclassDefaults,
	}
}

func convertStoreTableMetadata(table *storepb.TableMetadata) *v1pb.TableMetadata {
	t := &v1pb.TableMetadata{
		Name:          table.Name,
		Engine:        table.Engine,
		Collation:     table.Collation,
		RowCount:      table.RowCount,
		DataSize:      table.DataSize,
		IndexSize:     table.IndexSize,
		DataFree:      table.DataFree,
		CreateOptions: table.CreateOptions,
		Comment:       table.Comment,
		Charset:       table.Charset,
		Owner:         table.Owner,
		SortingKeys:   table.SortingKeys,
		SkipDump:      table.SkipDump,
	}
	for _, partition := range table.Partitions {
		if partition == nil {
			continue
		}
		t.Partitions = append(t.Partitions, convertStoreTablePartitionMetadata(partition))
	}

	for _, column := range table.Columns {
		if column == nil {
			continue
		}
		t.Columns = append(t.Columns, convertStoreColumnMetadata(column))
	}
	for _, index := range table.Indexes {
		if index == nil {
			continue
		}
		t.Indexes = append(t.Indexes, convertStoreIndexMetadata(index))
	}
	for _, foreignKey := range table.ForeignKeys {
		if foreignKey == nil {
			continue
		}
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
	for _, check := range table.CheckConstraints {
		if check == nil {
			continue
		}
		t.CheckConstraints = append(t.CheckConstraints, &v1pb.CheckConstraintMetadata{
			Name:       check.Name,
			Expression: check.Expression,
		})
	}
	for _, trigger := range table.Triggers {
		if trigger == nil {
			continue
		}
		t.Triggers = append(t.Triggers, convertStoreTriggerMetadata(trigger))
	}
	return t
}

func convertStoreTriggerMetadata(trigger *storepb.TriggerMetadata) *v1pb.TriggerMetadata {
	return &v1pb.TriggerMetadata{
		Name:                trigger.Name,
		Timing:              trigger.Timing,
		Event:               trigger.Event,
		Body:                trigger.Body,
		SqlMode:             trigger.SqlMode,
		CharacterSetClient:  trigger.CharacterSetClient,
		CollationConnection: trigger.CollationConnection,
		Comment:             trigger.Comment,
		SkipDump:            trigger.SkipDump,
	}
}

func convertStoreExternalTableMetadata(externalTable *storepb.ExternalTableMetadata) *v1pb.ExternalTableMetadata {
	t := &v1pb.ExternalTableMetadata{
		Name:                 externalTable.Name,
		ExternalServerName:   externalTable.ExternalServerName,
		ExternalDatabaseName: externalTable.ExternalDatabaseName,
	}
	// Now we'd like to return column info for external table by default.
	for _, column := range externalTable.Columns {
		if column == nil {
			continue
		}
		t.Columns = append(t.Columns, convertStoreColumnMetadata(column))
	}
	return t
}

func convertStoreTablePartitionMetadata(partition *storepb.TablePartitionMetadata) *v1pb.TablePartitionMetadata {
	metadata := &v1pb.TablePartitionMetadata{
		Name:       partition.Name,
		Expression: partition.Expression,
		Value:      partition.Value,
		UseDefault: partition.UseDefault,
	}
	switch partition.Type {
	case storepb.TablePartitionMetadata_RANGE:
		metadata.Type = v1pb.TablePartitionMetadata_RANGE
	case storepb.TablePartitionMetadata_RANGE_COLUMNS:
		metadata.Type = v1pb.TablePartitionMetadata_RANGE_COLUMNS
	case storepb.TablePartitionMetadata_LIST:
		metadata.Type = v1pb.TablePartitionMetadata_LIST
	case storepb.TablePartitionMetadata_LIST_COLUMNS:
		metadata.Type = v1pb.TablePartitionMetadata_LIST_COLUMNS
	case storepb.TablePartitionMetadata_HASH:
		metadata.Type = v1pb.TablePartitionMetadata_HASH
	case storepb.TablePartitionMetadata_LINEAR_HASH:
		metadata.Type = v1pb.TablePartitionMetadata_LINEAR_HASH
	case storepb.TablePartitionMetadata_KEY:
		metadata.Type = v1pb.TablePartitionMetadata_KEY
	case storepb.TablePartitionMetadata_LINEAR_KEY:
		metadata.Type = v1pb.TablePartitionMetadata_LINEAR_KEY
	default:
		metadata.Type = v1pb.TablePartitionMetadata_TYPE_UNSPECIFIED
	}
	for _, index := range partition.Indexes {
		if index == nil {
			continue
		}
		metadata.Indexes = append(metadata.Indexes, convertStoreIndexMetadata(index))
	}
	for _, subpartition := range partition.Subpartitions {
		if subpartition == nil {
			continue
		}
		metadata.Subpartitions = append(metadata.Subpartitions, convertStoreTablePartitionMetadata(subpartition))
	}
	return metadata
}

func convertStoreColumnMetadata(column *storepb.ColumnMetadata) *v1pb.ColumnMetadata {
	metadata := &v1pb.ColumnMetadata{
		Name:                  column.Name,
		Position:              column.Position,
		HasDefault:            column.Default != "",
		DefaultOnNull:         column.DefaultOnNull,
		OnUpdate:              column.OnUpdate,
		Nullable:              column.Nullable,
		Type:                  column.Type,
		CharacterSet:          column.CharacterSet,
		Collation:             column.Collation,
		Comment:               column.Comment,
		Generation:            convertStoreGenerationMetadata(column.Generation),
		IdentityGeneration:    v1pb.ColumnMetadata_IdentityGeneration(column.IdentityGeneration),
		IsIdentity:            column.IsIdentity,
		IdentitySeed:          column.IdentitySeed,
		IdentityIncrement:     column.IdentityIncrement,
		DefaultConstraintName: column.DefaultConstraintName,
		Default:               column.Default,
	}
	switch column.IdentityGeneration {
	case storepb.ColumnMetadata_ALWAYS:
		metadata.IdentityGeneration = v1pb.ColumnMetadata_ALWAYS
	case storepb.ColumnMetadata_BY_DEFAULT:
		metadata.IdentityGeneration = v1pb.ColumnMetadata_BY_DEFAULT
	default:
		metadata.IdentityGeneration = v1pb.ColumnMetadata_IDENTITY_GENERATION_UNSPECIFIED
	}
	return metadata
}

func convertStoreGenerationMetadata(generation *storepb.GenerationMetadata) *v1pb.GenerationMetadata {
	if generation == nil {
		return nil
	}
	meta := &v1pb.GenerationMetadata{
		Expression: generation.Expression,
	}
	switch generation.Type {
	case storepb.GenerationMetadata_TYPE_VIRTUAL:
		meta.Type = v1pb.GenerationMetadata_VIRTUAL
	case storepb.GenerationMetadata_TYPE_STORED:
		meta.Type = v1pb.GenerationMetadata_STORED
	default:
		meta.Type = v1pb.GenerationMetadata_TYPE_UNSPECIFIED
	}
	return meta
}

func convertV1DatabaseMetadata(metadata *v1pb.DatabaseMetadata) *storepb.DatabaseSchemaMetadata {
	m := &storepb.DatabaseSchemaMetadata{
		Name:         metadata.Name,
		CharacterSet: metadata.CharacterSet,
		Collation:    metadata.Collation,
		Owner:        metadata.Owner,
	}
	for _, schema := range metadata.Schemas {
		if schema == nil {
			continue
		}
		s := &storepb.SchemaMetadata{
			Name:     schema.Name,
			Owner:    schema.Owner,
			Comment:  schema.Comment,
			SkipDump: schema.SkipDump,
		}
		for _, table := range schema.Tables {
			if table == nil {
				continue
			}
			s.Tables = append(s.Tables, convertV1TableMetadata(table))
		}
		for _, view := range schema.Views {
			if view == nil {
				continue
			}
			storeView := &storepb.ViewMetadata{
				Name:       view.Name,
				Definition: view.Definition,
				Comment:    view.Comment,
				SkipDump:   view.SkipDump,
			}

			for _, column := range view.Columns {
				if column == nil {
					continue
				}
				storeView.Columns = append(storeView.Columns, convertV1ColumnMetadata(column))
			}

			for _, dependencyColumn := range view.DependencyColumns {
				storeView.DependencyColumns = append(storeView.DependencyColumns,
					&storepb.DependencyColumn{
						Schema: dependencyColumn.Schema,
						Table:  dependencyColumn.Table,
						Column: dependencyColumn.Column,
					})
			}

			for _, trigger := range view.Triggers {
				if trigger == nil {
					continue
				}
				storeView.Triggers = append(storeView.Triggers, convertV1TriggerMetadata(trigger))
			}

			s.Views = append(s.Views, storeView)
		}
		for _, materializedView := range schema.MaterializedViews {
			if materializedView == nil {
				continue
			}
			storeMaterializedView := &storepb.MaterializedViewMetadata{
				Name:       materializedView.Name,
				Definition: materializedView.Definition,
				Comment:    materializedView.Comment,
				SkipDump:   materializedView.SkipDump,
			}
			for _, dependencyColumn := range materializedView.DependencyColumns {
				if dependencyColumn == nil {
					continue
				}
				storeMaterializedView.DependencyColumns = append(storeMaterializedView.DependencyColumns,
					&storepb.DependencyColumn{
						Schema: dependencyColumn.Schema,
						Table:  dependencyColumn.Table,
						Column: dependencyColumn.Column,
					})
			}

			for _, index := range materializedView.Indexes {
				if index == nil {
					continue
				}
				storeMaterializedView.Indexes = append(storeMaterializedView.Indexes, convertV1IndexMetadata(index))
			}

			for _, trigger := range materializedView.Triggers {
				if trigger == nil {
					continue
				}
				storeMaterializedView.Triggers = append(storeMaterializedView.Triggers, convertV1TriggerMetadata(trigger))
			}

			s.MaterializedViews = append(s.MaterializedViews, storeMaterializedView)
		}
		for _, function := range schema.Functions {
			if function == nil {
				continue
			}
			storeFunc := &storepb.FunctionMetadata{
				Name:                function.Name,
				Definition:          function.Definition,
				Signature:           function.Signature,
				CharacterSetClient:  function.CharacterSetClient,
				CollationConnection: function.CollationConnection,
				DatabaseCollation:   function.DatabaseCollation,
				SqlMode:             function.SqlMode,
				Comment:             function.Comment,
				SkipDump:            function.SkipDump,
			}
			for _, dep := range function.DependencyTables {
				storeFunc.DependencyTables = append(storeFunc.DependencyTables, &storepb.DependencyTable{
					Schema: dep.Schema,
					Table:  dep.Table,
				})
			}
			s.Functions = append(s.Functions, storeFunc)
		}
		for _, procedure := range schema.Procedures {
			if procedure == nil {
				continue
			}
			storeProcedure := &storepb.ProcedureMetadata{
				Name:                procedure.Name,
				Definition:          procedure.Definition,
				Signature:           procedure.Signature,
				CharacterSetClient:  procedure.CharacterSetClient,
				CollationConnection: procedure.CollationConnection,
				DatabaseCollation:   procedure.DatabaseCollation,
				SqlMode:             procedure.SqlMode,
				Comment:             procedure.Comment,
				SkipDump:            procedure.SkipDump,
			}
			s.Procedures = append(s.Procedures, storeProcedure)
		}
		for _, p := range schema.Packages {
			if p == nil {
				continue
			}
			storePackage := &storepb.PackageMetadata{
				Name:       p.Name,
				Definition: p.Definition,
			}
			s.Packages = append(s.Packages, storePackage)
		}
		for _, task := range schema.Tasks {
			if task == nil {
				continue
			}
			storeTask := &storepb.TaskMetadata{
				Name:         task.Name,
				Id:           task.Id,
				Owner:        task.Owner,
				Comment:      task.Comment,
				Warehouse:    task.Warehouse,
				Schedule:     task.Schedule,
				Predecessors: task.Predecessors,
				State:        storepb.TaskMetadata_State(task.State),
				Condition:    task.Condition,
				Definition:   task.Definition,
			}
			s.Tasks = append(s.Tasks, storeTask)
		}
		for _, stream := range schema.Streams {
			if stream == nil {
				continue
			}
			storeStream := &storepb.StreamMetadata{
				Name:       stream.Name,
				TableName:  stream.TableName,
				Owner:      stream.Owner,
				Comment:    stream.Comment,
				Type:       storepb.StreamMetadata_Type(stream.Type),
				Stale:      stream.Stale,
				Mode:       storepb.StreamMetadata_Mode(stream.Mode),
				Definition: stream.Definition,
			}
			s.Streams = append(s.Streams, storeStream)
		}
		for _, event := range schema.Events {
			if event == nil {
				continue
			}
			storeEvent := &storepb.EventMetadata{
				Name:                event.Name,
				TimeZone:            event.TimeZone,
				Definition:          event.Definition,
				SqlMode:             event.SqlMode,
				CharacterSetClient:  event.CharacterSetClient,
				CollationConnection: event.CollationConnection,
				Comment:             event.Comment,
			}
			s.Events = append(s.Events, storeEvent)
		}
		for _, enum := range schema.EnumTypes {
			if enum == nil {
				continue
			}
			storeEnum := &storepb.EnumTypeMetadata{
				Name:     enum.Name,
				Values:   enum.Values,
				Comment:  enum.Comment,
				SkipDump: enum.SkipDump,
			}
			s.EnumTypes = append(s.EnumTypes, storeEnum)
		}
		for _, sequence := range schema.Sequences {
			if sequence == nil {
				continue
			}
			storeSequence := &storepb.SequenceMetadata{
				Name:        sequence.Name,
				DataType:    sequence.DataType,
				Start:       sequence.Start,
				MinValue:    sequence.MinValue,
				MaxValue:    sequence.MaxValue,
				Increment:   sequence.Increment,
				Cycle:       sequence.Cycle,
				CacheSize:   sequence.CacheSize,
				LastValue:   sequence.LastValue,
				OwnerTable:  sequence.OwnerTable,
				OwnerColumn: sequence.OwnerColumn,
				Comment:     sequence.Comment,
				SkipDump:    sequence.SkipDump,
			}
			s.Sequences = append(s.Sequences, storeSequence)
		}
		m.Schemas = append(m.Schemas, s)
	}
	for _, extension := range metadata.Extensions {
		if extension == nil {
			continue
		}
		m.Extensions = append(m.Extensions, &storepb.ExtensionMetadata{
			Name:        extension.Name,
			Schema:      extension.Schema,
			Version:     extension.Version,
			Description: extension.Description,
		})
	}
	return m
}

func convertV1IndexMetadata(index *v1pb.IndexMetadata) *storepb.IndexMetadata {
	return &storepb.IndexMetadata{
		Name:              index.Name,
		Expressions:       index.Expressions,
		KeyLength:         index.KeyLength,
		Descending:        index.Descending,
		Type:              index.Type,
		Unique:            index.Unique,
		Primary:           index.Primary,
		Visible:           index.Visible,
		Comment:           index.Comment,
		Definition:        index.Definition,
		ParentIndexSchema: index.ParentIndexSchema,
		ParentIndexName:   index.ParentIndexName,
		Granularity:       index.Granularity,
		IsConstraint:      index.IsConstraint,
		SpatialConfig:     convertV1SpatialIndexConfig(index.SpatialConfig),
	}
}

func convertStoreSpatialIndexConfig(spatial *storepb.SpatialIndexConfig) *v1pb.SpatialIndexConfig {
	if spatial == nil {
		return nil
	}
	return &v1pb.SpatialIndexConfig{
		Method:       spatial.Method,
		Tessellation: convertStoreTessellationConfig(spatial.Tessellation),
		Storage:      convertStoreStorageConfig(spatial.Storage),
		Dimensional:  convertStoreDimensionalConfig(spatial.Dimensional),
	}
}

func convertV1SpatialIndexConfig(spatial *v1pb.SpatialIndexConfig) *storepb.SpatialIndexConfig {
	if spatial == nil {
		return nil
	}
	return &storepb.SpatialIndexConfig{
		Method:       spatial.Method,
		Tessellation: convertV1TessellationConfig(spatial.Tessellation),
		Storage:      convertV1StorageConfig(spatial.Storage),
		Dimensional:  convertV1DimensionalConfig(spatial.Dimensional),
	}
}

func convertStoreTessellationConfig(tessellation *storepb.TessellationConfig) *v1pb.TessellationConfig {
	if tessellation == nil {
		return nil
	}
	gridLevels := make([]*v1pb.GridLevel, len(tessellation.GridLevels))
	for i, level := range tessellation.GridLevels {
		gridLevels[i] = &v1pb.GridLevel{
			Level:   level.Level,
			Density: level.Density,
		}
	}
	return &v1pb.TessellationConfig{
		Scheme:         tessellation.Scheme,
		GridLevels:     gridLevels,
		CellsPerObject: tessellation.CellsPerObject,
		BoundingBox:    convertStoreBoundingBox(tessellation.BoundingBox),
	}
}

func convertV1TessellationConfig(tessellation *v1pb.TessellationConfig) *storepb.TessellationConfig {
	if tessellation == nil {
		return nil
	}
	gridLevels := make([]*storepb.GridLevel, len(tessellation.GridLevels))
	for i, level := range tessellation.GridLevels {
		gridLevels[i] = &storepb.GridLevel{
			Level:   level.Level,
			Density: level.Density,
		}
	}
	return &storepb.TessellationConfig{
		Scheme:         tessellation.Scheme,
		GridLevels:     gridLevels,
		CellsPerObject: tessellation.CellsPerObject,
		BoundingBox:    convertV1BoundingBox(tessellation.BoundingBox),
	}
}

func convertStoreBoundingBox(bbox *storepb.BoundingBox) *v1pb.BoundingBox {
	if bbox == nil {
		return nil
	}
	return &v1pb.BoundingBox{
		Xmin: bbox.Xmin,
		Ymin: bbox.Ymin,
		Xmax: bbox.Xmax,
		Ymax: bbox.Ymax,
	}
}

func convertV1BoundingBox(bbox *v1pb.BoundingBox) *storepb.BoundingBox {
	if bbox == nil {
		return nil
	}
	return &storepb.BoundingBox{
		Xmin: bbox.Xmin,
		Ymin: bbox.Ymin,
		Xmax: bbox.Xmax,
		Ymax: bbox.Ymax,
	}
}

func convertStoreStorageConfig(storage *storepb.StorageConfig) *v1pb.StorageConfig {
	if storage == nil {
		return nil
	}
	return &v1pb.StorageConfig{
		Fillfactor:      storage.Fillfactor,
		Buffering:       storage.Buffering,
		Tablespace:      storage.Tablespace,
		WorkTablespace:  storage.WorkTablespace,
		SdoLevel:        storage.SdoLevel,
		CommitInterval:  storage.CommitInterval,
		PadIndex:        storage.PadIndex,
		SortInTempdb:    storage.SortInTempdb,
		DropExisting:    storage.DropExisting,
		Online:          storage.Online,
		AllowRowLocks:   storage.AllowRowLocks,
		AllowPageLocks:  storage.AllowPageLocks,
		Maxdop:          storage.Maxdop,
		DataCompression: storage.DataCompression,
	}
}

func convertV1StorageConfig(storage *v1pb.StorageConfig) *storepb.StorageConfig {
	if storage == nil {
		return nil
	}
	return &storepb.StorageConfig{
		Fillfactor:      storage.Fillfactor,
		Buffering:       storage.Buffering,
		Tablespace:      storage.Tablespace,
		WorkTablespace:  storage.WorkTablespace,
		SdoLevel:        storage.SdoLevel,
		CommitInterval:  storage.CommitInterval,
		PadIndex:        storage.PadIndex,
		SortInTempdb:    storage.SortInTempdb,
		DropExisting:    storage.DropExisting,
		Online:          storage.Online,
		AllowRowLocks:   storage.AllowRowLocks,
		AllowPageLocks:  storage.AllowPageLocks,
		Maxdop:          storage.Maxdop,
		DataCompression: storage.DataCompression,
	}
}

func convertStoreDimensionalConfig(dimensional *storepb.DimensionalConfig) *v1pb.DimensionalConfig {
	if dimensional == nil {
		return nil
	}
	return &v1pb.DimensionalConfig{
		Dimensions: dimensional.Dimensions,
		DataType:   dimensional.DataType,
		// Note: Srid and Constraints fields exist only in v1 proto
		// Store proto uses OperatorClass instead
	}
}

func convertV1DimensionalConfig(dimensional *v1pb.DimensionalConfig) *storepb.DimensionalConfig {
	if dimensional == nil {
		return nil
	}
	return &storepb.DimensionalConfig{
		Dimensions: dimensional.Dimensions,
		DataType:   dimensional.DataType,
		// Note: OperatorClass field exists only in store proto
		// V1 proto uses Srid and Constraints instead
	}
}

func convertV1TableMetadata(table *v1pb.TableMetadata) *storepb.TableMetadata {
	t := &storepb.TableMetadata{
		Name:          table.Name,
		Engine:        table.Engine,
		Collation:     table.Collation,
		RowCount:      table.RowCount,
		DataSize:      table.DataSize,
		IndexSize:     table.IndexSize,
		DataFree:      table.DataFree,
		CreateOptions: table.CreateOptions,
		Comment:       table.Comment,
		Charset:       table.Charset,
		Owner:         table.Owner,
		SortingKeys:   table.SortingKeys,
		SkipDump:      table.SkipDump,
	}
	for _, column := range table.Columns {
		if column == nil {
			continue
		}
		t.Columns = append(t.Columns, convertV1ColumnMetadata(column))
	}
	for _, index := range table.Indexes {
		if index == nil {
			continue
		}
		t.Indexes = append(t.Indexes, convertV1IndexMetadata(index))
	}
	for _, foreignKey := range table.ForeignKeys {
		if foreignKey == nil {
			continue
		}
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
	for _, partition := range table.Partitions {
		if partition == nil {
			continue
		}
		t.Partitions = append(t.Partitions, convertV1TablePartitionMetadata(partition))
	}
	for _, check := range table.CheckConstraints {
		if check == nil {
			continue
		}
		t.CheckConstraints = append(t.CheckConstraints, &storepb.CheckConstraintMetadata{
			Name:       check.Name,
			Expression: check.Expression,
		})
	}
	for _, trigger := range table.Triggers {
		if trigger == nil {
			continue
		}
		t.Triggers = append(t.Triggers, convertV1TriggerMetadata(trigger))
	}
	return t
}

func convertV1TriggerMetadata(trigger *v1pb.TriggerMetadata) *storepb.TriggerMetadata {
	return &storepb.TriggerMetadata{
		Name:                trigger.Name,
		Timing:              trigger.Timing,
		Event:               trigger.Event,
		Body:                trigger.Body,
		SqlMode:             trigger.SqlMode,
		CharacterSetClient:  trigger.CharacterSetClient,
		CollationConnection: trigger.CollationConnection,
		Comment:             trigger.Comment,
		SkipDump:            trigger.SkipDump,
	}
}

func convertV1TablePartitionMetadata(tablePartition *v1pb.TablePartitionMetadata) *storepb.TablePartitionMetadata {
	metadata := &storepb.TablePartitionMetadata{
		Name:       tablePartition.Name,
		Expression: tablePartition.Expression,
		Value:      tablePartition.Value,
		UseDefault: tablePartition.UseDefault,
	}
	switch tablePartition.Type {
	case v1pb.TablePartitionMetadata_RANGE:
		metadata.Type = storepb.TablePartitionMetadata_RANGE
	case v1pb.TablePartitionMetadata_RANGE_COLUMNS:
		metadata.Type = storepb.TablePartitionMetadata_RANGE_COLUMNS
	case v1pb.TablePartitionMetadata_LIST:
		metadata.Type = storepb.TablePartitionMetadata_LIST
	case v1pb.TablePartitionMetadata_LIST_COLUMNS:
		metadata.Type = storepb.TablePartitionMetadata_LIST_COLUMNS
	case v1pb.TablePartitionMetadata_HASH:
		metadata.Type = storepb.TablePartitionMetadata_HASH
	case v1pb.TablePartitionMetadata_LINEAR_HASH:
		metadata.Type = storepb.TablePartitionMetadata_LINEAR_HASH
	case v1pb.TablePartitionMetadata_KEY:
		metadata.Type = storepb.TablePartitionMetadata_KEY
	case v1pb.TablePartitionMetadata_LINEAR_KEY:
		metadata.Type = storepb.TablePartitionMetadata_LINEAR_KEY
	default:
		metadata.Type = storepb.TablePartitionMetadata_TYPE_UNSPECIFIED
	}
	for _, index := range tablePartition.Indexes {
		if index == nil {
			continue
		}
		metadata.Indexes = append(metadata.Indexes, convertV1IndexMetadata(index))
	}
	for _, subpartition := range tablePartition.Subpartitions {
		if subpartition == nil {
			continue
		}
		metadata.Subpartitions = append(metadata.Subpartitions, convertV1TablePartitionMetadata(subpartition))
	}
	return metadata
}

func convertV1ColumnMetadata(column *v1pb.ColumnMetadata) *storepb.ColumnMetadata {
	metadata := &storepb.ColumnMetadata{
		Name:                  column.Name,
		Position:              column.Position,
		Nullable:              column.Nullable,
		DefaultOnNull:         column.DefaultOnNull,
		Type:                  column.Type,
		CharacterSet:          column.CharacterSet,
		Collation:             column.Collation,
		Comment:               column.Comment,
		OnUpdate:              column.OnUpdate,
		Generation:            convertV1GenerationMetadata(column.Generation),
		IdentityGeneration:    storepb.ColumnMetadata_IdentityGeneration(column.IdentityGeneration),
		IsIdentity:            column.IsIdentity,
		IdentitySeed:          column.IdentitySeed,
		IdentityIncrement:     column.IdentityIncrement,
		DefaultConstraintName: column.DefaultConstraintName,
		Default:               column.Default,
	}

	switch column.IdentityGeneration {
	case v1pb.ColumnMetadata_ALWAYS:
		metadata.IdentityGeneration = storepb.ColumnMetadata_ALWAYS
	case v1pb.ColumnMetadata_BY_DEFAULT:
		metadata.IdentityGeneration = storepb.ColumnMetadata_BY_DEFAULT
	default:
		metadata.IdentityGeneration = storepb.ColumnMetadata_IDENTITY_GENERATION_UNSPECIFIED
	}
	return metadata
}

func convertV1GenerationMetadata(generation *v1pb.GenerationMetadata) *storepb.GenerationMetadata {
	if generation == nil {
		return nil
	}
	meta := &storepb.GenerationMetadata{
		Expression: generation.Expression,
	}
	switch generation.Type {
	case v1pb.GenerationMetadata_VIRTUAL:
		meta.Type = storepb.GenerationMetadata_TYPE_VIRTUAL
	case v1pb.GenerationMetadata_STORED:
		meta.Type = storepb.GenerationMetadata_TYPE_STORED
	default:
		meta.Type = storepb.GenerationMetadata_TYPE_UNSPECIFIED
	}
	return meta
}
