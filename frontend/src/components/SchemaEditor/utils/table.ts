import { useSchemaEditorStore } from "@/store";
import { DatabaseId } from "@/types";
import { isEqual } from "lodash-es";

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
    .find((pk) => pk.table === tableName)
    ?.columnList.map((columnRef) => columnRef.value)
    .sort();
  const primaryKeyColumnList = schema?.primaryKeyList
    .find((pk) => pk.table === tableName)
    ?.columnList.map((columnRef) => columnRef.value)
    .sort();

  return (
    !isEqual(originTable, table) ||
    !isEqual(originPrimaryKeyColumnList, primaryKeyColumnList)
  );
};
