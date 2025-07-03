import type { Permission } from "./permission";
import PERMISSION_DATA from "./permission.yaml";

export const PERMISSIONS: Permission[] = PERMISSION_DATA.permissions;

export * from "./permission";

export * from "./role";
