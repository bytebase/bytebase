import { type ComposedDatabase } from "@/types";
import type { Project } from "@/types/proto-es/v1/project_service_pb";
import { hasProjectPermissionV2 } from "./permission";

export const hasPermissionToCreateRequestGrantIssue = (
  database: ComposedDatabase
) => {
  return hasProjectPermissionV2(database.projectEntity, "bb.issues.create");
};

export const hasPermissionToCreateChangeDatabaseIssueInProject = (
  project: Project
) => {
  return (
    hasProjectPermissionV2(project, "bb.issues.create") &&
    hasProjectPermissionV2(project, "bb.plans.create") &&
    hasProjectPermissionV2(project, "bb.rollouts.create")
  );
};

export const hasPermissionToCreateChangeDatabaseIssue = (
  database: ComposedDatabase
) => {
  return hasPermissionToCreateChangeDatabaseIssueInProject(
    database.projectEntity
  );
};

export const hasPermissionToCreateDataExportIssueInProject = (
  project: Project
) => {
  return (
    hasProjectPermissionV2(project, "bb.issues.create") &&
    hasProjectPermissionV2(project, "bb.plans.create") &&
    hasProjectPermissionV2(project, "bb.rollouts.create")
  );
};

export const hasPermissionToCreateDataExportIssue = (
  database: ComposedDatabase
) => {
  return hasPermissionToCreateDataExportIssueInProject(database.projectEntity);
};

export const hasPermissionToCreatePlanInProject = (project: Project) => {
  return hasProjectPermissionV2(project, "bb.plans.create");
};
