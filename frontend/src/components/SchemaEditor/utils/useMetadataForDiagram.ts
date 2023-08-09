import { computed, unref } from "vue";
import { EditStatus } from "@/components/SchemaDiagram";
import { Database, DatabaseSchema, MaybeRef } from "@/types";
import {
  ColumnMetadata,
  DatabaseMetadata,
  ForeignKeyMetadata,
  IndexMetadata,
  SchemaMetadata,
  TableMetadata,
} from "@/types/proto/store/database";
import { Column, Schema, Table } from "@/types/schemaEditor/atomType";
import { isColumnChanged } from "./column";
import { isSchemaChanged } from "./schema";
import { isTableChanged } from "./table";

type MetadataWithEditStatus<T, E> = T & {
  $$status?: EditStatus;
  $$edit?: E;
};

const statusOfSchema = (database: Database, schema: Schema) => {
  const { status } = schema;
  if (status === "created" || status === "dropped") {
    return status;
  }
  if (isSchemaChanged(database.id, schema.id)) {
    return "changed";
  }
  return "normal";
};

const statusOfTable = (
  database: Database,
  schema: Schema,
  table: Table
): EditStatus => {
  const { status } = table;
  if (status === "created" || status === "dropped") {
    return status;
  }
  if (isTableChanged(database.id, schema.id, table.id)) {
    return "changed";
  }

  return "normal";
};

const statusOfColumn = (
  database: Database,
  schema: Schema,
  table: Table,
  column: Column
): EditStatus => {
  const { status } = column;
  if (status === "created" || status === "dropped") {
    return status;
  }
  if (isColumnChanged(database.id, schema.id, table.id, column.id)) {
    return "changed";
  }

  return "normal";
};

export const useMetadataForDiagram = (
  databaseSchema: MaybeRef<DatabaseSchema>
) => {
  const databaseMetadata = computed(() => {
    const { database, schemaList } = unref(databaseSchema);
    const databaseMeta = DatabaseMetadata.fromPartial({});
    databaseMeta.name = database.name;
    databaseMeta.collation = database.collation;
    databaseMeta.characterSet = database.characterSet;
    databaseMeta.schemas = schemaList.map((schema) => {
      const schemaMeta = SchemaMetadata.fromPartial({});
      Object.defineProperty(schemaMeta, "$$status", {
        enumerable: false,
        value: statusOfSchema(database, schema),
      });
      Object.defineProperty(schemaMeta, "$$edit", {
        enumerable: false,
        value: schema,
      });

      schemaMeta.name = schema.name;
      schemaMeta.tables = schema.tableList.map((table) => {
        const tableMeta = TableMetadata.fromPartial({});
        Object.defineProperty(tableMeta, "$$status", {
          enumerable: false,
          value: statusOfTable(database, schema, table),
        });
        Object.defineProperty(tableMeta, "$$edit", {
          enumerable: false,
          value: table,
        });

        tableMeta.name = table.name;
        tableMeta.engine = table.engine;
        tableMeta.collation = table.collation;
        tableMeta.rowCount = table.rowCount;
        tableMeta.dataSize = table.dataSize;
        tableMeta.comment = table.comment;

        tableMeta.columns = table.columnList.map((column) => {
          const columnMeta = ColumnMetadata.fromPartial({});
          Object.defineProperty(columnMeta, "$$status", {
            enumerable: false,
            value: statusOfColumn(database, schema, table, column),
          });
          Object.defineProperty(columnMeta, "$$edit", {
            enumerable: false,
            value: column,
          });

          columnMeta.name = column.name;
          columnMeta.type = column.type;
          columnMeta.nullable = column.nullable;
          columnMeta.comment = column.comment;
          columnMeta.default = column.default;

          return columnMeta;
        });

        // We don't have indexes other than primary key in Schema Editor,
        // so something will lost here when converting Table back to
        // TableMetadata.
        // But they will be back soon when editing indexes is supported in
        // Schema Editor.
        const pk = IndexMetadata.fromPartial({});
        Object.assign(pk, {
          primary: true,
          name: table.primaryKey.name,
          expressions: table.primaryKey.columnIdList.map(
            (id) => table.columnList.find((col) => col.id === id)!.name
          ),
        });
        tableMeta.indexes = [pk];

        const foreignKeyList = schema.foreignKeyList.filter(
          (fk) => fk.tableId === table.id
        );
        tableMeta.foreignKeys = foreignKeyList.map((fk) => {
          // In PostgreSQL, foreign keys can cross different schemas.
          // So we need to search the schemaList here.
          const refSchema = schemaList.find(
            (schema) => schema.id === fk.referencedSchemaId
          )!;
          const refTable = refSchema.tableList.find(
            (table) => table.id === fk.referencedTableId
          )!;
          const fkMeta = ForeignKeyMetadata.fromPartial({});
          Object.assign(fkMeta, {
            columns: fk.columnIdList.map(
              (id) => table.columnList.find((col) => col.id === id)!.name
            ),
            name: fk.name,
            referencedSchema: refSchema.name,
            referencedTable: refTable.name,
            referencedColumns: fk.referencedColumnIdList.map(
              (id) => refTable.columnList.find((col) => col.id === id)!.name
            ),
          });
          return fkMeta;
        });
        return tableMeta;
      });
      return schemaMeta;
    });

    return databaseMeta;
  });

  const schemaStatus = (schemaMeta: SchemaMetadata): EditStatus => {
    const status = (schemaMeta as MetadataWithEditStatus<SchemaMetadata, Table>)
      .$$status;
    if (typeof status !== "undefined") {
      return status;
    }
    return "normal";
  };
  const tableStatus = (tableMeta: TableMetadata): EditStatus => {
    const status = (tableMeta as MetadataWithEditStatus<TableMetadata, Table>)
      .$$status;
    if (typeof status !== "undefined") {
      return status;
    }
    return "normal";
  };
  const columnStatus = (columnMeta: ColumnMetadata): EditStatus => {
    const status = (
      columnMeta as MetadataWithEditStatus<ColumnMetadata, Column>
    ).$$status;
    if (typeof status !== "undefined") {
      return status;
    }
    return "normal";
  };
  const editableSchema = (schemaMeta: SchemaMetadata) => {
    const schema = (
      schemaMeta as MetadataWithEditStatus<SchemaMetadata, Schema>
    ).$$edit;
    return schema;
  };
  const editableTable = (tableMeta: TableMetadata) => {
    const table = (tableMeta as MetadataWithEditStatus<TableMetadata, Table>)
      .$$edit;
    return table;
  };
  const editableColumn = (columnMeta: ColumnMetadata) => {
    const column = (
      columnMeta as MetadataWithEditStatus<ColumnMetadata, Column>
    ).$$edit;
    return column;
  };

  return {
    databaseMetadata,
    schemaStatus,
    tableStatus,
    columnStatus,
    editableSchema,
    editableTable,
    editableColumn,
  };
};
