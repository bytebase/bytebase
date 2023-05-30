import { isEqual, isUndefined } from "lodash-es";
import { useSchemaEditorStore } from "@/store";

export const isColumnChanged = (
  databaseId: string,
  schemaId: string,
  tableId: string,
  columnId: string
): boolean => {
  const editorStore = useSchemaEditorStore();
  const originSchema = editorStore.getOriginSchema(databaseId, schemaId);
  const schema = editorStore.getSchema(databaseId, schemaId);
  const table = schema?.tableList.find((table) => table.id === tableId);
  const originTable = originSchema?.tableList.find(
    (table) => table.id === tableId
  );
  const column = table?.columnList.find((column) => column.id === columnId);
  const originColumn = originTable?.columnList.find(
    (column) => column.id === columnId
  );

  const isPrimaryKey = table?.primaryKey.columnIdList.includes(columnId);
  const isPrimaryKeyOrigin =
    originTable?.primaryKey.columnIdList.includes(columnId);

  const originForeignKey = originSchema?.foreignKeyList.find(
    (fk) => fk.tableId === table?.id && fk.columnIdList.includes(columnId)
  );
  const foreignKey = schema?.foreignKeyList.find(
    (fk) => fk.tableId === table?.id && fk.columnIdList.includes(columnId)
  );
  const originIndex = originForeignKey?.columnIdList.findIndex(
    (column) => column === columnId
  );
  const originForeignKeyColumn = isUndefined(originIndex)
    ? undefined
    : originForeignKey?.referencedColumnIdList[originIndex];
  const index = foreignKey?.columnIdList.findIndex(
    (column) => column === columnId
  );
  const foreignKeyColumn = isUndefined(index)
    ? undefined
    : foreignKey?.referencedColumnIdList[index];

  return (
    !isEqual(column, originColumn) ||
    !isEqual(isPrimaryKey, isPrimaryKeyOrigin) ||
    !isEqual(foreignKeyColumn, originForeignKeyColumn)
  );
};
