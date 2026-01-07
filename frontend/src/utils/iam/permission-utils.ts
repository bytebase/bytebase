import { type ComposedDatabase } from "@/types";
import type { Project } from "@/types/proto-es/v1/project_service_pb";
import { hasProjectPermissionV2 } from "./permission";

export const hasPermissionToCreateChangeDatabaseIssueInProject = (
  project: Project
) => {
  return (
    hasProjectPermissionV2(project, "bb.issues.create") &&
    hasProjectPermissionV2(project, "bb.plans.create") &&
    hasProjectPermissionV2(project, "bb.rollouts.create") &&
    hasProjectPermissionV2(project, "bb.sheets.create")
  );
};

export const hasPermissionToCreateChangeDatabaseIssue = (
  database: ComposedDatabase
) => {
  return hasPermissionToCreateChangeDatabaseIssueInProject(
    database.projectEntity
  );
};
