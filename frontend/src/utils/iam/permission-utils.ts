import type { ComposedDatabase, Permission } from "@/types";
import type { Project } from "@/types/proto-es/v1/project_service_pb";
import { hasProjectPermissionV2 } from "./permission";

export const PERMISSIONS_FOR_DATABASE_CHANGE_ISSUE: Permission[] = [
  "bb.plans.create",
  "bb.sheets.create",
];

export const PERMISSIONS_FOR_DATABASE_EXPORT_ISSUE: Permission[] = [
  "bb.issues.create",
  ...PERMISSIONS_FOR_DATABASE_CHANGE_ISSUE,
];

export const PERMISSIONS_FOR_DATABASE_CREATE_ISSUE: Permission[] = [
  ...PERMISSIONS_FOR_DATABASE_EXPORT_ISSUE,
];

export const hasPermissionToCreateChangeDatabaseIssueInProject = (
  project: Project
) => {
  return PERMISSIONS_FOR_DATABASE_CHANGE_ISSUE.every((p) =>
    hasProjectPermissionV2(project, p)
  );
};

export const hasPermissionToCreateChangeDatabaseIssue = (
  database: ComposedDatabase
) => {
  return hasPermissionToCreateChangeDatabaseIssueInProject(
    database.projectEntity
  );
};
