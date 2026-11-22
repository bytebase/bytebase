import type { Permission } from "@/types";
import type { Project } from "@/types/proto-es/v1/project_service_pb";
import { PlanFeature } from "@/types/proto-es/v1/subscription_service_pb";

export const isQueryAccessGrantPermission = (permission: string): boolean =>
  permission === "bb.sql.select" || permission === "bb.sql.info";

export function getQueryAccessRequest(
  project:
    | Pick<Project, "allowJustInTimeAccess" | "allowRequestRole">
    | undefined,
  requiredPermissions: readonly string[]
):
  | {
      kind: "accessGrant" | "requestRole";
      requiredPermission: Permission;
      requiredFeature: PlanFeature;
    }
  | undefined {
  if (
    project?.allowJustInTimeAccess &&
    requiredPermissions.every(isQueryAccessGrantPermission)
  ) {
    return {
      kind: "accessGrant",
      requiredPermission: "bb.accessGrants.create",
      requiredFeature: PlanFeature.FEATURE_JIT,
    };
  }
  if (project?.allowRequestRole) {
    return {
      kind: "requestRole",
      requiredPermission: "bb.issues.create",
      requiredFeature: PlanFeature.FEATURE_REQUEST_ROLE_WORKFLOW,
    };
  }
  return undefined;
}
