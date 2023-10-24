import { cloneDeep, isEqual, uniq } from "lodash-es";
import {
  ColumnMetadata,
  DatabaseMetadata,
  ForeignKeyMetadata,
  IndexMetadata,
  SchemaMetadata,
  TableMetadata,
  SchemaConfig,
  ColumnConfig,
  TableConfig,
} from "@/types/proto/v1/database_service";
import {
  Column,
  ForeignKey,
  Schema,
  Table,
  convertColumnMetadataToColumn,
  convertSchemaMetadataToSchema,
  convertTableMetadataToTable,
} from "@/types/v1/schemaEditor";
import { randomString } from "@/utils";

export const mergeSchemaEditToMetadata = (
  schemaEdits: Schema[],
  metadata: DatabaseMetadata
): DatabaseMetadata => {
  for (const schemaEdit of schemaEdits) {
    if (schemaEdit.status === "created") {
      // Remove schema if it exists.
      metadata.schemas = metadata.schemas.filter(
        (item) => item.name !== schemaEdit.name
      );
      metadata.schemas.push(transformSchemaEditToMetadata(schemaEdit));
    } else if (schemaEdit.status === "dropped") {
      metadata.schemas = metadata.schemas.filter(
        (item) => item.name !== schemaEdit.name
      );
    } else {
      const schema = metadata.schemas.find(
        (item) => item.name === schemaEdit.name
      );
      if (!schema) {
        metadata.schemas.push(transformSchemaEditToMetadata(schemaEdit));
        continue;
      }

      for (const tableEdit of schemaEdit.tableList) {
        if (tableEdit.status === "created") {
          // Remove table if it exists.
          schema.tables = schema.tables.filter(
            (item) => item.name !== tableEdit.name
          );
          schema.tables.push(transformTableEditToMetadata(tableEdit));
        } else if (tableEdit.status === "dropped") {
          schema.tables = schema.tables.filter(
            (item) => item.name !== tableEdit.name
          );
        } else {
          const table = schema.tables.find(
            (item) => item.name === tableEdit.name
          );
          if (!table) {
            schema.tables.push(transformTableEditToMetadata(tableEdit));
            continue;
          }

          for (const columnEdit of tableEdit.columnList) {
            if (columnEdit.status === "created") {
              // Remove column if it exists.
              table.columns = table.columns.filter(
                (item) => item.name !== columnEdit.name
              );
              table.columns.push(transformColumnEditToMetadata(columnEdit));
            } else if (columnEdit.status === "dropped") {
              table.columns = table.columns.filter(
                (item) => item.name !== columnEdit.name
              );
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
              column.classification = columnEdit.classification ?? "";
              column.hasDefault = columnEdit.hasDefault;
              column.defaultNull = columnEdit.defaultNull;
              column.defaultString = columnEdit.defaultString;
              column.defaultExpression = columnEdit.defaultExpression;
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
          if (tableEdit.primaryKey.columnIdList.length > 0) {
            const primaryIndex =
              table.indexes.find((index) => index.primary) ||
              IndexMetadata.fromPartial({
                // TODO: Maybe we should let user to specify the index name.
                name: tableEdit.primaryKey.name || `pk_${tableEdit.name}`,
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
            primaryIndex.expressions = uniq(primaryIndex.expressions);
          } else {
            table.indexes = table.indexes.filter((index) => !index.primary);
          }
          table.comment = tableEdit.comment;
          table.userComment = tableEdit.userComment;
          table.classification = tableEdit.classification || "";
        }
      }

      for (const table of schema.tables) {
        const tableEdit = schemaEdit.tableList.find(
          (item) => item.name === table.name
        );
        if (!tableEdit) {
          schema.tables = schema.tables.filter(
            (item) => item.name !== table.name
          );
        }
      }
    }
  }

  for (const schema of metadata.schemas) {
    const schemaEdit = schemaEdits.find((item) => item.name === schema.name);
    if (!schemaEdit) {
      metadata.schemas = metadata.schemas.filter(
        (item) => item.name !== schema.name
      );
      continue;
    }

    // Build foreign keys.
    for (const tableEdit of schemaEdit?.tableList ?? []) {
      for (const foreignKey of tableEdit.foreignKeyList) {
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
          ?.tableList.find(
            (table) => table.id === foreignKey.referencedTableId
          );
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
  }

  metadata.schemaConfigs = schemaEdits.map((schema) =>
    SchemaConfig.fromPartial({
      name: schema.name,
      tableConfigs: schema.tableList.map((table) =>
        TableConfig.fromPartial({
          name: table.name,
          columnConfigs: table.columnList.map((column) =>
            ColumnConfig.fromPartial({
              ...column.config,
              name: column.name,
            })
          ),
        })
      ),
    })
  );

  return metadata;
};

export const initialSchemaConfigToMetadata = (metadata: DatabaseMetadata) => {
  metadata.schemaConfigs = metadata.schemas.map((schema) =>
    SchemaConfig.fromPartial({
      name: schema.name,
      tableConfigs: schema.tables.map((table) =>
        TableConfig.fromPartial({
          name: table.name,
          columnConfigs: table.columns.map((column) =>
            ColumnConfig.fromPartial({
              name: column.name,
            })
          ),
        })
      ),
    })
  );
};

const transformSchemaEditToMetadata = (schemaEdit: Schema): SchemaMetadata => {
  const schemaMetadata = SchemaMetadata.fromPartial({
    name: schemaEdit.name,
    tables: [],
  });

  for (const table of schemaEdit.tableList) {
    const tableMetadata = transformTableEditToMetadata(table);
    schemaMetadata.tables.push(tableMetadata);
  }

  return schemaMetadata;
};

export const transformTableEditToMetadata = (
  tableEdit: Table
): TableMetadata => {
  const tableMetadata = TableMetadata.fromPartial({
    name: tableEdit.name,
    columns: [],
    indexes: [],
    foreignKeys: [],
    comment: tableEdit.comment,
    userComment: tableEdit.userComment,
    classification: tableEdit.classification,
  });

  for (const column of tableEdit.columnList) {
    tableMetadata.columns.push(
      ColumnMetadata.fromPartial({
        name: column.name,
        type: column.type,
        nullable: column.nullable,
        hasDefault: column.hasDefault,
        defaultNull: column.defaultNull,
        defaultString: column.defaultString,
        defaultExpression: column.defaultExpression,
        comment: column.comment,
        userComment: column.userComment,
        classification: column.classification,
      })
    );
  }

  if (tableEdit.primaryKey.columnIdList.length > 0) {
    const primaryIndex = IndexMetadata.fromPartial({
      // TODO: Maybe we should let user to specify the index name.
      name: tableEdit.primaryKey.name || `pk_${tableEdit.name}`,
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
    hasDefault: columnEdit.hasDefault,
    defaultNull: columnEdit.defaultNull,
    defaultString: columnEdit.defaultString,
    defaultExpression: columnEdit.defaultExpression,
    comment: columnEdit.comment,
    userComment: columnEdit.userComment,
    classification: columnEdit.classification,
  });
};

export const rebuildEditableSchemas = (
  originalSchemas: Schema[],
  schemas: SchemaMetadata[],
  schemaConfigList: SchemaConfig[]
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

    const schemaConfig =
      schemaConfigList.find(
        (schemaConfig) => schemaConfig.name === editableSchema.name
      ) ?? SchemaConfig.fromPartial({});

    for (const editableTable of editableSchema.tableList) {
      const table = schema.tables.find(
        (table) => table.name === editableTable.name
      );
      if (!table) {
        editableTable.status = "dropped";
        continue;
      }

      editableTable.userComment = table.userComment;
      editableTable.classification = table.classification;
      editableTable.comment = table.comment;

      const tableConfig =
        schemaConfig.tableConfigs.find(
          (tableConfig) => tableConfig.name === editableTable.name
        ) ?? TableConfig.fromPartial({});

      for (const editableColumn of editableTable.columnList) {
        editableColumn.config =
          tableConfig.columnConfigs.find(
            (columnConfig) => columnConfig.name === editableColumn.name
          ) ?? ColumnConfig.fromPartial({ name: editableColumn.name });

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
        editableColumn.classification = column.classification;
        editableColumn.hasDefault = column.hasDefault;
        editableColumn.defaultNull = column.defaultNull;
        editableColumn.defaultString = column.defaultString;
        editableColumn.defaultExpression = column.defaultExpression;
      }

      for (const column of table.columns) {
        const editableColumn = editableTable.columnList.find(
          (item) => item.name === column.name
        );
        if (!editableColumn) {
          const newColumn = convertColumnMetadataToColumn(
            column,
            "created",
            tableConfig.columnConfigs.find(
              (columnConfig) => columnConfig.name === column.name
            )
          );
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
      } else {
        editableTable.primaryKey = {
          name: "",
          columnIdList: [],
        };
      }
    }

    for (const table of schema.tables) {
      const editableTable = editableSchema.tableList.find(
        (item) => item.name === table.name
      );
      if (!editableTable) {
        const newTable = convertTableMetadataToTable(
          table,
          "created",
          schemaConfig.tableConfigs.find(
            (tableConfig) => tableConfig.name === table.name
          )
        );
        editableSchema.tableList.push(newTable);
      }
    }
  }

  for (const schema of schemas) {
    const editableSchema = editableSchemas.find(
      (item) => item.name === schema.name
    );
    if (!editableSchema) {
      const newSchema = convertSchemaMetadataToSchema(
        schema,
        "created",
        schemaConfigList.find(
          (schemaConfig) => schemaConfig.name === schema.name
        )
      );
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

    for (const table of schema.tables) {
      const editableTable = editableSchema.tableList.find(
        (item) => item.name === table.name
      );
      if (!editableTable) {
        continue;
      }

      const foreignKeyList: ForeignKey[] = [];
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
      editableTable.foreignKeyList = foreignKeyList;
    }
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
