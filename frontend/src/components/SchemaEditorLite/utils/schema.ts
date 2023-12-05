import { isEqual } from "lodash-es";
import { useSchemaEditorV1Store } from "@/store";

export const isSchemaChanged = (
  parentName: string,
  schemaId: string
): boolean => {
  const schemaEditorV1Store = useSchemaEditorV1Store();
  const editorSchema = schemaEditorV1Store.getSchema(parentName, schemaId);
  const originSchema = schemaEditorV1Store.getOriginSchema(
    parentName,
    schemaId
  );
  return !isEqual(originSchema, editorSchema);
};
