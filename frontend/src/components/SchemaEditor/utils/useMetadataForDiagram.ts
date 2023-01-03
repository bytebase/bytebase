import { computed, unref } from "vue";

import { Database, DatabaseSchema, MaybeRef } from "@/types";
import { Schema, Table } from "@/types/schemaEditor/atomType";
import { EditStatus } from "@/components/SchemaDiagram";
import {
  ColumnMetadata,
  ForeignKeyMetadata,
  TableMetadata,
} from "@/types/proto/store/database";
import { isTableChanged } from "./table";

type MetadataWithEditStatus<T> = T & {
  $$status: EditStatus;
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
  if (isTableChanged(database.id, schema.name, table.id)) {
    return "changed";
  }

  return "normal";
};

export const useMetadataForDiagram = (
  databaseSchema: MaybeRef<DatabaseSchema>
) => {
  const tableMetadataList = computed(() => {
    const { database, schemaList } = unref(databaseSchema);

    return unref(schemaList).flatMap((schema) => {
      const { tableList } = schema;

      return tableList.map((table) => {
        const tableMeta = TableMetadata.fromPartial({});
        Object.defineProperty(tableMeta, "$$status", {
          enumerable: false,
          value: statusOfTable(database, schema, table),
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
            value: column.status,
          });

          columnMeta.name = column.name;
          columnMeta.type = column.type;
          columnMeta.nullable = column.nullable;
          columnMeta.comment = column.comment;
          columnMeta.default = column.default;

          return columnMeta;
        });

        tableMeta.indexes = [
          {
            primary: true,
            type: "",
            unique: true,
            comment: "",
            visible: true,
            name: table.primaryKey.name,
            expressions: table.primaryKey.columnIdList.map(
              (id) => table.columnList.find((col) => col.id === id)!.name
            ),
          },
        ];
        const foreignKeyList = schema.foreignKeyList.filter(
          (fk) => fk.tableId === table.id
        );

        tableMeta.foreignKeys = foreignKeyList.map((fk) => {
          const refSchema = unref(schemaList).find(
            (schema) => schema.name === fk.referencedSchema
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
    });
  });

  const tableStatus = (tableMeta: TableMetadata): EditStatus => {
    const status = (tableMeta as MetadataWithEditStatus<TableMetadata>)
      .$$status;
    if (typeof status !== "undefined") {
      return status;
    }
    return "normal";
  };
  const columnStatus = (columnMeta: ColumnMetadata): EditStatus => {
    const status = (columnMeta as MetadataWithEditStatus<ColumnMetadata>)
      .$$status;
    if (typeof status !== "undefined") {
      return status;
    }
    return "normal";
  };

  return {
    tableMetadataList,
    tableStatus,
    columnStatus,
  };
};
