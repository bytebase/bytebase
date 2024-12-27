import { isEqual, isUndefined, orderBy, uniqBy } from "lodash-es";
import Long from "long";
import { t } from "@/plugins/i18n";
import { useDBSchemaV1Store, useDatabaseV1Store } from "@/store";
import type { ComposedDatabase } from "@/types";
import { EMPTY_ID, UNKNOWN_ID } from "@/types";
import { type AffectedTable, EmptyAffectedTable } from "@/types";
import { Changelog_Type } from "@/types/proto/v1/database_service";
import { Changelog } from "@/types/proto/v1/database_service";
import { databaseV1Url, extractDatabaseResourceName } from "./database";

export const extractChangelogUID = (name: string) => {
  const pattern = /(?:^|\/)changelogs\/([^/]+)(?:$|\/)/;
  const matches = name.match(pattern);
  return matches?.[1] ?? "";
};

export const extractDatabaseNameAndChangelogUID = (changelogName: string) => {
  const parts = changelogName.split("/changelogs/");
  if (parts.length !== 2) {
    throw new Error("Invalid changelog name");
  }
  return {
    databaseName: parts[0],
    changelogUID: extractChangelogUID(changelogName),
  };
};

export const isValidChangelogName = (name: string | undefined) => {
  if (!name) {
    return false;
  }
  const uid = extractChangelogUID(name);
  return uid && uid !== String(EMPTY_ID) && uid !== String(UNKNOWN_ID);
};

export const changelogLink = (changelog: Changelog): string => {
  const { changelogUID } = extractDatabaseNameAndChangelogUID(changelog.name);
  const { database } = extractDatabaseResourceName(changelog.name);
  const composedDatabase = useDatabaseV1Store().getDatabaseByName(database);
  return [databaseV1Url(composedDatabase), "changelogs", changelogUID].join(
    "/"
  );
};

export const getAffectedTablesOfChangelog = (
  changelog: Changelog
): AffectedTable[] => {
  const { databaseName } = extractDatabaseNameAndChangelogUID(changelog.name);
  const database = useDatabaseV1Store().getDatabaseByName(databaseName);
  const metadata = useDBSchemaV1Store().getDatabaseMetadata(database.name);
  return uniqBy(
    orderBy(
      changelog.changedResources?.databases
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

export const mockLatestChangelog = (
  database: ComposedDatabase,
  schema: string = ""
) => {
  return Changelog.fromPartial({
    name: `${database.name}/changelogs/${UNKNOWN_ID}`,
    schema: schema,
    schemaSize: Long.fromNumber(new TextEncoder().encode(schema).length),
  });
};

export const getChangelogChangeType = (type: Changelog_Type) => {
  switch (type) {
    case Changelog_Type.BASELINE:
    case Changelog_Type.MIGRATE:
    case Changelog_Type.MIGRATE_SDL:
    case Changelog_Type.MIGRATE_GHOST:
      return "DDL";
    case Changelog_Type.DATA:
      return "DML";
    default:
      return "-";
  }
};
