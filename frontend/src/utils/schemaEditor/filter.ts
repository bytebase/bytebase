import { cloneDeep } from "lodash-es";
import {
  ColumnMetadata,
  DatabaseMetadata,
  ForeignKeyMetadata,
  FunctionMetadata,
  IndexMetadata,
  ProcedureMetadata,
  SchemaMetadata,
  TableMetadata,
  TablePartitionMetadata,
  ViewMetadata,
} from "@/types/proto/v1/database_service";

// filterDatabaseMetadata filter out the objects/attributes we do not support.
// TODO: While supporting new objects/attributes, we should update this function.
// see backend's api/v1/branch_service.go
export const filterDatabaseMetadata = (metadata: DatabaseMetadata) => {
  return DatabaseMetadata.fromPartial({
    name: metadata.name,
    schemaConfigs: metadata.schemaConfigs,
    schemas: metadata.schemas.map((schema) => {
      return SchemaMetadata.fromPartial({
        name: schema.name,
        tables: schema.tables.map((table) => filterTableMetadata(table)),
        views: schema.views.map((view) => filterViewMetadata(view)),
        procedures: schema.procedures.map((procedure) =>
          filterProcedureMetadata(procedure)
        ),
        functions: schema.functions.map((func) => filterFunctionMetadata(func)),
      });
    }),
  });
};

export const filterColumnMetadata = (column: ColumnMetadata) => {
  return ColumnMetadata.fromPartial({
    name: column.name,
    comment: column.comment,
    userComment: column.userComment,
    type: column.type,
    hasDefault: column.hasDefault,
    defaultExpression: column.defaultExpression,
    defaultNull: column.defaultNull,
    defaultString: column.defaultString,
    onUpdate: column.onUpdate,
    nullable: column.nullable,
  });
};

export const filterIndexMetadata = (index: IndexMetadata) => {
  return IndexMetadata.fromPartial({
    name: index.name,
    definition: index.definition,
    primary: index.primary,
    unique: index.unique,
    comment: index.comment,
    expressions: index.expressions,
  });
};

export const filterForeignKeyMetadata = (fk: ForeignKeyMetadata) => {
  return ForeignKeyMetadata.fromPartial({
    name: fk.name,
    columns: fk.columns,
    referencedSchema: fk.referencedSchema,
    referencedTable: fk.referencedTable,
    referencedColumns: fk.referencedColumns,
  });
};

export const filterTablePartitionMetadata = (
  partition: TablePartitionMetadata
): TablePartitionMetadata => {
  return TablePartitionMetadata.fromPartial({
    name: partition.name,
    type: partition.type,
    useDefault: partition.useDefault,
    value: partition.value,
    expression: partition.expression,
    subpartitions: partition.subpartitions
      ? partition.subpartitions.map((sub) => filterTablePartitionMetadata(sub))
      : undefined,
  });
};

export const filterTableMetadata = (table: TableMetadata) => {
  return TableMetadata.fromPartial({
    name: table.name,
    comment: table.comment,
    userComment: table.userComment,
    collation: table.collation,
    engine: table.engine,
    columns: table.columns.map((column) => filterColumnMetadata(column)),
    indexes: table.indexes.map((index) => filterIndexMetadata(index)),
    foreignKeys: table.foreignKeys.map((fk) => filterForeignKeyMetadata(fk)),
    partitions: table.partitions.map((partition) =>
      filterTablePartitionMetadata(partition)
    ),
  });
};

export const filterViewMetadata = (view: ViewMetadata) => {
  return ViewMetadata.fromPartial({
    name: view.name,
    comment: view.comment,
    definition: view.definition,
    dependencyColumns: cloneDeep(view.dependencyColumns),
  });
};

export const filterProcedureMetadata = (procedure: ProcedureMetadata) => {
  return ProcedureMetadata.fromPartial({
    name: procedure.name,
    definition: procedure.definition,
  });
};

export const filterFunctionMetadata = (func: FunctionMetadata) => {
  return FunctionMetadata.fromPartial({
    name: func.name,
    definition: func.definition,
  });
};

export const ComparableTableFields: (keyof TableMetadata)[] = [
  "name",
  "comment",
  "userComment",
  "collation",
  "engine",
];
export const ComparableIndexFields: (keyof IndexMetadata)[] = [
  "name",
  "definition",
  "primary",
  "unique",
  "comment",
  "expressions",
];
export const ComparableForeignKeyFields: (keyof ForeignKeyMetadata)[] = [
  "name",
  "columns",
  "referencedSchema",
  "referencedTable",
  "referencedColumns",
];
export const ComparableTablePartitionFields: (keyof TablePartitionMetadata)[] =
  ["name", "type", "expression", "value"];
export const ComparableColumnFields: (keyof ColumnMetadata)[] = [
  "name",
  "comment",
  "userComment",
  "type",
  "hasDefault",
  "defaultExpression",
  "defaultNull",
  "defaultString",
  "onUpdate",
  "nullable",
];
