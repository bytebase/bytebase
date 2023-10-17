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
  if (!isEqual(originTable, editorTable)) {
    return true;
  }

  if (editorTable && originTable) {
    const schema = schemaEditorV1Store.getSchema(parentName, schemaId);
    const originSchema = schemaEditorV1Store.getOriginSchema(
      parentName,
      schemaId
    );
    if (schema && originSchema) {
      const tableConfig = schemaEditorV1Store.getTableConfig(
        schema,
        editorTable.name
      );
      const originTableConfig = schemaEditorV1Store.getTableConfig(
        originSchema,
        originTable.name
      );
      if (!isEqual(tableConfig, originTableConfig)) {
        return true;
      }
    }
  }

  return false;
};
