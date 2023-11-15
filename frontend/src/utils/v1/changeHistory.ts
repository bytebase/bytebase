import { isEqual, isUndefined, orderBy, uniqBy } from "lodash-es";
import slug from "slug";
import { t } from "@/plugins/i18n";
import { useDBSchemaV1Store, useDatabaseV1Store } from "@/store";
import { ComposedDatabase, UNKNOWN_ID } from "@/types";
import { AffectedTable, EmptyAffectedTable } from "@/types/changeHistory";
import {
  ChangeHistory,
  ChangeHistory_Type,
  DatabaseSchema,
} from "@/types/proto/v1/database_service";
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

export const changeHistoryLinkRaw = (
  parent: string,
  uid: string,
  version: string
) => {
  const { instance, database } = extractDatabaseResourceName(parent);
  const path = [
    "instances",
    encodeURIComponent(instance),
    "databases",
    encodeURIComponent(database),
    "changeHistories",
    changeHistorySlug(uid, version),
  ].join("/");
  return `/${path}`;
};

export const changeHistoryLink = (changeHistory: ChangeHistory): string => {
  const { name, uid, version } = changeHistory;
  return changeHistoryLinkRaw(name, uid, version);
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

export const getAffectedTableDisplayName = (affectedTable: AffectedTable) => {
  if (isEqual(affectedTable, EmptyAffectedTable)) {
    return t("change-history.all-tables");
  }

  const { schema, table, dropped } = affectedTable;
  let name = table;
  if (schema !== "") {
    name = `${schema}.${table}`;
  }
  if (dropped) {
    name = `${name} (deleted)`;
  }
  return name;
};

export const getHistoryChangeType = (type: ChangeHistory_Type) => {
  switch (type) {
    case ChangeHistory_Type.BASELINE:
    case ChangeHistory_Type.MIGRATE:
    case ChangeHistory_Type.MIGRATE_SDL:
    case ChangeHistory_Type.BRANCH:
    case ChangeHistory_Type.MIGRATE_GHOST:
      return "DDL";
    case ChangeHistory_Type.DATA:
      return "DML";
    default:
      return "-";
  }
};

export const mockLatestSchemaChangeHistory = (
  database: ComposedDatabase,
  schema: DatabaseSchema | undefined = undefined
) => {
  return ChangeHistory.fromPartial({
    name: `${database.name}/changeHistories/${UNKNOWN_ID}`,
    uid: String(UNKNOWN_ID),
    schema: schema?.schema,
    version: "Latest version",
    description: "the latest schema of database",
  });
};
