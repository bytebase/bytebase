import type { Permission } from "@/types";

export const REQUEST_ROLE_REQUIRED_PERMISSIONS = [
  "bb.issues.create",
  "bb.roles.list",
] as const satisfies readonly Permission[];

export type RequestRoleButtonDisabledReason =
  | {
      kind: "loading";
    }
  | {
      kind: "allow-request-role-disabled";
    }
  | {
      kind: "can-grant-access-directly";
    }
  | {
      kind: "feature-unavailable";
    };

interface RequestRoleButtonStateArgs {
  readonly projectName?: string;
  readonly projectReady: boolean;
  readonly allowRequestRole: boolean;
  // True when the current user already holds every PROJECT_OWNER permission
  // (workspace- or project-scoped), so they can grant access directly and have
  // no reason to request a role. Mirrors the Vue `hasMissingPermission` gate,
  // which checks the full owner permission set rather than `setIamPolicy` alone.
  readonly hasFullProjectAccess: boolean;
  readonly hasRequestRoleFeature: boolean;
}

interface RequestRoleButtonState {
  readonly visible: boolean;
  readonly disabledReason?: RequestRoleButtonDisabledReason;
}

export const getRequestRoleButtonState = ({
  projectName,
  projectReady,
  allowRequestRole,
  hasFullProjectAccess,
  hasRequestRoleFeature,
}: RequestRoleButtonStateArgs): RequestRoleButtonState => {
  if (!projectName) {
    return {
      visible: false,
    };
  }

  if (!projectReady) {
    return {
      visible: true,
      disabledReason: {
        kind: "loading",
      },
    };
  }

  if (!allowRequestRole) {
    return {
      visible: true,
      disabledReason: {
        kind: "allow-request-role-disabled",
      },
    };
  }

  if (hasFullProjectAccess) {
    return {
      visible: true,
      disabledReason: {
        kind: "can-grant-access-directly",
      },
    };
  }

  if (!hasRequestRoleFeature) {
    return {
      visible: true,
      disabledReason: {
        kind: "feature-unavailable",
      },
    };
  }

  return {
    visible: true,
  };
};
