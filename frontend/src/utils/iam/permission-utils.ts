import { ComposedDatabase, ComposedProject } from "@/types";
import { User } from "@/types/proto/v1/auth_service";
import { hasProjectPermissionV2 } from "./permission";

export const hasPermissionToCreateRequestGrantIssueInProject = (
  project: ComposedProject,
  user: User
) => {
  return hasProjectPermissionV2(project, user, "bb.issues.create");
};
export const hasPermissionToCreateRequestGrantIssue = (
  database: ComposedDatabase,
  user: User
) => {
  return hasPermissionToCreateRequestGrantIssueInProject(
    database.projectEntity,
    user
  );
};

export const hasPermissionToCreateChangeDatabaseIssueInProject = (
  project: ComposedProject,
  user: User
) => {
  return (
    hasProjectPermissionV2(project, user, "bb.issues.create") &&
    hasProjectPermissionV2(project, user, "bb.plans.create") &&
    hasProjectPermissionV2(project, user, "bb.rollouts.create")
  );
};
export const hasPermissionToCreateChangeDatabaseIssue = (
  database: ComposedDatabase,
  user: User
) => {
  return hasPermissionToCreateChangeDatabaseIssueInProject(
    database.projectEntity,
    user
  );
};
