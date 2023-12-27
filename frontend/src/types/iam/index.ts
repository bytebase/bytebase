import { ProjectPermission, WorkspacePermission } from "./permission";
import PERMISSION_DATA from "./permission.yaml";

export const WORKSPACE_PERMISSIONS: WorkspacePermission[] =
  PERMISSION_DATA.permissions.workspacePermissions;

export const PROJECT_PERMISSIONS: ProjectPermission[] =
  PERMISSION_DATA.permissions.projectPermissions;

export * from "./permission";
