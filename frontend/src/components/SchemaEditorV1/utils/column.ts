import { isEqual, isUndefined } from "lodash-es";
import { useSchemaEditorV1Store } from "@/store";

export const isColumnChanged = (
  parentName: string,
  schemaId: string,
  tableId: string,
  columnId: string
): boolean => {
  const schemaEditorV1Store = useSchemaEditorV1Store();
  const table = schemaEditorV1Store.getTable(parentName, schemaId, tableId);
  const originTable = schemaEditorV1Store.getOriginTable(
    parentName,
    schemaId,
    tableId
  );
  const column = table?.columnList.find((column) => column.id === columnId);
  const originColumn = originTable?.columnList.find(
    (column) => column.id === columnId
  );

  const isPrimaryKey = table?.primaryKey.columnIdList.includes(columnId);
  const isPrimaryKeyOrigin =
    originTable?.primaryKey.columnIdList.includes(columnId);

  const originForeignKey = originTable?.foreignKeyList.find(
    (fk) => fk.tableId === table?.id && fk.columnIdList.includes(columnId)
  );
  const foreignKey = table?.foreignKeyList.find(
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
