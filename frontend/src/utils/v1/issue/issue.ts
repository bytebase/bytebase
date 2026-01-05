import dayjs from "dayjs";
import slug from "slug";
import { t } from "@/plugins/i18n";
import {
  PROJECT_V1_ROUTE_ISSUE_DETAIL,
  PROJECT_V1_ROUTE_ISSUE_DETAIL_V1,
} from "@/router/dashboard/projectV1";
import { type ComposedIssue, EMPTY_ID, UNKNOWN_ID } from "@/types";
import { Issue_Type } from "@/types/proto-es/v1/issue_service_pb";
import type { Plan } from "@/types/proto-es/v1/plan_service_pb";
import type { Rollout } from "@/types/proto-es/v1/rollout_service_pb";
import { Task_Type } from "@/types/proto-es/v1/rollout_service_pb";
import { extractProjectResourceName } from "../project";

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

export const getRolloutFromPlan = (planName: string): string => {
  return `${planName}/rollout`;
};

export const flattenTaskV1List = (rollout: Rollout | undefined) => {
  return rollout?.stages.flatMap((stage) => stage.tasks) || [];
};

const DATABASE_RELATED_TASK_TYPE_LIST = [
  Task_Type.DATABASE_CREATE,
  Task_Type.DATABASE_MIGRATE,
];

export const isDatabaseChangeRelatedIssue = (issue: ComposedIssue): boolean => {
  return (
    Boolean(issue.plan) &&
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
    | "bb.issue.database.update"
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
        type === "bb.issue.database.update"
          ? t("issue.title.change-database")
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

/**
 * Determines whether an issue should use the new CICD layout route.
 *
 * Rules:
 * - Grant request issues: ALWAYS use new layout
 * - Create database issues: ALWAYS use new layout
 * - Data export issues: ALWAYS use new layout
 * - Database changing issues: Use new layout ONLY when enabledNewLayout is true
 */
export const shouldUseNewIssueLayout = (
  issue: { type?: Issue_Type; name?: string },
  plan?: Plan | { specs?: Array<{ config?: { case?: string } }> },
  enabledNewLayout = true
): boolean => {
  // Grant request issues always use new layout
  if (issue.type === Issue_Type.GRANT_REQUEST) {
    return true;
  }

  // Check if it's a create database or export data plan
  if (plan?.specs) {
    const isCreatingDatabasePlan = plan.specs.every(
      (spec) => spec.config?.case === "createDatabaseConfig"
    );
    const isExportDataPlan = plan.specs.every(
      (spec) => spec.config?.case === "exportDataConfig"
    );

    if (isCreatingDatabasePlan || isExportDataPlan) {
      return true;
    }
  }

  // Database export issue type always uses new layout
  if (issue.type === Issue_Type.DATABASE_EXPORT) {
    return true;
  }

  // For database changing issues, respect the layout preference
  return enabledNewLayout;
};

/**
 * Gets the appropriate route name and params for an issue based on its type and layout settings.
 *
 * @param issue - The issue object (must have name, and optionally type)
 * @param plan - Optional plan object to check for create database or export data specs
 * @param enabledNewLayout - Whether the new layout is enabled (default: true)
 * @returns Route configuration with name and params
 */
export const getIssueRoute = (
  issue: { type?: Issue_Type; name: string; title?: string },
  plan?: Plan | { specs?: Array<{ config?: { case?: string } }> },
  enabledNewLayout = true
): {
  name: string;
  params: { projectId: string; issueSlug?: string; issueId?: string };
} => {
  const projectId = extractProjectResourceName(issue.name);
  const useNewLayout = shouldUseNewIssueLayout(issue, plan, enabledNewLayout);

  if (useNewLayout) {
    // Use new CICD layout route with numeric issueId
    return {
      name: PROJECT_V1_ROUTE_ISSUE_DETAIL_V1,
      params: {
        projectId,
        issueId: extractIssueUID(issue.name),
      },
    };
  } else {
    // Use legacy route with issueSlug
    return {
      name: PROJECT_V1_ROUTE_ISSUE_DETAIL,
      params: {
        projectId,
        issueSlug: issueV1Slug(issue.name, issue.title || "issue"),
      },
    };
  }
};
