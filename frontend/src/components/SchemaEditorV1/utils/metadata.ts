import { cloneDeep, isEqual, uniq } from "lodash-es";
import type { DatabaseMetadata } from "@/types/proto/v1/database_service";
import {
  ColumnMetadata,
  IndexMetadata,
  SchemaMetadata,
  TableMetadata,
  SchemaConfig,
  ColumnConfig,
  TableConfig,
} from "@/types/proto/v1/database_service";
import type {
  Column,
  ForeignKey,
  Schema,
  Table,
} from "@/types/v1/schemaEditor";
import {
  convertColumnMetadataToColumn,
  convertSchemaMetadataToSchema,
  convertTableMetadataToTable,
} from "@/types/v1/schemaEditor";
import { keyBy } from "@/utils";

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
  });

  for (const column of tableEdit.columnList) {
    tableMetadata.columns.push(transformColumnEditToMetadata(column));
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
  });
};

export const rebuildEditableSchemas = (
  originalSchemas: Schema[],
  schemas: SchemaMetadata[],
  schemaConfigList: SchemaConfig[]
): Schema[] => {
  const tag = [
    "rebuildEditableSchemas",
    `[${originalSchemas.map((s) => s.tableList.length)}]`,
    `[${schemas.map((s) => s.tables.length)}]`,
    `[${schemaConfigList.map((sc) => sc.tableConfigs.length)}]`,
  ].join("--");
  console.debug("go!", tag);
  console.time(tag);
  const editableSchemas = cloneDeep(originalSchemas);

  console.time("loop-1");
  const schemaMetadataByName = keyBy(schemas, (s) => s.name);
  const schemaConfigByName = keyBy(schemaConfigList, (sc) => sc.name);
  for (let i = 0; i < editableSchemas.length; i++) {
    const editableSchema = editableSchemas[i];
    const schema = schemaMetadataByName.get(editableSchema.name);
    if (!schema) {
      editableSchema.status = "dropped";
      continue;
    }

    const schemaConfig =
      schemaConfigByName.get(editableSchema.name) ??
      SchemaConfig.fromPartial({});

    for (const editableTable of editableSchema.tableList) {
      const table = schema.tables.find(
        (table) => table.name === editableTable.name
      );
      if (!table) {
        editableTable.status = "dropped";
        continue;
      }

      editableTable.userComment = table.userComment;
      editableTable.comment = table.comment;

      editableTable.config =
        schemaConfig.tableConfigs.find(
          (tableConfig) => tableConfig.name === editableTable.name
        ) ?? TableConfig.fromPartial({ name: editableTable.name });

      for (const editableColumn of editableTable.columnList) {
        editableColumn.config =
          editableTable.config.columnConfigs.find(
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
            editableTable.config.columnConfigs.find(
              (columnConfig) => columnConfig.name === column.name
            )
          );
          editableTable.columnList.push(newColumn);
        }
      }

      // Rebuild primary key from primary index.
      const primaryIndex = table.indexes.find((index) => index.primary);
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
  console.timeEnd("loop-1");

  console.time("loop-2");
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
  console.timeEnd("loop-2");

  console.time("loop-3");
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
  console.timeEnd("loop-3");

  console.timeEnd(tag);
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
