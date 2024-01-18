import {
  ColumnMetadata,
  DatabaseMetadata,
  ForeignKeyMetadata,
  IndexMetadata,
  SchemaMetadata,
  TableMetadata,
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
      });
    }),
  });
};

export const filterColumnMetadata = (column: ColumnMetadata) => {
  return ColumnMetadata.fromPartial({
    name: column.name,
    comment: column.comment,
    userComment: column.userComment,
    classification: column.classification,
    type: column.type,
    hasDefault: column.hasDefault,
    defaultExpression: column.defaultExpression,
    defaultNull: column.defaultNull,
    defaultString: column.defaultString,
    nullable: column.nullable,
    position: column.position,
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

export const filterTableMetadata = (table: TableMetadata) => {
  return TableMetadata.fromPartial({
    name: table.name,
    classification: table.classification,
    comment: table.comment,
    userComment: table.userComment,
    collation: table.collation,
    engine: table.engine,
    columns: table.columns.map((column) => filterColumnMetadata(column)),
    indexes: table.indexes.map((index) => filterIndexMetadata(index)),
    foreignKeys: table.foreignKeys.map((fk) => filterForeignKeyMetadata(fk)),
  });
};

export const ComparableTableFields: (keyof TableMetadata)[] = [
  "name",
  "classification",
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
export const ComparableColumnFields: (keyof ColumnMetadata)[] = [
  "name",
  "comment",
  "userComment",
  "classification",
  "type",
  "hasDefault",
  "defaultExpression",
  "defaultNull",
  "defaultString",
  "nullable",
  "position",
];
