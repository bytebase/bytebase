import IAM_ACL_DATA from "./acl.yaml";
import PERMISSION_DATA from "./permission.yaml";

interface Role {
  name: string;
  permissions: string[];
}

export const SYSTEM_ROLES: Role[] = IAM_ACL_DATA.roles;

export const WORKSPACE_PERMISSIONS: string[] =
  PERMISSION_DATA.permissions.workspacePermissions;

export const PROJECT_PERMISSIONS: string[] =
  PERMISSION_DATA.permissions.projectPermissions;
