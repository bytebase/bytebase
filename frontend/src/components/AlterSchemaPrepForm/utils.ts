import { useSchemaEditorV1Store } from "@/store";
import { DatabaseEdit } from "@/types";
import {
  checkHasSchemaChanges,
  diffSchema,
  mergeDiffResults,
} from "@/utils/schemaEditor/diffSchema";

// getDatabaseEditListWithSchemaEditor will calculate the diff between the origin schema and the current schema
// and return a list of DatabaseEdit. The DatabaseEdit will be only used to check if the schema has changes.
export const getDatabaseEditListWithSchemaEditor = () => {
  const schemaEditorV1Store = useSchemaEditorV1Store();

  const databaseEditList: DatabaseEdit[] = [];
  for (const databaseSchema of Array.from(
    schemaEditorV1Store.resourceMap["database"].values()
  )) {
    const database = databaseSchema.database;
    for (const schema of databaseSchema.schemaList) {
      const originSchema = databaseSchema.originSchemaList.find(
        (originSchema) => originSchema.id === schema.id
      );
      const diffSchemaResult = diffSchema(database.name, originSchema, schema);
      if (checkHasSchemaChanges(diffSchemaResult)) {
        const index = databaseEditList.findIndex(
          (edit) => String(edit.databaseId) === database.uid
        );
        if (index !== -1) {
          databaseEditList[index] = {
            databaseId: Number(database.uid),
            ...mergeDiffResults([diffSchemaResult, databaseEditList[index]]),
          };
        } else {
          databaseEditList.push({
            databaseId: Number(database.uid),
            ...diffSchemaResult,
          });
        }
      }
    }
  }
  return databaseEditList;
};
