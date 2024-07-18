import type { ComposedDatabase, ComposedProject, ComposedUser } from "@/types";
import { hasProjectPermissionV2 } from "./permission";

export const hasPermissionToCreateRequestGrantIssueInProject = (
  project: ComposedProject,
  user: ComposedUser
) => {
  return hasProjectPermissionV2(project, user, "bb.issues.create");
};
export const hasPermissionToCreateRequestGrantIssue = (
  database: ComposedDatabase,
  user: ComposedUser
) => {
  return hasPermissionToCreateRequestGrantIssueInProject(
    database.projectEntity,
    user
  );
};

export const hasPermissionToCreateChangeDatabaseIssueInProject = (
  project: ComposedProject,
  user: ComposedUser
) => {
  return (
    hasProjectPermissionV2(project, user, "bb.issues.create") &&
    hasProjectPermissionV2(project, user, "bb.plans.create") &&
    hasProjectPermissionV2(project, user, "bb.rollouts.create")
  );
};
export const hasPermissionToCreateChangeDatabaseIssue = (
  database: ComposedDatabase,
  user: ComposedUser
) => {
  return hasPermissionToCreateChangeDatabaseIssueInProject(
    database.projectEntity,
    user
  );
};

export const hasPermissionToCreateDataExportIssueInProject = (
  project: ComposedProject,
  user: ComposedUser
) => {
  return (
    hasProjectPermissionV2(project, user, "bb.issues.create") &&
    hasProjectPermissionV2(project, user, "bb.plans.create") &&
    hasProjectPermissionV2(project, user, "bb.rollouts.create")
  );
};
export const hasPermissionToCreateDataExportIssue = (
  database: ComposedDatabase,
  user: ComposedUser
) => {
  return hasPermissionToCreateDataExportIssueInProject(
    database.projectEntity,
    user
  );
};

export const hasPermissionToCreatePlanInProject = (
  project: ComposedProject,
  user: ComposedUser
) => {
  return hasProjectPermissionV2(project, user, "bb.plans.create");
};
export const hasPermissionToCreateReviewIssueIssue = (
  database: ComposedDatabase,
  user: ComposedUser
) => {
  return hasPermissionToCreatePlanInProject(database.projectEntity, user);
};
