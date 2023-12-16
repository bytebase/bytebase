import { cloneDeep, isEqual } from "lodash-es";
import { ComposedDatabase } from "@/types";
import {
  ColumnConfig,
  ColumnMetadata,
  DatabaseMetadata,
  SchemaConfig,
  SchemaMetadata,
  TableConfig,
  TableMetadata,
} from "@/types/proto/v1/database_service";
import { TinyTimer } from "@/utils";
import { SchemaEditorContext } from "../context";
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
type RichSchemaConfig = {
  schemaConfig: SchemaConfig;
};
type RichTableConfig = RichSchemaConfig & {
  tableConfig: TableConfig;
};
type RichColumnConfig = RichTableConfig & {
  columnConfig: ColumnConfig;
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
  sourceSchemaConfigMap = new Map<string, SchemaConfig>();
  targetSchemaConfigMap = new Map<string, SchemaConfig>();
  sourceTableConfigMap = new Map<string, TableConfig>();
  targetTableConfigMap = new Map<string, TableConfig>();
  sourceColumnConfigMap = new Map<string, ColumnConfig>();
  targetColumnConfigMap = new Map<string, ColumnConfig>();
  timer = new TinyTimer<
    | "merge"
    | "mergeSchemas"
    | "mergeTables"
    | "mergeColumns"
    | "diffColumn"
    | "mergeSchemaConfigs"
    | "mergeTableConfigs"
    | "mergeColumnConfigs"
    | "diffColumnConfig"
  >("DiffMerge");
  constructor(
    context: SchemaEditorContext,
    database: ComposedDatabase,
    sourceMetadata: DatabaseMetadata,
    targetMetadata: DatabaseMetadata
  ) {
    this.context = context;
    this.database = database;
    this.sourceMetadata = sourceMetadata;
    this.targetMetadata = targetMetadata;
  }
  merge() {
    this.timer.begin("merge");
    this.mergeSchemas();
    this.mergeSchemaConfigs();
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
      sourceSchemaConfigMap,
      targetSchemaConfigMap,
    } = this;
    const sourceSchemas = sourceMetadata.schemas;
    const targetSchemas = targetMetadata.schemas;
    this.timer.begin("mergeSchemas");
    mapSchemas(database, sourceSchemas, sourceSchemaMap);
    mapSchemas(database, targetSchemas, targetSchemaMap);
    mapSchemaConfigs(
      database,
      sourceMetadata.schemaConfigs,
      sourceSchemaConfigMap
    );
    mapSchemaConfigs(
      database,
      targetMetadata.schemaConfigs,
      targetSchemaConfigMap
    );

    const mergedSchemas: SchemaMetadata[] = [];
    for (let i = 0; i < sourceSchemas.length; i++) {
      const sourceSchema = sourceSchemas[i];
      const key = keyForResourceName(database.name, sourceSchema.name);
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
      const key = keyForResourceName(database.name, targetSchema.name);
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
      const key = keyForResourceName(
        database.name,
        sourceSchema.name,
        sourceTable.name
      );
      let targetTable = targetTableMap.get(key);
      if (targetTable) {
        // existed table
        mergedTables.push(targetTable);
        // merge columns for existed (maybe updated) table
        this.mergeColumns(
          { ...source, table: sourceTable },
          { ...target, table: targetTable }
        );
        if (
          !isEqual(sourceTable.classification, targetTable.classification) ||
          !isEqual(sourceTable.foreignKeys, targetTable.foreignKeys) ||
          !isEqual(sourceTable.indexes, targetTable.indexes)
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
      const key = keyForResourceName(
        database.name,
        targetSchema.name,
        targetTable.name
      );
      const sourceTable = sourceTableMap.get(key);
      if (!sourceTable) {
        // newly created table
        // mark it as 'created'
        mergedTables.push(targetTable);
        context.markEditStatusByKey(key, "created");
      }
    }
    targetSchema.tables = mergedTables;
    this.timer.end("mergeTables", sourceTables.length + targetTables.length);
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
    for (let i = 0; i < sourceColumns.length; i++) {
      const sourceColumn = sourceColumns[i];
      const key = keyForResourceName(
        database.name,
        sourceSchema.name,
        sourceTable.name,
        sourceColumn.name
      );
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
    for (let i = 0; i < targetColumns.length; i++) {
      const targetColumn = targetColumns[i];
      const key = keyForResourceName(
        database.name,
        targetSchema.name,
        targetTable.name,
        targetColumn.name
      );
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

    if (!isEqual(sourceColumn, targetColumn)) {
      const key = keyForResourceName(
        this.database.name,
        targetSchema.name,
        targetTable.name,
        targetColumn.name
      );
      this.context.markEditStatusByKey(key, "updated");
    }
    this.timer.end("diffColumn", 1);
  }
  mergeSchemaConfigs() {
    const {
      context,
      database,
      sourceMetadata,
      targetMetadata,
      sourceSchemaConfigMap,
      targetSchemaConfigMap,
    } = this;
    this.timer.begin("mergeSchemaConfigs");
    const sourceSchemaConfigs = sourceMetadata.schemaConfigs;
    const targetSchemaConfigs = targetMetadata.schemaConfigs;
    mapSchemaConfigs(database, sourceSchemaConfigs, sourceSchemaConfigMap);
    mapSchemaConfigs(database, targetSchemaConfigs, targetSchemaConfigMap);
    const mergedSchemaConfigs: SchemaConfig[] = [];
    for (let i = 0; i < targetMetadata.schemas.length; i++) {
      const schema = targetMetadata.schemas[i];
      const key = keyForResourceName(database.name, schema.name);
      const sourceSchemaConfig = sourceSchemaConfigMap.get(key);
      const targetSchemaConfig = targetSchemaConfigMap.get(key);
      const schemaStatus = context.getEditStatusByKey(key);
      if (schemaStatus === "dropped") {
        // copy source schemaConfig for further restoring
        if (sourceSchemaConfig) {
          mergedSchemaConfigs.push(cloneDeep(sourceSchemaConfig));
        }
      } else if (schemaStatus === "created") {
        // use newly created schemaConfig
        if (targetSchemaConfig) {
          mergedSchemaConfigs.push(targetSchemaConfig);
        }
      } else {
        // use the updated schemaConfig and diff recursively
        if (targetSchemaConfig) {
          mergedSchemaConfigs.push(targetSchemaConfig);
        }
        this.mergeTableConfigs(
          schema,
          {
            schemaConfig:
              sourceSchemaConfig ??
              SchemaConfig.fromPartial({ name: schema.name }),
          },
          {
            schemaConfig:
              targetSchemaConfig ??
              SchemaConfig.fromPartial({ name: schema.name }),
          }
        );
      }
    }
    targetMetadata.schemaConfigs = mergedSchemaConfigs;
    this.timer.end("mergeSchemaConfigs", targetMetadata.schemas.length);
  }
  mergeTableConfigs(
    schema: SchemaMetadata,
    source: RichSchemaConfig,
    target: RichSchemaConfig
  ) {
    const { context, database, sourceTableConfigMap, targetTableConfigMap } =
      this;
    const { schemaConfig: sourceSchemaConfig } = source;
    const { schemaConfig: targetSchemaConfig } = target;
    const sourceTableConfigs = sourceSchemaConfig.tableConfigs;
    const targetTableConfigs = targetSchemaConfig.tableConfigs;
    this.timer.begin("mergeTableConfigs");
    mapTableConfigs(
      database,
      sourceSchemaConfig,
      sourceTableConfigs,
      sourceTableConfigMap
    );
    mapTableConfigs(
      database,
      targetSchemaConfig,
      targetTableConfigs,
      targetTableConfigMap
    );
    const mergedTableConfigs: TableConfig[] = [];
    for (let i = 0; i < schema.tables.length; i++) {
      const table = schema.tables[i];
      const key = keyForResourceName(database.name, schema.name, table.name);
      const sourceTableConfig = sourceTableConfigMap.get(key);
      const targetTableConfig = targetTableConfigMap.get(key);
      const tableStatus = context.getEditStatusByKey(key);
      if (tableStatus === "dropped") {
        // copy source tableConfig for further restoring
        if (sourceTableConfig) {
          mergedTableConfigs.push(cloneDeep(sourceTableConfig));
        }
      } else if (tableStatus === "created") {
        // use newly created tableConfig
        if (targetTableConfig) {
          mergedTableConfigs.push(targetTableConfig);
        }
      } else {
        // use the updated tableConfig and diff recursively
        if (targetTableConfig) {
          mergedTableConfigs.push(targetTableConfig);
        }
        this.mergeColumnConfigs(
          schema,
          table,
          {
            ...source,
            tableConfig:
              sourceTableConfig ??
              TableConfig.fromPartial({ name: table.name }),
          },
          {
            ...target,
            tableConfig:
              targetTableConfig ??
              TableConfig.fromPartial({ name: table.name }),
          }
        );
      }
    }
    targetSchemaConfig.tableConfigs = mergedTableConfigs;
    this.timer.end("mergeTableConfigs", schema.tables.length);
  }
  mergeColumnConfigs(
    schema: SchemaMetadata,
    table: TableMetadata,
    source: RichTableConfig,
    target: RichTableConfig
  ) {
    const { context, database, sourceColumnConfigMap, targetColumnConfigMap } =
      this;
    const { schemaConfig: sourceSchemaConfig, tableConfig: sourceTableConfig } =
      source;
    const { schemaConfig: targetSchemaConfig, tableConfig: targetTableConfig } =
      target;
    const sourceColumnConfigs = sourceTableConfig.columnConfigs;
    const targetColumnConfigs = targetTableConfig.columnConfigs;
    this.timer.begin("mergeColumnConfigs");
    mapColumnConfigs(
      database,
      sourceSchemaConfig,
      sourceTableConfig,
      sourceColumnConfigs,
      sourceColumnConfigMap
    );
    mapColumnConfigs(
      database,
      targetSchemaConfig,
      targetTableConfig,
      targetColumnConfigs,
      targetColumnConfigMap
    );
    const mergedColumnConfigs: ColumnConfig[] = [];
    for (let i = 0; i < table.columns.length; i++) {
      const column = table.columns[i];
      const key = keyForResourceName(
        database.name,
        schema.name,
        table.name,
        column.name
      );
      const sourceColumnConfig = sourceColumnConfigMap.get(key);
      const targetColumnConfig = targetColumnConfigMap.get(key);
      const columnStatus = context.getEditStatusByKey(key);
      if (columnStatus === "dropped") {
        // copy source columnConfig for further restoring
        if (sourceColumnConfig) {
          mergedColumnConfigs.push(cloneDeep(sourceColumnConfig));
        }
      } else if (columnStatus === "created") {
        // use newly created columnConfig
        if (targetColumnConfig) {
          mergedColumnConfigs.push(targetColumnConfig);
        }
      } else {
        // use the updated columnConfig and diff recursively
        if (targetColumnConfig) {
          mergedColumnConfigs.push(targetColumnConfig);
        }
        this.diffColumnConfig(
          schema,
          table,
          column,
          {
            ...source,
            columnConfig:
              sourceColumnConfig ??
              ColumnConfig.fromPartial({ name: column.name }),
          },
          {
            ...target,
            columnConfig:
              targetColumnConfig ??
              ColumnConfig.fromPartial({ name: column.name }),
          }
        );
      }
    }
    targetTableConfig.columnConfigs = mergedColumnConfigs;
    this.timer.end("mergeColumnConfigs", table.columns.length);
  }
  diffColumnConfig(
    schema: SchemaMetadata,
    table: TableMetadata,
    column: ColumnMetadata,
    source: RichColumnConfig,
    target: RichColumnConfig
  ) {
    const { columnConfig: sourceColumnConfig } = source;
    const {
      schemaConfig: targetSchemaConfig,
      tableConfig: targetTableConfig,
      columnConfig: targetColumnConfig,
    } = target;
    this.timer.begin("diffColumnConfig");

    if (!isEqual(sourceColumnConfig, targetColumnConfig)) {
      const key = keyForResourceName(
        this.database.name,
        targetSchemaConfig.name,
        targetTableConfig.name,
        targetColumnConfig.name
      );
      const status = this.context.getEditStatusByKey(key);
      if (status !== "created" && status !== "dropped") {
        this.context.markEditStatusByKey(key, "updated");
      }
    }
    this.timer.end("diffColumnConfig", 1);
  }
}

const mapSchemas = (
  database: ComposedDatabase,
  schemas: SchemaMetadata[],
  map: Map<string, SchemaMetadata>
) => {
  schemas.forEach((schema) => {
    const key = keyForResourceName(database.name, schema.name);
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
    const key = keyForResourceName(database.name, schema.name, table.name);
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
    const key = keyForResourceName(
      database.name,
      schema.name,
      table.name,
      column.name
    );
    map.set(key, column);
  });
};
const mapSchemaConfigs = (
  db: ComposedDatabase,
  schemaConfigs: SchemaConfig[],
  map: Map<string, SchemaConfig>
) => {
  schemaConfigs.forEach((schemaConfig) => {
    const key = keyForResourceName(db.name, schemaConfig.name);
    map.set(key, schemaConfig);
  });
};
const mapTableConfigs = (
  db: ComposedDatabase,
  schemaConfig: SchemaConfig,
  tableConfigs: TableConfig[],
  map: Map<string, TableConfig>
) => {
  tableConfigs.forEach((tableConfig) => {
    const key = keyForResourceName(
      db.name,
      schemaConfig.name,
      tableConfig.name
    );
    map.set(key, tableConfig);
  });
};

const mapColumnConfigs = (
  db: ComposedDatabase,
  schemaConfig: SchemaConfig,
  tableConfig: TableConfig,
  columnConfigs: ColumnConfig[],
  map: Map<string, ColumnConfig>
) => {
  columnConfigs.forEach((columnConfig) => {
    const key = keyForResourceName(
      db.name,
      schemaConfig.name,
      tableConfig.name,
      columnConfig.name
    );
    map.set(key, columnConfig);
  });
};
