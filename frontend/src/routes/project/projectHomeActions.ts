import type { Permission } from "@/types/iam";

export type DatabaseAvailability = "loading" | "present" | "empty" | "unknown";
export type ProjectHomeAction =
  | "plans"
  | "query"
  | "access"
  | "issues"
  | "instances"
  | "databases"
  | "members"
  | "gitops";

export type ProjectHomeActionInput = {
  availability: DatabaseAvailability;
  isDefault: boolean;
  accessGrantsAvailable: boolean;
  hasPermission: (permission: Permission) => boolean;
};

export function getProjectHomeActions({
  availability,
  isDefault,
  accessGrantsAvailable,
  hasPermission,
}: ProjectHomeActionInput): ProjectHomeAction[] {
  const actions: ProjectHomeAction[] = [];
  const canListDatabases = hasPermission("bb.databases.list");
  const canListPlans =
    canListDatabases && hasPermission("bb.plans.list") && !isDefault;
  const canListIssues = hasPermission("bb.issues.list") && !isDefault;
  const canListInstances = hasPermission("bb.instances.list") && !isDefault;
  const canPrepare =
    canListPlans &&
    availability === "present" &&
    hasPermission("bb.plans.create");

  if (availability === "empty" && canListInstances) {
    actions.push("instances");
  }
  if (canListPlans && availability !== "empty") {
    actions.push("plans");
  }
  if (
    availability === "present" &&
    canListDatabases &&
    hasPermission("bb.projects.getIamPolicy")
  ) {
    actions.push("query");
  }
  if (
    accessGrantsAvailable &&
    hasPermission("bb.accessGrants.list") &&
    !isDefault
  ) {
    actions.push("access");
  }
  if (canListIssues && !canPrepare) {
    actions.push("issues");
  }
  if (actions.length === 0 && canListDatabases) {
    actions.push("databases");
  }
  if (actions.length === 0 && canListInstances) {
    actions.push("instances");
  }
  if (
    actions.length === 0 &&
    !isDefault &&
    hasPermission("bb.projects.getIamPolicy")
  ) {
    actions.push("members");
  }
  return actions;
}

export function getProjectHomeAvailableActions(
  input: ProjectHomeActionInput
): ProjectHomeAction[] {
  const actions = getProjectHomeActions(input);
  const { hasPermission, isDefault } = input;
  if (
    hasPermission("bb.databases.list") &&
    hasPermission("bb.plans.list") &&
    !isDefault &&
    !actions.includes("plans")
  ) {
    actions.push("plans");
  }
  if (
    hasPermission("bb.issues.list") &&
    !isDefault &&
    !actions.includes("issues")
  ) {
    actions.push("issues");
  }
  if (
    hasPermission("bb.instances.list") &&
    !isDefault &&
    !actions.includes("instances")
  ) {
    actions.push("instances");
  }
  if (hasPermission("bb.databases.list") && !actions.includes("databases")) {
    actions.push("databases");
  }
  if (
    hasPermission("bb.projects.getIamPolicy") &&
    !isDefault &&
    !actions.includes("members")
  ) {
    actions.push("members");
  }
  if (
    hasPermission("bb.databases.list") &&
    hasPermission("bb.workloadIdentities.list") &&
    !isDefault
  ) {
    actions.push("gitops");
  }
  return actions;
}
