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
      permission: Permission;
    }
  | {
      kind: "feature-unavailable";
    };

interface RequestRoleButtonStateArgs {
  readonly projectName?: string;
  readonly projectReady: boolean;
  readonly allowRequestRole: boolean;
  readonly canSetIamPolicy: boolean;
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
  canSetIamPolicy,
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

  if (canSetIamPolicy) {
    return {
      visible: true,
      disabledReason: {
        kind: "can-grant-access-directly",
        permission: "bb.projects.setIamPolicy",
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
