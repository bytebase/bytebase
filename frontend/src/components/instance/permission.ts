import type { Permission } from "@/types";
import type { Project } from "@/types/proto-es/v1/project_service_pb";
import {
  hasProjectPermissionV2,
  hasWorkspacePermissionV2,
} from "@/utils/iam/permission";

export const hasInstancePermission = (
  project: Project | undefined,
  permission: Permission
): boolean =>
  project
    ? hasProjectPermissionV2(project, permission)
    : hasWorkspacePermissionV2(permission);
