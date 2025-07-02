import dayjs from "dayjs";
import slug from "slug";
import { t } from "@/plugins/i18n";
import { EMPTY_ID, UNKNOWN_ID, type ComposedIssue } from "@/types";
import { Issue_Type } from "@/types/proto-es/v1/issue_service_pb";
import type { Rollout } from "@/types/proto-es/v1/rollout_service_pb";
import { Task_Type } from "@/types/proto-es/v1/rollout_service_pb";

export const issueV1Slug = (name: string, title: string = "issue") => {
  return [slug(title), extractIssueUID(name)].join("-");
};

export const extractIssueUID = (name: string) => {
  const pattern = /(?:^|\/)issues\/(\d+)(?:$|\/)/;
  const matches = name.match(pattern);
  return matches?.[1] ?? "";
};

export const isValidIssueName = (name: string | undefined) => {
  if (!name) {
    return false;
  }
  const uid = extractIssueUID(name);
  return uid && uid !== String(EMPTY_ID) && uid !== String(UNKNOWN_ID);
};

export const flattenTaskV1List = (rollout: Rollout | undefined) => {
  return rollout?.stages.flatMap((stage) => stage.tasks) || [];
};

const DATABASE_RELATED_TASK_TYPE_LIST = [
  Task_Type.DATABASE_CREATE,
  Task_Type.DATABASE_SCHEMA_UPDATE,
  Task_Type.DATABASE_SCHEMA_UPDATE_GHOST,
  Task_Type.DATABASE_SCHEMA_UPDATE_SDL,
  Task_Type.DATABASE_DATA_UPDATE,
];

export const isDatabaseChangeRelatedIssue = (issue: ComposedIssue): boolean => {
  return (
    Boolean(issue.rollout) &&
    flattenTaskV1List(issue.rolloutEntity).some((task) => {
      return DATABASE_RELATED_TASK_TYPE_LIST.includes(task.type);
    })
  );
};

export const isGrantRequestIssue = (issue: ComposedIssue): boolean => {
  return issue.type === Issue_Type.GRANT_REQUEST;
};

export const isDatabaseDataExportIssue = (issue: ComposedIssue): boolean => {
  return issue.type === Issue_Type.DATABASE_EXPORT;
};

export const generateIssueTitle = (
  type:
    | "bb.issue.database.schema.update"
    | "bb.issue.database.data.update"
    | "bb.issue.database.data.export"
    | "bb.issue.grant.request",
  databaseNameList?: string[],
  title?: string
) => {
  // Create a user friendly default issue name
  const parts: string[] = [];

  if (databaseNameList !== undefined) {
    if (databaseNameList.length === 0) {
      parts.push(`[All databases]`);
    } else if (databaseNameList.length === 1) {
      parts.push(`[${databaseNameList[0]}]`);
    } else {
      parts.push(`[${databaseNameList.length} databases]`);
    }
  }

  if (title) {
    parts.push(title);
  } else {
    if (type.startsWith("bb.issue.database")) {
      parts.push(
        type === "bb.issue.database.schema.update"
          ? t("issue.title.edit-schema")
          : type === "bb.issue.database.data.update"
            ? t("issue.title.change-data")
            : t("issue.title.export-data")
      );
    } else {
      parts.push(t("issue.title.request-role"));
    }
  }

  const datetime = dayjs().format("@MM-DD HH:mm");
  const tz = "UTC" + dayjs().format("ZZ");
  parts.push(`${datetime} ${tz}`);

  return parts.join(" ");
};
