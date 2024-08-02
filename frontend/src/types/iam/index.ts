import type { Permission } from "./permission";
import PERMISSION_DATA from "./permission.yaml";

export const WORKSPACE_PERMISSIONS: Permission[] =
  PERMISSION_DATA.permissions.workspacePermissions;

export const PROJECT_PERMISSIONS: Permission[] =
  PERMISSION_DATA.permissions.projectPermissions;

export * from "./permission";

export * from "./role";
