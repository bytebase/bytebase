import dayjs from "dayjs";
import slug from "slug";
import { t } from "@/plugins/i18n";
import { EMPTY_ID, UNKNOWN_ID, type ComposedIssue } from "@/types";
import { Issue, Issue_Type } from "@/types/proto/v1/issue_service";
import type { Plan } from "@/types/proto/v1/plan_service";
import type { Rollout } from "@/types/proto/v1/rollout_service";
import { Task_Type } from "@/types/proto/v1/rollout_service";

export const issueV1Slug = (issue: Issue) => {
  return [slug(issue.title), extractIssueUID(issue.name)].join("-");
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

export const flattenSpecList = (plan: Plan | undefined) => {
  return plan?.steps.flatMap((step) => step.specs) || [];
};

export const isDatabaseChangeRelatedIssue = (issue: ComposedIssue): boolean => {
  return (
    Boolean(issue.rollout) &&
    flattenTaskV1List(issue.rolloutEntity).some((task) => {
      const DATABASE_RELATED_TASK_TYPE_LIST = [
        Task_Type.DATABASE_CREATE,
        Task_Type.DATABASE_SCHEMA_BASELINE,
        Task_Type.DATABASE_SCHEMA_UPDATE,
        Task_Type.DATABASE_SCHEMA_UPDATE_GHOST_SYNC,
        Task_Type.DATABASE_SCHEMA_UPDATE_GHOST_CUTOVER,
        Task_Type.DATABASE_SCHEMA_UPDATE_SDL,
        Task_Type.DATABASE_DATA_UPDATE,
      ];
      return DATABASE_RELATED_TASK_TYPE_LIST.includes(task.type);
    })
  );
};

export const isGrantRequestIssue = (issue: ComposedIssue): boolean => {
  return issue.type === Issue_Type.GRANT_REQUEST;
};

export const isDatabaseDataExportIssue = (issue: ComposedIssue): boolean => {
  return issue.type === Issue_Type.DATABASE_DATA_EXPORT;
};

export const generateIssueTitle = (
  type:
    | "bb.issue.database.schema.update"
    | "bb.issue.database.data.update"
    | "bb.issue.database.data.export"
    | "bb.issue.grant.request.querier"
    | "bb.issue.grant.request.exporter",
  databaseNameList: string[],
  isOnlineMode = false
) => {
  // Create a user friendly default issue name
  const parts: string[] = [];
  if (databaseNameList.length === 0) {
    parts.push(`[All databases]`);
  } else if (databaseNameList.length === 1) {
    parts.push(`[${databaseNameList[0]}]`);
  } else {
    parts.push(`[${databaseNameList.length} databases]`);
  }
  if (type.startsWith("bb.issue.database")) {
    if (isOnlineMode) {
      parts.push("Online schema change");
    } else {
      parts.push(
        type === "bb.issue.database.schema.update"
          ? t("issue.title.edit-schema")
          : type === "bb.issue.database.data.update"
            ? t("issue.title.change-data")
            : t("issue.title.export-data")
      );
    }
  } else {
    parts.push(
      type === "bb.issue.grant.request.querier"
        ? t("issue.title.request-querier-role")
        : t("issue.title.request-exporter-role")
    );
  }
  const datetime = dayjs().format("@MM-DD HH:mm");
  const tz = "UTC" + dayjs().format("ZZ");
  parts.push(`${datetime} ${tz}`);

  return parts.join(" ");
};
