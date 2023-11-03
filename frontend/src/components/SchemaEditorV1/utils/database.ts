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

export const fetchSchemaListByDatabaseName = async ({
  database,
  skipCache = true,
}: {
  database: string;
  skipCache?: boolean;
}) => {
  const schemaEditorV1Store = useSchemaEditorV1Store();
  const db = useDatabaseV1Store().getDatabaseByName(database);
  const databaseMetadata =
    await useDBSchemaV1Store().getOrFetchDatabaseMetadata({
      database,
      skipCache,
    });
  const schemaList = convertSchemaMetadataList(
    databaseMetadata.schemas,
    databaseMetadata.schemaConfigs
  );
  if (schemaList.length === 0 && db.instanceEntity.engine === Engine.MYSQL) {
    schemaList.push(
      convertSchemaMetadataToSchema(SchemaMetadata.fromPartial({}), "normal")
    );
  }

  schemaEditorV1Store.resourceMap.database.set(database, {
    database: db,
    schemaList,
    originSchemaList: cloneDeep(schemaList),
  });
  return schemaList;
};
