import { ComposedDatabase } from "@/types";
import { User } from "@/types/proto/v1/auth_service";
import { hasProjectPermissionV2 } from "./permission";

export const hasPermissionToCreateRequestGrantIssue = (
  database: ComposedDatabase,
  user: User
) => {
  return hasProjectPermissionV2(
    database.projectEntity,
    user,
    "bb.issues.create"
  );
};

export const hasPermissionToCreateChangeDatabaseIssue = (
  database: ComposedDatabase,
  user: User
) => {
  return (
    hasProjectPermissionV2(database.projectEntity, user, "bb.issues.create") &&
    hasProjectPermissionV2(database.projectEntity, user, "bb.plans.create") &&
    hasProjectPermissionV2(database.projectEntity, user, "bb.plans.create")
  );
};
