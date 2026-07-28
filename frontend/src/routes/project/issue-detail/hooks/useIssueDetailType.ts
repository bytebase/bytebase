import type { Issue } from "@/types/proto-es/v1/issue_service_pb";
import { Issue_Type } from "@/types/proto-es/v1/issue_service_pb";
import type { Plan } from "@/types/proto-es/v1/plan_service_pb";

export type IssueDetailType =
  | "DATABASE_CHANGE"
  | "CREATE_DATABASE"
  | "EXPORT_DATA"
  | "ROLE_GRANT"
  | "ACCESS_GRANT";

export const resolveIssueDetailType = (
  issue?: Issue,
  plan?: Plan
): IssueDetailType | undefined => {
  if (!issue) {
    return undefined;
  }

  if (issue.type === Issue_Type.ROLE_GRANT) {
    return "ROLE_GRANT";
  }
  if (issue.type === Issue_Type.ACCESS_GRANT) {
    return "ACCESS_GRANT";
  }

  const specs = plan?.specs ?? [];
  if (specs.length === 0) {
    return undefined;
  }

  if (specs.every((spec) => spec.config.case === "createDatabaseConfig")) {
    return "CREATE_DATABASE";
  }
  if (specs.every((spec) => spec.config.case === "exportDataConfig")) {
    return "EXPORT_DATA";
  }
  if (specs.some((spec) => spec.config.case === "changeDatabaseConfig")) {
    return "DATABASE_CHANGE";
  }

  return undefined;
};
