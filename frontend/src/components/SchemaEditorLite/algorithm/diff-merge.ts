import { cloneDeep } from "lodash-es";
import { ComposedDatabase } from "@/types";
import {
  ColumnMetadata,
  DatabaseMetadata,
  SchemaMetadata,
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
  timer = new TinyTimer<
    "merge" | "mergeSchemas" | "mergeTables" | "mergeColumns" | "diffColumn"
  >();
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
    // TODO: diff column and check if it is updated
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
