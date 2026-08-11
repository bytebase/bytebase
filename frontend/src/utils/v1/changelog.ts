import { create } from "@bufbuild/protobuf";
import { getDatabaseByName } from "@/stores/app/databaseAccess";
import { UNKNOWN_ID } from "@/types";
import type { Changelog } from "@/types/proto-es/v1/changelog_service_pb";
import { ChangelogSchema } from "@/types/proto-es/v1/changelog_service_pb";
import type { Database } from "@/types/proto-es/v1/database_service_pb";
import { databaseV1UrlWithSuffix } from "./database";

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
  const { changelogUID, databaseName } = extractDatabaseNameAndChangelogUID(
    changelog.name
  );
  const composedDatabase = getDatabaseByName(databaseName);
  return databaseV1UrlWithSuffix(
    composedDatabase,
    `/changelogs/${changelogUID}`
  );
};

export const mockLatestChangelog = (
  database: Database,
  schema: string = ""
) => {
  return create(ChangelogSchema, {
    name: `${database.name}/changelogs/${UNKNOWN_ID}`,
    schema: schema,
    schemaSize: BigInt(new TextEncoder().encode(schema).length),
  });
};
