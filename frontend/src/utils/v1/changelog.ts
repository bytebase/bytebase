import { create } from "@bufbuild/protobuf";
import { t } from "@/plugins/i18n";
import { useDatabaseV1Store } from "@/store";
import type { ComposedDatabase } from "@/types";
import { UNKNOWN_ID } from "@/types";
import type { Changelog } from "@/types/proto-es/v1/database_service_pb";
import {
  Changelog_Type,
  ChangelogSchema,
} from "@/types/proto-es/v1/database_service_pb";
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
  return uid && uid !== String(UNKNOWN_ID);
};

export const changelogLink = (changelog: Changelog): string => {
  const { changelogUID } = extractDatabaseNameAndChangelogUID(changelog.name);
  const { database } = extractDatabaseResourceName(changelog.name);
  const composedDatabase = useDatabaseV1Store().getDatabaseByName(database);
  return [databaseV1Url(composedDatabase), "changelogs", changelogUID].join(
    "/"
  );
};

export const mockLatestChangelog = (
  database: ComposedDatabase,
  schema: string = ""
) => {
  return create(ChangelogSchema, {
    name: `${database.name}/changelogs/${UNKNOWN_ID}`,
    schema: schema,
    schemaSize: BigInt(new TextEncoder().encode(schema).length),
  });
};

export const getChangelogChangeType = (type: Changelog_Type) => {
  switch (type) {
    case Changelog_Type.SDL:
      return "SDL";
    case Changelog_Type.MIGRATE:
      return t("changelog.type.migrate");
    case Changelog_Type.BASELINE:
      return t("common.baseline");
    default:
      return "-";
  }
};
