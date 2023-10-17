import { isEqual } from "lodash-es";
import { useSchemaEditorV1Store } from "@/store";

export const isTableChanged = (
  parentName: string,
  schemaId: string,
  tableId: string
): boolean => {
  const schemaEditorV1Store = useSchemaEditorV1Store();
  const editorTable = schemaEditorV1Store.getTable(
    parentName,
    schemaId,
    tableId
  );
  const originTable = schemaEditorV1Store.getOriginTable(
    parentName,
    schemaId,
    tableId
  );
  return !isEqual(originTable, editorTable);
};
