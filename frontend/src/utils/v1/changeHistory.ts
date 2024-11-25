import { isEqual, isUndefined, orderBy, uniqBy } from "lodash-es";
import Long from "long";
import { t } from "@/plugins/i18n";
import { useDBSchemaV1Store, useDatabaseV1Store, useUserStore } from "@/store";
import type { ComposedDatabase } from "@/types";
import { EMPTY_ID, UNKNOWN_ID } from "@/types";
import type { AffectedTable } from "@/types/changeHistory";
import { EmptyAffectedTable } from "@/types/changeHistory";
import type { DatabaseSchema } from "@/types/proto/v1/database_service";
import {
  ChangeHistory,
  ChangeHistory_Type,
} from "@/types/proto/v1/database_service";
import { extractUserResourceName } from ".";
import { databaseV1Url, extractDatabaseResourceName } from "./database";

export const extractChangeHistoryUID = (name: string) => {
  const pattern = /(?:^|\/)changeHistories\/([^/]+)(?:$|\/)/;
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

export const isValidChangeHistoryName = (name: string | undefined) => {
  if (!name) {
    return false;
  }
  const uid = extractChangeHistoryUID(name);
  return uid && uid !== String(EMPTY_ID) && uid !== String(UNKNOWN_ID);
};

export const changeHistoryLinkRaw = (parent: string, uid: string) => {
  const { database } = extractDatabaseResourceName(parent);
  const composedDatabase = useDatabaseV1Store().getDatabaseByName(database);
  const path = [databaseV1Url(composedDatabase), "change-histories", uid].join(
    "/"
  );
  return path;
};

export const changeHistoryLink = (changeHistory: ChangeHistory): string => {
  const { name } = changeHistory;
  return changeHistoryLinkRaw(name, extractChangeHistoryUID(name));
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

export const stringifyAffectedTable = (affectedTable: AffectedTable) => {
  const { schema, table } = affectedTable;
  if (schema !== "") {
    return `${schema}.${table}`;
  }
  return table;
};

export const getAffectedTableDisplayName = (affectedTable: AffectedTable) => {
  if (isEqual(affectedTable, EmptyAffectedTable)) {
    return t("change-history.all-tables");
  }

  let name = stringifyAffectedTable(affectedTable);
  if (affectedTable.dropped) {
    name = `${name}(deleted)`;
  }
  return name;
};

export const getHistoryChangeType = (type: ChangeHistory_Type) => {
  switch (type) {
    case ChangeHistory_Type.BASELINE:
    case ChangeHistory_Type.MIGRATE:
    case ChangeHistory_Type.MIGRATE_SDL:
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
    schema: schema?.schema,
    schemaSize: Long.fromNumber(
      new TextEncoder().encode(schema?.schema).length
    ),
    version: "Latest version",
  });
};

export const creatorOfChangeHistory = (history: ChangeHistory) => {
  const email = extractUserResourceName(history.creator);
  return useUserStore().getUserByEmail(email);
};
