import { Column, Schema, Table } from "@/types";
import {
  ColumnMetadata,
  DatabaseMetadata,
  ForeignKeyMetadata,
  IndexMetadata,
  SchemaMetadata,
  TableMetadata,
} from "@/types/proto/v1/database_service";
import { randomString } from "@/utils";

export const mergeSchemaEditToMetadata = (
  schemaEdits: Schema[],
  metadata: DatabaseMetadata
): DatabaseMetadata => {
  for (const schemaEdit of schemaEdits) {
    if (schemaEdit.status === "created") {
      metadata.schemas.push(transformSchemaEditToMetadata(schemaEdit));
      continue;
    } else if (schemaEdit.status === "dropped") {
      metadata.schemas = metadata.schemas.filter(
        (item) => item.name !== schemaEdit.name
      );
      continue;
    } else {
      const schema = metadata.schemas.find(
        (item) => item.name === schemaEdit.name
      );
      if (!schema) {
        continue;
      }
      for (const tableEdit of schemaEdit.tableList) {
        if (tableEdit.status === "created") {
          schema.tables.push(transformTableEditToMetadata(tableEdit));
          continue;
        } else if (tableEdit.status === "dropped") {
          schema.tables = schema.tables.filter(
            (item) => item.name !== tableEdit.name
          );
          continue;
        } else {
          const table = schema.tables.find(
            (item) => item.name === tableEdit.name
          );
          if (!table) {
            continue;
          }
          for (const columnEdit of tableEdit.columnList) {
            if (columnEdit.status === "created") {
              table.columns.push(transformColumnEditToMetadata(columnEdit));
              continue;
            } else if (columnEdit.status === "dropped") {
              table.columns = table.columns.filter(
                (item) => item.name !== columnEdit.name
              );
              continue;
            } else {
              const column = table.columns.find(
                (item) => item.name === columnEdit.name
              );
              if (!column) {
                continue;
              }
              column.type = columnEdit.type;
              column.nullable = columnEdit.nullable;
              column.comment = columnEdit.comment;
              column.userComment = columnEdit.userComment;
              column.default = columnEdit.default;
            }
          }
        }
      }
    }

    for (const foreignKey of schemaEdit.foreignKeyList) {
      const schema = metadata.schemas.find(
        (schema) => schema.name === schemaEdit.name
      );
      if (!schema) {
        continue;
      }
      const tableEdit = schemaEdit.tableList.find(
        (table) => table.id === foreignKey.tableId
      );
      const table = schema.tables.find(
        (table) => table.name === tableEdit?.name
      );
      if (!tableEdit || !table) {
        continue;
      }
      const referencedSchema = metadata.schemas.find(
        (schema) =>
          schema.name ===
          schemaEdits.find(
            (schemaEdit) => schemaEdit.id === foreignKey.referencedSchemaId
          )?.name
      );
      if (!referencedSchema) {
        continue;
      }
      const referencedTableEdit = schemaEdits
        .find((schemaEdit) => schemaEdit.id === foreignKey.referencedSchemaId)
        ?.tableList.find((table) => table.id === foreignKey.referencedTableId);
      const referencedTable = referencedSchema.tables.find(
        (table) => table.name === referencedTableEdit?.name
      );
      if (!referencedTableEdit || !referencedTable) {
        continue;
      }

      const foreignKeyName = foreignKey.name
        ? foreignKey.name
        : `${table.name}-fk-${randomString(8).toLowerCase()}`;
      const fk = ForeignKeyMetadata.fromPartial({
        name: foreignKeyName,
        referencedSchema: referencedSchema.name,
        referencedTable: referencedTable.name,
      });
      if (table.foreignKeys.find((fk) => fk.name === foreignKeyName)) {
        continue;
      }
      if (
        foreignKey.columnIdList.length !==
        foreignKey.referencedColumnIdList.length
      ) {
        continue;
      }
      for (const columnId of foreignKey.columnIdList) {
        const column = tableEdit.columnList.find(
          (column) => column.id === columnId
        );
        if (column) {
          fk.columns.push(column.name);
        }
      }
      for (const columnId of foreignKey.referencedColumnIdList) {
        const column = referencedTableEdit.columnList.find(
          (column) => column.id === columnId
        );
        if (column) {
          fk.referencedColumns.push(column.name);
        }
      }
      table.foreignKeys.push(fk);
    }
  }

  return metadata;
};

const transformSchemaEditToMetadata = (schemaEdit: Schema): SchemaMetadata => {
  const schemaMetadata = SchemaMetadata.fromPartial({
    name: schemaEdit.name,
    tables: [],
  });

  for (const table of schemaEdit.tableList) {
    const tableMetadata = TableMetadata.fromPartial({
      name: table.name,
      columns: [],
      indexes: [],
      foreignKeys: [],
    });

    for (const column of table.columnList) {
      tableMetadata.columns.push(
        ColumnMetadata.fromPartial({
          name: column.name,
          type: column.type,
          nullable: column.nullable,
          default: column.default,
          comment: column.comment,
          userComment: column.userComment,
        })
      );
    }

    if (table.primaryKey.columnIdList.length > 0) {
      const primaryIndex = IndexMetadata.fromPartial({
        name: `${table.name}-pk-${randomString(8).toLowerCase()}`,
        primary: true,
        expressions: [],
      });
      for (const columnId of table.primaryKey.columnIdList) {
        const column = table.columnList.find(
          (column) => column.id === columnId
        );
        if (column) {
          primaryIndex.expressions.push(column.name);
        }
      }
      if (primaryIndex.expressions.length > 0) {
        tableMetadata.indexes.push(primaryIndex);
      }
    }

    schemaMetadata.tables.push(tableMetadata);
  }

  return schemaMetadata;
};

const transformTableEditToMetadata = (tableEdit: Table): TableMetadata => {
  const tableMetadata = TableMetadata.fromPartial({
    name: tableEdit.name,
    columns: [],
    indexes: [],
    foreignKeys: [],
  });

  for (const column of tableEdit.columnList) {
    tableMetadata.columns.push(
      ColumnMetadata.fromPartial({
        name: column.name,
        type: column.type,
        nullable: column.nullable,
        default: column.default,
        comment: column.comment,
        userComment: column.userComment,
      })
    );
  }

  if (tableEdit.primaryKey.columnIdList.length > 0) {
    const primaryIndex = IndexMetadata.fromPartial({
      name: `${tableEdit.name}-pk-${randomString(8).toLowerCase()}`,
      primary: true,
      expressions: [],
    });
    for (const columnId of tableEdit.primaryKey.columnIdList) {
      const column = tableEdit.columnList.find(
        (column) => column.id === columnId
      );
      if (column) {
        primaryIndex.expressions.push(column.name);
      }
    }
    if (primaryIndex.expressions.length > 0) {
      tableMetadata.indexes.push(primaryIndex);
    }
  }

  return tableMetadata;
};

const transformColumnEditToMetadata = (columnEdit: Column): ColumnMetadata => {
  return ColumnMetadata.fromPartial({
    name: columnEdit.name,
    type: columnEdit.type,
    nullable: columnEdit.nullable,
    default: columnEdit.default,
    comment: columnEdit.comment,
    userComment: columnEdit.userComment,
  });
};
