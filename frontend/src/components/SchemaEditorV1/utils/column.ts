import { isEqual, isUndefined } from "lodash-es";
import { useSchemaEditorV1Store } from "@/store";
import { Column, Table, Schema } from "@/types/v1/schemaEditor";

export const isColumnChanged = (
  parentName: string,
  schema: Schema,
  table: Table,
  column: Column
): boolean => {
  const schemaEditorV1Store = useSchemaEditorV1Store();

  const originTable = schemaEditorV1Store.getOriginTable(
    parentName,
    schema.id,
    table.id
  );

  const originColumn = originTable?.columnList.find(
    (col) => col.id === column.id
  );
  if (!isEqual(column, originColumn)) {
    return true;
  }

  const isPrimaryKey = table?.primaryKey.columnIdList.includes(column.id);
  const isPrimaryKeyOrigin = originTable?.primaryKey.columnIdList.includes(
    column.id
  );

  const originForeignKey = originTable?.foreignKeyList.find(
    (fk) => fk.tableId === table?.id && fk.columnIdList.includes(column.id)
  );
  const foreignKey = table?.foreignKeyList.find(
    (fk) => fk.tableId === table?.id && fk.columnIdList.includes(column.id)
  );
  const originIndex = originForeignKey?.columnIdList.findIndex(
    (colId) => colId === column.id
  );
  const originForeignKeyColumn = isUndefined(originIndex)
    ? undefined
    : originForeignKey?.referencedColumnIdList[originIndex];
  const index = foreignKey?.columnIdList.findIndex(
    (colId) => colId === column.id
  );
  const foreignKeyColumn = isUndefined(index)
    ? undefined
    : foreignKey?.referencedColumnIdList[index];

  if (
    !isEqual(isPrimaryKey, isPrimaryKeyOrigin) ||
    !isEqual(foreignKeyColumn, originForeignKeyColumn)
  ) {
    return true;
  }

  const originSchema = schemaEditorV1Store.getOriginSchema(
    parentName,
    schema.id
  );
  if (!originSchema) {
    return true;
  }

  const columnConfig = schemaEditorV1Store.getColumnConfig(
    schema,
    table.name,
    column.name
  );
  const originColumnConfig = schemaEditorV1Store.getColumnConfig(
    originSchema,
    table.name,
    column.name
  );
  return !isEqual(columnConfig, originColumnConfig);
};
