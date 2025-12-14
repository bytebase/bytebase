import { cloneDeep, isEqual, pick } from "lodash-es";
import type { ComposedDatabase } from "@/types";
import type {
  FunctionMetadata,
  ProcedureMetadata,
  TablePartitionMetadata,
  ViewMetadata,
} from "@/types/proto-es/v1/database_service_pb";
import {
  type ColumnMetadata,
  type DatabaseMetadata,
  type ForeignKeyMetadata,
  type IndexMetadata,
  type SchemaMetadata,
  type TableMetadata,
} from "@/types/proto-es/v1/database_service_pb";
import {
  ComparableColumnFields,
  ComparableForeignKeyFields,
  ComparableIndexFields,
  ComparableTableFields,
  ComparableTablePartitionFields,
  keyBy,
  TinyTimer,
} from "@/utils";
import type { SchemaEditorContext } from "../context";
import { keyForResourceName } from "../context/common";

type RichSchemaMetadata = {
  database: DatabaseMetadata;
  schema: SchemaMetadata;
};
type RichTableMetadata = RichSchemaMetadata & {
  table: TableMetadata;
};
type RichColumnMetadata = RichTableMetadata & {
  column: ColumnMetadata;
};

export class DiffMerge {
  context: SchemaEditorContext;
  database: ComposedDatabase;
  sourceMetadata: DatabaseMetadata;
  targetMetadata: DatabaseMetadata;
  sourceSchemaMap = new Map<string, SchemaMetadata>();
  targetSchemaMap = new Map<string, SchemaMetadata>();
  sourceTableMap = new Map<string, TableMetadata>();
  targetTableMap = new Map<string, TableMetadata>();
  sourceColumnMap = new Map<string, ColumnMetadata>();
  targetColumnMap = new Map<string, ColumnMetadata>();
  sourceViewMap = new Map<string, ViewMetadata>();
  targetViewMap = new Map<string, ViewMetadata>();
  sourceProcedureMap = new Map<string, ProcedureMetadata>();
  targetProcedureMap = new Map<string, ProcedureMetadata>();
  sourceFunctionMap = new Map<string, FunctionMetadata>();
  targetFunctionMap = new Map<string, FunctionMetadata>();
  timer = new TinyTimer<
    | "merge"
    | "mergeSchemas"
    | "mergeTables"
    | "mergeColumns"
    | "mergeTablePartitions"
    | "diffColumn"
    | "mergeViews"
    | "mergeProcedures"
    | "mergeFunctions"
  >("DiffMerge");
  constructor({
    context,
    database,
    sourceMetadata,
    targetMetadata,
  }: {
    context: SchemaEditorContext;
    database: ComposedDatabase;
    sourceMetadata: DatabaseMetadata;
    targetMetadata: DatabaseMetadata;
  }) {
    this.context = context;
    this.database = database;
    this.sourceMetadata = sourceMetadata;
    this.targetMetadata = targetMetadata;
  }
  merge() {
    this.timer.begin("merge");
    this.mergeSchemas();
    this.timer.end("merge");
  }
  mergeSchemas() {
    const {
      context,
      database,
      sourceMetadata,
      targetMetadata,
      sourceSchemaMap,
      targetSchemaMap,
    } = this;
    const sourceSchemas = sourceMetadata.schemas;
    const targetSchemas = targetMetadata.schemas;
    this.timer.begin("mergeSchemas");
    mapSchemas(database, sourceSchemas, sourceSchemaMap);
    mapSchemas(database, targetSchemas, targetSchemaMap);

    const mergedSchemas: SchemaMetadata[] = [];
    for (let i = 0; i < sourceSchemas.length; i++) {
      const sourceSchema = sourceSchemas[i];
      const key = keyForResourceName({
        database: database.name,
        schema: sourceSchema.name,
      });
      let targetSchema = targetSchemaMap.get(key);
      if (targetSchema) {
        // existed schema
        mergedSchemas.push(targetSchema);
        // merge tables for existed (maybe updated) schema
        this.mergeTables(
          {
            database: sourceMetadata,
            schema: sourceSchema,
          },
          {
            database: targetMetadata,
            schema: targetSchema,
          }
        );
        this.mergeViews(
          {
            database: sourceMetadata,
            schema: sourceSchema,
          },
          {
            database: targetMetadata,
            schema: targetSchema,
          }
        );
        this.mergeProcedures(
          {
            database: sourceMetadata,
            schema: sourceSchema,
          },
          {
            database: targetMetadata,
            schema: targetSchema,
          }
        );
        this.mergeFunctions(
          {
            database: sourceMetadata,
            schema: sourceSchema,
          },
          {
            database: targetMetadata,
            schema: targetSchema,
          }
        );

        continue;
      }
      // dropped schema
      // copy the source schema to target and mark it as 'dropped'
      targetSchema = cloneDeep(sourceSchema);
      mergedSchemas.push(targetSchema);
      context.markEditStatusByKey(key, "dropped");
    }
    for (let i = 0; i < targetSchemas.length; i++) {
      const targetSchema = targetSchemas[i];
      const key = keyForResourceName({
        database: database.name,
        schema: targetSchema.name,
      });
      const sourceSchema = sourceSchemaMap.get(key);
      if (!sourceSchema) {
        // newly created schema
        // mark it as 'created'
        mergedSchemas.push(targetSchema);
        context.markEditStatusByKey(key, "created");
      }
    }
    targetMetadata.schemas = mergedSchemas;
    this.timer.end("mergeSchemas", sourceSchemas.length + targetSchemas.length);
  }
  mergeTables(source: RichSchemaMetadata, target: RichSchemaMetadata) {
    const { context, database, sourceTableMap, targetTableMap } = this;
    const { schema: sourceSchema } = source;
    const { schema: targetSchema } = target;
    const sourceTables = sourceSchema.tables;
    const targetTables = targetSchema.tables;
    this.timer.begin("mergeTables");
    mapTables(database, sourceSchema, sourceTables, sourceTableMap);
    mapTables(database, targetSchema, targetTables, targetTableMap);

    const mergedTables: TableMetadata[] = [];
    for (let i = 0; i < sourceTables.length; i++) {
      const sourceTable = sourceTables[i];
      const key = keyForResourceName({
        database: database.name,
        schema: sourceSchema.name,
        table: sourceTable.name,
      });
      let targetTable = targetTableMap.get(key);
      if (targetTable) {
        // existed table
        mergedTables.push(targetTable);
        // merge columns for existed (maybe updated) table
        this.mergeColumns(
          { ...source, table: sourceTable },
          { ...target, table: targetTable }
        );
        this.mergeTablePartitions(
          { ...source, table: sourceTable },
          { ...target, table: targetTable }
        );

        if (
          !isEqual(
            pick(sourceTable, ComparableTableFields),
            pick(targetTable, ComparableTableFields)
          ) ||
          !this.isEqualForeignKeys(
            sourceTable.foreignKeys,
            targetTable.foreignKeys
          ) ||
          !this.isEqualIndexes(sourceTable.indexes, targetTable.indexes)
        ) {
          // Index and foreignKey changes are considered as table updating by now
          // for simplification
          context.markEditStatusByKey(key, "updated");
        }

        continue;
      }
      // dropped table
      // copy the source table to target and mark it as 'dropped'
      targetTable = cloneDeep(sourceTable);
      mergedTables.push(targetTable);
      context.markEditStatusByKey(key, "dropped");
    }
    for (let i = 0; i < targetTables.length; i++) {
      const targetTable = targetTables[i];
      const key = keyForResourceName({
        database: database.name,
        schema: targetSchema.name,
        table: targetTable.name,
      });
      const sourceTable = sourceTableMap.get(key);
      if (!sourceTable) {
        // newly created table
        // mark it as 'created'
        mergedTables.push(targetTable);
        context.markEditStatusByKey(key, "created");

        for (const column of targetTable.columns) {
          context.markEditStatus(
            database,
            {
              schema: targetSchema,
              table: targetTable,
              column,
            },
            "created"
          );
        }
      }
    }
    targetSchema.tables = mergedTables;
    this.timer.end("mergeTables", sourceTables.length + targetTables.length);
  }
  isEqualForeignKeys(
    sourceForeignKeys: ForeignKeyMetadata[],
    targetForeignKeys: ForeignKeyMetadata[]
  ) {
    if (sourceForeignKeys.length !== targetForeignKeys.length) {
      return false;
    }
    for (let i = 0; i < sourceForeignKeys.length; i++) {
      const sourceForeignKey = sourceForeignKeys[i];
      const targetForeignKey = targetForeignKeys[i];
      if (
        !isEqual(
          pick(sourceForeignKey, ComparableForeignKeyFields),
          pick(targetForeignKey, ComparableForeignKeyFields)
        )
      ) {
        return false;
      }
    }
    return true;
  }
  isEqualIndexes(
    sourceIndexes: IndexMetadata[],
    targetIndexes: IndexMetadata[]
  ) {
    if (sourceIndexes.length !== targetIndexes.length) {
      return false;
    }
    const targetIndexesByName = keyBy(targetIndexes, (idx) => idx.name);

    for (let i = 0; i < sourceIndexes.length; i++) {
      const sourceIndex = sourceIndexes[i];
      const targetIndex = targetIndexesByName.get(sourceIndex.name);
      // targetIndex not found
      if (!targetIndex) return false;
      if (
        !isEqual(
          pick(sourceIndex, ComparableIndexFields),
          pick(targetIndex, ComparableIndexFields)
        )
      ) {
        return false;
      }
    }
    return true;
  }
  isEqualTablePartitions(
    sourcePartitions: TablePartitionMetadata[],
    targetPartitions: TablePartitionMetadata[]
  ) {
    if (sourcePartitions.length !== targetPartitions.length) {
      return false;
    }
    const targetPartitionsByName = keyBy(targetPartitions, (part) => part.name);

    for (let i = 0; i < sourcePartitions.length; i++) {
      const sourcePartition = sourcePartitions[i];
      const targetPartition = targetPartitionsByName.get(sourcePartition.name);
      // targetPartition not found
      if (!targetPartition) return false;
      if (
        !isEqual(
          pick(sourcePartition, ComparableTablePartitionFields),
          pick(targetPartition, ComparableTablePartitionFields)
        )
      ) {
        return false;
      }
      if (
        !this.isEqualTablePartitions(
          sourcePartition.subpartitions ?? [],
          targetPartition.subpartitions ?? []
        )
      ) {
        return false;
      }
    }
    return true;
  }
  mergeTablePartitions(source: RichTableMetadata, target: RichTableMetadata) {
    const { context, database } = this;
    const { schema: sourceSchema, table: sourceTable } = source;
    const { schema: targetSchema, table: targetTable } = target;

    const doMergePartitions = (
      sourcePartitions: TablePartitionMetadata[],
      targetPartitions: TablePartitionMetadata[]
    ) => {
      const sourcePartitionMap = new Map<string, TablePartitionMetadata>();
      const targetPartitionMap = new Map<string, TablePartitionMetadata>();

      mapTablePartitions(
        database,
        sourceSchema,
        sourceTable,
        sourcePartitions,
        sourcePartitionMap
      );
      mapTablePartitions(
        database,
        targetSchema,
        targetTable,
        targetPartitions,
        targetPartitionMap
      );

      const mergedPartitions: TablePartitionMetadata[] = [];
      for (let i = 0; i < sourcePartitions.length; i++) {
        const sourcePartition = sourcePartitions[i];
        const key = keyForResourceName({
          database: database.name,
          schema: sourceSchema.name,
          table: sourceTable.name,
          partition: sourcePartition.name,
        });
        let targetPartition = targetPartitionMap.get(key);
        if (targetPartition) {
          // existed partition
          mergedPartitions.push(targetPartition);
          targetPartition.subpartitions = doMergePartitions(
            sourcePartition.subpartitions ?? [],
            targetPartition.subpartitions ?? []
          );
          continue;
        }
        // dropped partition
        // copy the source partition to target and mark it as 'dropped'
        targetPartition = cloneDeep(sourcePartition);
        mergedPartitions.push(targetPartition);
        context.markEditStatusByKey(key, "dropped");
      }
      for (let i = 0; i < targetPartitions.length; i++) {
        const targetPartition = targetPartitions[i];
        const key = keyForResourceName({
          database: database.name,
          schema: targetSchema.name,
          table: targetTable.name,
          partition: targetPartition.name,
        });
        const sourcePartition = sourcePartitionMap.get(key);
        if (!sourcePartition) {
          // newly created partition
          // mark it as 'created'
          mergedPartitions.push(targetPartition);
          context.markEditStatusByKey(key, "created");

          // Then mark all its subpartitions as 'created', by performing
          // a diff from empty array to targetPartition.subpartitions
          doMergePartitions(
            /* sourcePartitions */ [],
            targetPartition.subpartitions
          );
        }
      }
      return mergedPartitions;
    };

    this.timer.begin("mergeTablePartitions");
    targetTable.partitions = doMergePartitions(
      sourceTable.partitions,
      targetTable.partitions
    );
    this.timer.end(
      "mergeTablePartitions",
      sourceTable.partitions.length + targetTable.partitions.length
    );
  }
  mergeColumns(source: RichTableMetadata, target: RichTableMetadata) {
    const { context, database, sourceColumnMap, targetColumnMap } = this;
    const { schema: sourceSchema, table: sourceTable } = source;
    const { schema: targetSchema, table: targetTable } = target;
    const sourceColumns = sourceTable.columns;
    const targetColumns = targetTable.columns;
    this.timer.begin("mergeColumns");
    mapColumns(
      database,
      sourceSchema,
      sourceTable,
      sourceColumns,
      sourceColumnMap
    );
    mapColumns(
      database,
      targetSchema,
      targetTable,
      targetColumns,
      targetColumnMap
    );

    const mergedColumns: ColumnMetadata[] = [];
    for (const sourceColumn of sourceColumns) {
      const key = keyForResourceName({
        database: database.name,
        schema: sourceSchema.name,
        table: sourceTable.name,
        column: sourceColumn.name,
      });
      let targetColumn = targetColumnMap.get(key);
      if (targetColumn) {
        // existed column
        mergedColumns.push(targetColumn);
        this.diffColumn(
          { ...source, column: sourceColumn },
          { ...target, column: targetColumn }
        );
        continue;
      }
      // dropped column
      // copy the source column to target and mark it as 'dropped'
      targetColumn = cloneDeep(sourceColumn);
      mergedColumns.push(targetColumn);
      context.markEditStatusByKey(key, "dropped");
    }
    for (const targetColumn of targetColumns) {
      const key = keyForResourceName({
        database: database.name,
        schema: targetSchema.name,
        table: targetTable.name,
        column: targetColumn.name,
      });
      const sourceColumn = sourceColumnMap.get(key);
      if (!sourceColumn) {
        // newly created column
        // mark it as 'created'
        mergedColumns.push(targetColumn);
        context.markEditStatusByKey(key, "created");
      }
    }
    targetTable.columns = mergedColumns;
    this.timer.end("mergeColumns", sourceColumns.length + targetColumns.length);
  }
  diffColumn(source: RichColumnMetadata, target: RichColumnMetadata) {
    const { column: sourceColumn } = source;
    const {
      schema: targetSchema,
      table: targetTable,
      column: targetColumn,
    } = target;
    this.timer.begin("diffColumn");

    if (
      !isEqual(
        pick(sourceColumn, ComparableColumnFields),
        pick(targetColumn, ComparableColumnFields)
      )
    ) {
      const key = keyForResourceName({
        database: this.database.name,
        schema: targetSchema.name,
        table: targetTable.name,
        column: targetColumn.name,
      });
      this.context.markEditStatusByKey(key, "updated");
    }
    this.timer.end("diffColumn", 1);
  }
  mergeViews(source: RichSchemaMetadata, target: RichSchemaMetadata) {
    const { context, database, sourceViewMap, targetViewMap } = this;
    const { schema: sourceSchema } = source;
    const { schema: targetSchema } = target;
    const sourceViews = sourceSchema.views;
    const targetViews = targetSchema.views;
    this.timer.begin("mergeViews");
    mapViews(database, sourceSchema, sourceViews, sourceViewMap);
    mapViews(database, targetSchema, targetViews, targetViewMap);

    const mergedViews: ViewMetadata[] = [];
    for (let i = 0; i < sourceViews.length; i++) {
      const sourceView = sourceViews[i];
      const key = keyForResourceName({
        database: database.name,
        schema: sourceSchema.name,
        view: sourceView.name,
      });
      let targetView = targetViewMap.get(key);
      if (targetView) {
        // existed view
        mergedViews.push(targetView);
        if (sourceView.definition !== targetView.definition) {
          context.markEditStatusByKey(key, "updated");
        }
        continue;
      }
      // dropped view
      // copy the source view to target and mark it as 'dropped'
      targetView = cloneDeep(sourceView);
      mergedViews.push(targetView);
      context.markEditStatusByKey(key, "dropped");
    }
    for (let i = 0; i < targetViews.length; i++) {
      const targetView = targetViews[i];
      const key = keyForResourceName({
        database: database.name,
        schema: targetSchema.name,
        view: targetView.name,
      });
      const sourceView = sourceViewMap.get(key);
      if (!sourceView) {
        // newly created view
        // mark it as 'created'
        mergedViews.push(targetView);
        context.markEditStatusByKey(key, "created");
      }
    }
    targetSchema.views = mergedViews;
    this.timer.end("mergeViews", sourceViews.length + targetViews.length);
  }
  mergeProcedures(source: RichSchemaMetadata, target: RichSchemaMetadata) {
    const { context, database, sourceProcedureMap, targetProcedureMap } = this;
    const { schema: sourceSchema } = source;
    const { schema: targetSchema } = target;
    const sourceProcedures = sourceSchema.procedures;
    const targetProcedures = targetSchema.procedures;
    this.timer.begin("mergeProcedures");
    mapProcedures(database, sourceSchema, sourceProcedures, sourceProcedureMap);
    mapProcedures(database, targetSchema, targetProcedures, targetProcedureMap);

    const mergedProcedures: ProcedureMetadata[] = [];
    for (let i = 0; i < sourceProcedures.length; i++) {
      const sourceProcedure = sourceProcedures[i];
      const key = keyForResourceName({
        database: database.name,
        schema: sourceSchema.name,
        procedure: sourceProcedure.name,
      });
      let targetProcedure = targetProcedureMap.get(key);
      if (targetProcedure) {
        // existed procedure
        mergedProcedures.push(targetProcedure);
        if (sourceProcedure.definition !== targetProcedure.definition) {
          context.markEditStatusByKey(key, "updated");
        }
        continue;
      }
      // dropped procedure
      // copy the source procedure to target and mark it as 'dropped'
      targetProcedure = cloneDeep(sourceProcedure);
      mergedProcedures.push(targetProcedure);
      context.markEditStatusByKey(key, "dropped");
    }
    for (let i = 0; i < targetProcedures.length; i++) {
      const targetProcedure = targetProcedures[i];
      const key = keyForResourceName({
        database: database.name,
        schema: targetSchema.name,
        procedure: targetProcedure.name,
      });
      const sourceProcedure = sourceProcedureMap.get(key);
      if (!sourceProcedure) {
        // newly created procedure
        // mark it as 'created'
        mergedProcedures.push(targetProcedure);
        context.markEditStatusByKey(key, "created");
      }
    }
    targetSchema.procedures = mergedProcedures;
    this.timer.end(
      "mergeProcedures",
      sourceProcedures.length + targetProcedures.length
    );
  }
  mergeFunctions(source: RichSchemaMetadata, target: RichSchemaMetadata) {
    const { context, database, sourceFunctionMap, targetFunctionMap } = this;
    const { schema: sourceSchema } = source;
    const { schema: targetSchema } = target;
    const sourceFunctions = sourceSchema.functions;
    const targetFunctions = targetSchema.functions;
    this.timer.begin("mergeFunctions");
    mapFunctions(database, sourceSchema, sourceFunctions, sourceFunctionMap);
    mapFunctions(database, targetSchema, targetFunctions, targetFunctionMap);

    const mergedFunctions: FunctionMetadata[] = [];
    for (let i = 0; i < sourceFunctions.length; i++) {
      const sourceFunction = sourceFunctions[i];
      const key = keyForResourceName({
        database: database.name,
        schema: sourceSchema.name,
        function: sourceFunction.name,
      });
      let targetFunction = targetFunctionMap.get(key);
      if (targetFunction) {
        // existed function
        mergedFunctions.push(targetFunction);
        if (sourceFunction.definition !== targetFunction.definition) {
          context.markEditStatusByKey(key, "updated");
        }
        continue;
      }
      // dropped function
      // copy the source function to target and mark it as 'dropped'
      targetFunction = cloneDeep(sourceFunction);
      mergedFunctions.push(targetFunction);
      context.markEditStatusByKey(key, "dropped");
    }
    for (let i = 0; i < targetFunctions.length; i++) {
      const targetFunction = targetFunctions[i];
      const key = keyForResourceName({
        database: database.name,
        schema: targetSchema.name,
        function: targetFunction.name,
      });
      const sourceFunction = sourceFunctionMap.get(key);
      if (!sourceFunction) {
        // newly created function
        // mark it as 'created'
        mergedFunctions.push(targetFunction);
        context.markEditStatusByKey(key, "created");
      }
    }
    targetSchema.functions = mergedFunctions;
    this.timer.end(
      "mergeFunctions",
      sourceFunctions.length + targetFunctions.length
    );
  }
}

// database schema
const mapSchemas = (
  database: ComposedDatabase,
  schemas: SchemaMetadata[],
  map: Map<string, SchemaMetadata>
) => {
  schemas.forEach((schema) => {
    const key = keyForResourceName({
      database: database.name,
      schema: schema.name,
    });
    map.set(key, schema);
  });
};
const mapTables = (
  database: ComposedDatabase,
  schema: SchemaMetadata,
  tables: TableMetadata[],
  map: Map<string, TableMetadata>
) => {
  tables.forEach((table) => {
    const key = keyForResourceName({
      database: database.name,
      schema: schema.name,
      table: table.name,
    });
    map.set(key, table);
  });
};
const mapColumns = (
  database: ComposedDatabase,
  schema: SchemaMetadata,
  table: TableMetadata,
  columns: ColumnMetadata[],
  map: Map<string, ColumnMetadata>
) => {
  columns.forEach((column) => {
    const key = keyForResourceName({
      database: database.name,
      schema: schema.name,
      table: table.name,
      column: column.name,
    });
    map.set(key, column);
  });
};
const mapTablePartitions = (
  database: ComposedDatabase,
  schema: SchemaMetadata,
  table: TableMetadata,
  partitions: TablePartitionMetadata[],
  map: Map<string, TablePartitionMetadata>
) => {
  partitions.forEach((partition) => {
    const key = keyForResourceName({
      database: database.name,
      schema: schema.name,
      table: table.name,
      partition: partition.name,
    });
    map.set(key, partition);
  });
};
const mapViews = (
  database: ComposedDatabase,
  schema: SchemaMetadata,
  views: ViewMetadata[],
  map: Map<string, ViewMetadata>
) => {
  views.forEach((view) => {
    const key = keyForResourceName({
      database: database.name,
      schema: schema.name,
      view: view.name,
    });
    map.set(key, view);
  });
};
const mapProcedures = (
  database: ComposedDatabase,
  schema: SchemaMetadata,
  procedures: ProcedureMetadata[],
  map: Map<string, ProcedureMetadata>
) => {
  procedures.forEach((procedure) => {
    const key = keyForResourceName({
      database: database.name,
      schema: schema.name,
      procedure: procedure.name,
    });
    map.set(key, procedure);
  });
};
const mapFunctions = (
  database: ComposedDatabase,
  schema: SchemaMetadata,
  functions: FunctionMetadata[],
  map: Map<string, FunctionMetadata>
) => {
  functions.forEach((fn) => {
    const key = keyForResourceName({
      database: database.name,
      schema: schema.name,
      function: fn.name,
    });
    map.set(key, fn);
  });
};
