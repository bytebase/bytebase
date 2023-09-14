import { cloneDeep } from "lodash-es";
import {
  useDBSchemaV1Store,
  useDatabaseV1Store,
  useSchemaEditorV1Store,
} from "@/store";
import { Engine } from "@/types/proto/v1/common";
import { SchemaMetadata } from "@/types/proto/v1/database_service";
import {
  convertSchemaMetadataList,
  convertSchemaMetadataToSchema,
} from "@/types/v1/schemaEditor";

export const fetchSchemaListByDatabaseName = async (
  databaseName: string,
  skipCache = false
) => {
  const schemaEditorV1Store = useSchemaEditorV1Store();
  const database = useDatabaseV1Store().getDatabaseByName(databaseName);
  const schemaMetadataList = await useDBSchemaV1Store().getOrFetchSchemaList(
    database.name,
    skipCache
  );
  const schemaList = convertSchemaMetadataList(schemaMetadataList);
  if (
    schemaList.length === 0 &&
    database.instanceEntity.engine === Engine.MYSQL
  ) {
    schemaList.push(
      convertSchemaMetadataToSchema(SchemaMetadata.fromPartial({}))
    );
  }

  schemaEditorV1Store.resourceMap.database.set(database.name, {
    database,
    schemaList: schemaList,
    originSchemaList: cloneDeep(schemaList),
  });
};
