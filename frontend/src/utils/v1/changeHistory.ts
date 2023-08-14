import { isUndefined, orderBy, uniqBy } from "lodash-es";
import slug from "slug";
import { useDBSchemaV1Store, useDatabaseV1Store } from "@/store";
import { AffectedTable } from "@/types/changeHistory";
import { ChangeHistory } from "@/types/proto/v1/database_service";
import { extractDatabaseResourceName } from "./database";

export const extractChangeHistoryUID = (name: string) => {
  const pattern = /(?:^|\/)(?:changeHistories|migrations)\/([^/]+)(?:$|\/)/;
  const matches = name.match(pattern);
  return matches?.[1] ?? "";
};

export const extractDatabaseNameAndChangeHistoryUID = (
  changeHistoryName: string
) => {
  const parts = changeHistoryName.split("/changeHistories/");
  if (parts.length !== 2) {
    throw new Error("Invalid change history name");
  }
  return {
    databaseName: parts[0],
    uid: extractChangeHistoryUID(parts[1]),
  };
};

export const changeHistorySlug = (uid: string, version: string): string => {
  return [slug(version), uid].join("-");
};

export const changeHistoryLink = (changeHistory: ChangeHistory): string => {
  const { name, uid, version } = changeHistory;
  const { instance, database } = extractDatabaseResourceName(name);
  const parent = `instances/${instance}/databases/${database}`;
  return `/${parent}/changeHistories/${changeHistorySlug(uid, version)}`;
};

export const getAffectedTablesOfChangeHistory = (
  changeHistory: ChangeHistory
): AffectedTable[] => {
  const { databaseName } = extractDatabaseNameAndChangeHistoryUID(
    changeHistory.name
  );
  const database = useDatabaseV1Store().getDatabaseByName(databaseName);
  const metadata = useDBSchemaV1Store().getDatabaseMetadata(database.name);
  return uniqBy(
    orderBy(
      changeHistory.changedResources?.databases
        .find((db) => db.name === database.databaseName)
        ?.schemas.map((schema) => {
          return schema.tables.map((table) => {
            const dropped = isUndefined(
              metadata.schemas
                .find((s) => s.name === schema.name)
                ?.tables.find((t) => t.name === table.name)
            );
            return {
              schema: schema.name,
              table: table.name,
              dropped,
            };
          });
        })
        .flat() || [],
      ["dropped"]
    ),
    (affectedTable) => `${affectedTable.schema}.${affectedTable.table}`
  );
};
