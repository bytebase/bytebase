import { cloneDeep, isEqual, uniq } from "lodash-es";
import { useCurrentUserV1 } from "@/store";
import { useSchemaDesignStore } from "@/store/modules/schemaDesign";
import {
  Column,
  ForeignKey,
  Schema,
  Table,
  convertColumnMetadataToColumn,
  convertSchemaMetadataToSchema,
  convertTableMetadataToTable,
} from "@/types";
import {
  ColumnMetadata,
  DatabaseMetadata,
  ForeignKeyMetadata,
  IndexMetadata,
  SchemaMetadata,
  TableMetadata,
} from "@/types/proto/v1/database_service";
import { SchemaDesign } from "@/types/proto/v1/schema_design_service";
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
                table.columns.push(transformColumnEditToMetadata(columnEdit));
                continue;
              }
              column.type = columnEdit.type;
              column.nullable = columnEdit.nullable;
              column.comment = columnEdit.comment;
              column.userComment = columnEdit.userComment;
              column.default = columnEdit.default;
            }
          }
          for (const column of table.columns) {
            const columnEdit = tableEdit.columnList.find(
              (item) => item.name === column.name
            );
            if (!columnEdit) {
              table.columns = table.columns.filter(
                (item) => item.name !== column.name
              );
            }
          }
        }
      }
    }

    // Build foreign keys.
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

export const rebuildEditableSchemas = (
  originalSchemas: Schema[],
  schemas: SchemaMetadata[]
): Schema[] => {
  const editableSchemas = cloneDeep(originalSchemas);

  for (const editableSchema of editableSchemas) {
    const schema = schemas.find(
      (schema) => schema.name === editableSchema.name
    );
    if (!schema) {
      editableSchema.status = "dropped";
      continue;
    }

    for (const editableTable of editableSchema.tableList) {
      const table = schema.tables.find(
        (table) => table.name === editableTable.name
      );
      if (!table) {
        editableTable.status = "dropped";
        continue;
      }

      for (const editableColumn of editableTable.columnList) {
        const column = table.columns.find(
          (column) => column.name === editableColumn.name
        );
        if (!column) {
          editableColumn.status = "dropped";
          continue;
        }
        if (isEqual(transformColumnEditToMetadata(editableColumn), column)) {
          continue;
        }

        editableColumn.type = column.type;
        editableColumn.nullable = column.nullable;
        editableColumn.comment = column.comment;
        editableColumn.userComment = column.userComment;
        editableColumn.default = column.default;
      }

      for (const column of table.columns) {
        const editableColumn = editableTable.columnList.find(
          (item) => item.name === column.name
        );
        if (!editableColumn) {
          const newColumn = convertColumnMetadataToColumn(column);
          newColumn.status = "created";
          editableTable.columnList.push(newColumn);
        }
      }

      // Rebuild primary key from primary index.
      const primaryIndex = table.indexes.find(
        (index) => index.primary === true
      );
      if (primaryIndex) {
        editableTable.primaryKey.name = primaryIndex.name;
        for (const columnName of primaryIndex.expressions) {
          const column = editableTable.columnList.find(
            (column) => column.name === columnName
          );
          if (column) {
            editableTable.primaryKey.columnIdList.push(column.id);
          }
        }
        editableTable.primaryKey.columnIdList = uniq(
          editableTable.primaryKey.columnIdList
        );
      }
    }

    for (const table of schema.tables) {
      const editableTable = editableSchema.tableList.find(
        (item) => item.name === table.name
      );
      if (!editableTable) {
        const newTable = convertTableMetadataToTable(table);
        newTable.status = "created";
        for (const column of newTable.columnList) {
          column.status = "created";
        }
        editableSchema.tableList.push(newTable);
      }
    }
  }

  for (const schema of schemas) {
    const editableSchema = editableSchemas.find(
      (item) => item.name === schema.name
    );
    if (!editableSchema) {
      const newSchema = convertSchemaMetadataToSchema(schema);
      newSchema.status = "created";
      for (const table of newSchema.tableList) {
        table.status = "created";
        for (const column of table.columnList) {
          column.status = "created";
        }
      }
      editableSchemas.push(newSchema);
    }
  }

  // Build foreign keys for schema and referenced schema.
  for (const schema of schemas) {
    const editableSchema = editableSchemas.find(
      (schema) => schema.name === schema.name
    );
    if (!editableSchema) {
      continue;
    }

    const foreignKeyList: ForeignKey[] = [];
    for (const table of schema.tables) {
      const editableTable = editableSchema.tableList.find(
        (item) => item.name === table.name
      );
      if (!editableTable) {
        continue;
      }

      for (const foreignKeyMetadata of table.foreignKeys) {
        const referencedSchema = editableSchemas.find(
          (schema) => schema.name === foreignKeyMetadata.referencedSchema
        );
        const referencedTable = referencedSchema?.tableList.find(
          (table) => table.name === foreignKeyMetadata.referencedTable
        );
        if (!referencedSchema || !referencedTable) {
          continue;
        }

        const fk: ForeignKey = {
          name: foreignKeyMetadata.name,
          tableId: editableTable.id,
          columnIdList: [],
          referencedSchemaId: referencedSchema.id,
          referencedTableId: referencedTable.id,
          referencedColumnIdList: [],
        };
        for (const columnName of foreignKeyMetadata.columns) {
          const column = editableTable.columnList.find(
            (column) => column.name === columnName
          );
          if (column) {
            fk.columnIdList.push(column.id);
          }
        }
        for (const referencedColumnName of foreignKeyMetadata.referencedColumns) {
          const referencedColumn = referencedTable.columnList.find(
            (column) => column.name === referencedColumnName
          );
          if (referencedColumn) {
            fk.referencedColumnIdList.push(referencedColumn.id);
          }
        }

        foreignKeyList.push(fk);
      }
    }
    editableSchema.foreignKeyList = foreignKeyList;
  }

  return editableSchemas;
};

export const validateDatabaseMetadata = (
  databaseMetadata: DatabaseMetadata
): string[] => {
  const messages: string[] = [];

  for (const schema of databaseMetadata.schemas) {
    for (const table of schema.tables) {
      if (!table.name) {
        messages.push(`Table name is required.`);
        continue;
      }

      for (const column of table.columns) {
        if (!column.name) {
          messages.push(`Column name is required.`);
          continue;
        }
        if (!column.type) {
          messages.push(`Column ${column.name} type is required.`);
        }
      }
    }
  }

  return uniq(messages);
};

export const generateForkedBranchName = (branch: SchemaDesign): string => {
  const currentUser = useCurrentUserV1();
  const schemaDesignStore = useSchemaDesignStore();
  const parentBranchName = branch.title;
  let branchName = `${currentUser.value.title}/${parentBranchName}`;
  const foundIndex = schemaDesignStore.schemaDesignList.findIndex((item) => {
    return item.title === branchName;
  });
  // If found, add a random string to the end of the branch name.
  if (foundIndex > -1) {
    branchName = `${
      currentUser.value.title
    }/${parentBranchName}-draft-${randomString(3).toLowerCase()}`;
  }
  return branchName;
};
