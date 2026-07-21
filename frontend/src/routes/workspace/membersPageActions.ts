import type { Permission } from "@/types";
import type { Project } from "@/types/proto-es/v1/project_service_pb";

interface PermissionGuardConfig {
  readonly permissions: Permission[];
  readonly project?: Project;
}

export const getSetIamPolicyPermissionGuardConfig = (
  project?: Project
): PermissionGuardConfig => {
  if (project) {
    return {
      permissions: ["bb.projects.setIamPolicy"],
      project,
    };
  }

  return {
    permissions: ["bb.workspaces.setIamPolicy"],
  };
};
