import { isEqual } from "lodash-es";
import { useSchemaEditorStore } from "@/store";
import { DatabaseId } from "@/types";

export const isSchemaChanged = (
  databaseId: DatabaseId,
  schemaId: string
): boolean => {
  const editorStore = useSchemaEditorStore();
  const originSchema = editorStore.getOriginSchema(databaseId, schemaId);
  const schema = editorStore.getSchema(databaseId, schemaId);

  return !isEqual(originSchema, schema);
};
