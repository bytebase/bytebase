import { isEqual } from "lodash-es";
import { useSchemaEditorContext } from "../common";

export const isSchemaChanged = (schemaId: string): boolean => {
  const { originalSchemas, editableSchemas } = useSchemaEditorContext();
  const originSchema = originalSchemas.value.find(
    (schema) => schema.id === schemaId
  );
  const schema = editableSchemas.value.find((schema) => schema.id === schemaId);
  return !isEqual(originSchema, schema);
};
