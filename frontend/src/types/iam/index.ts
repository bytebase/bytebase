import type { Permission } from "./permission";
import PERMISSION_DATA from "./permission.yaml";

interface PermissionYamlData {
  permissions: Permission[];
}

export const PERMISSIONS: Permission[] = (
  PERMISSION_DATA as unknown as PermissionYamlData
).permissions;

export * from "./permission";

export * from "./role";
