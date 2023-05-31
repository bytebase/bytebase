import { isEqual } from "lodash-es";
import { useSchemaEditorStore } from "@/store";

export const isTableChanged = (
  databaseId: string,
  schemaId: string,
  tableId: string
): boolean => {
  const editorStore = useSchemaEditorStore();
  const originSchema = editorStore.getOriginSchema(databaseId, schemaId);
  const schema = editorStore.getSchema(databaseId, schemaId);
  const table = schema?.tableList.find((table) => table.id === tableId);
  const originTable = originSchema?.tableList.find(
    (table) => table.id === tableId
  );

  const originForeignKeyList =
    originSchema?.foreignKeyList.filter((fk) => fk.tableId === table?.id) || [];
  const foreignKeyList =
    schema?.foreignKeyList.filter((fk) => fk.tableId === table?.id) || [];

  return (
    !isEqual(originTable, table) ||
    !isEqual(originForeignKeyList, foreignKeyList)
  );
};
