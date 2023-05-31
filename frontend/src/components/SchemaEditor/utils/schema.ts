import { isEqual } from "lodash-es";
import { useSchemaEditorStore } from "@/store";

export const isSchemaChanged = (
  databaseId: string,
  schemaId: string
): boolean => {
  const editorStore = useSchemaEditorStore();
  const originSchema = editorStore.getOriginSchema(databaseId, schemaId);
  const schema = editorStore.getSchema(databaseId, schemaId);

  return !isEqual(originSchema, schema);
};
