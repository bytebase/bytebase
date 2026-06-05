import i18n from "@/react/i18n";
import { getDatabaseByName } from "@/react/stores/app/databaseAccess";
import type { Revision } from "@/types/proto-es/v1/revision_service_pb";
import { Revision_Type } from "@/types/proto-es/v1/revision_service_pb";
import { databaseV1Url, extractDatabaseResourceName } from "./database";

export const extractRevisionUID = (name: string): string => {
  const pattern = /(?:^|\/)revisions\/([^/]+)(?:$|\/)/;
  const matches = name.match(pattern);
  return matches?.[1] ?? "";
};

export const revisionLink = (revision: Revision): string => {
  const parts = revision.name.split("/revisions/");
  if (parts.length !== 2) {
    return "";
  }
  const { database } = extractDatabaseResourceName(revision.name);
  const composedDatabase = getDatabaseByName(database);
  return `${databaseV1Url(composedDatabase)}/revisions/${parts[1]}`;
};

export const getRevisionType = (type: Revision_Type): string => {
  switch (type) {
    case Revision_Type.VERSIONED:
      return i18n.t("database.revision.type-versioned");
    case Revision_Type.DECLARATIVE:
      return i18n.t("database.revision.type-declarative");
    default:
      return "-";
  }
};

// Extract task link from taskRun resource name
// e.g., "projects/xxx/plans/yyy/rollout/stages/zzz/tasks/aaa/taskRuns/bbb" -> "/projects/xxx/plans/yyy/rollout/stages/zzz/tasks/aaa"
export const extractTaskLink = (taskRunName: string): string => {
  const parts = taskRunName.split("/taskRuns/");
  if (parts.length !== 2) {
    return "";
  }
  return `/${parts[0]}`;
};
