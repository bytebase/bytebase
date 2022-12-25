import { isEqual } from "lodash-es";
import { useSchemaEditorStore } from "@/store";
import { DatabaseId } from "@/types";
import { ForeignKey } from "@/types/schemaEditor/atomType";

export const isTableChanged = (
  databaseId: DatabaseId,
  schemaName: string,
  tableName: string
): boolean => {
  const editorStore = useSchemaEditorStore();
  const originSchema = editorStore.getOriginSchema(databaseId, schemaName);
  const schema = editorStore.getSchema(databaseId, schemaName);
  const table = schema?.tableList.find((table) => table.newName === tableName);
  const originTable = originSchema?.tableList.find(
    (item) => item.oldName === table?.oldName
  );
  const originPrimaryKeyColumnList = originSchema?.primaryKeyList
    .find((pk) => pk.table === table?.oldName)
    ?.columnList.map((columnRef) => columnRef.value)
    .sort();
  const primaryKeyColumnList = schema?.primaryKeyList
    .find((pk) => pk.table === tableName)
    ?.columnList.map((columnRef) => columnRef.value)
    .sort();
  const originForeignKeyList =
    originSchema?.foreignKeyList.filter((pk) => pk.table === table?.oldName) ||
    [];
  const foreignKeyList =
    schema?.foreignKeyList.filter((pk) => pk.table === tableName) || [];

  return (
    !isEqual(originTable, table) ||
    !isEqual(originPrimaryKeyColumnList, primaryKeyColumnList) ||
    !isEqualForeignKeyList(originForeignKeyList, foreignKeyList)
  );
};

export const isEqualForeignKeyList = (
  originForeignKeyList: ForeignKey[],
  foreignKeyList: ForeignKey[]
) => {
  if (originForeignKeyList.length !== foreignKeyList.length) {
    return false;
  }
  for (let i = 0; i < foreignKeyList.length; i++) {
    if (!isEqualForeignKey(originForeignKeyList[i], foreignKeyList[i])) {
      return false;
    }
  }
  return true;
};

export const isEqualForeignKey = (
  originForeignKey?: ForeignKey,
  foreignKey?: ForeignKey
) => {
  return isEqual(
    {
      columnList: originForeignKey?.columnList.map(
        (columnRef) => columnRef.value
      ),
      referencedColumns: originForeignKey?.referencedColumns,
    },
    {
      columnList: foreignKey?.columnList.map((columnRef) => columnRef.value),
      referencedColumns: foreignKey?.referencedColumns,
    }
  );
};
