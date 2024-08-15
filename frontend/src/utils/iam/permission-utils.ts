import type { ComposedDatabase, ComposedProject } from "@/types";
import { hasProjectPermissionV2 } from "./permission";

const hasPermissionToCreateRequestGrantIssueInProject = (
  project: ComposedProject
) => {
  return hasProjectPermissionV2(project, "bb.issues.create");
};

export const hasPermissionToCreateRequestGrantIssue = (
  database: ComposedDatabase
) => {
  return hasPermissionToCreateRequestGrantIssueInProject(
    database.projectEntity
  );
};

export const hasPermissionToCreateChangeDatabaseIssueInProject = (
  project: ComposedProject
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
  project: ComposedProject
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

export const hasPermissionToCreatePlanInProject = (
  project: ComposedProject
) => {
  return hasProjectPermissionV2(project, "bb.plans.create");
};
