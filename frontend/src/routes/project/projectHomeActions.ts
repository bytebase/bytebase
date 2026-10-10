import type { Permission } from "@/types/iam";

export type DatabaseAvailability = "loading" | "present" | "empty" | "unknown";
export type ProjectHomeAction =
  | "plans"
  | "query"
  | "createDatabase"
  | "access"
  | "issues"
  | "instances"
  | "databases"
  | "members"
  | "gitops";

export type ProjectHomeActionInput = {
  availability: DatabaseAvailability;
  instanceAvailability: DatabaseAvailability;
  isDefault: boolean;
  accessGrantsAvailable: boolean;
  hasPermission: (permission: Permission) => boolean;
};

export function getProjectHomeActions({
  availability,
  instanceAvailability,
  isDefault,
  hasPermission,
}: ProjectHomeActionInput): ProjectHomeAction[] {
  const actions: ProjectHomeAction[] = [];
  const canListDatabases = hasPermission("bb.databases.list");
  const canListPlans =
    canListDatabases && hasPermission("bb.plans.list") && !isDefault;
  const canListInstances = hasPermission("bb.instances.list") && !isDefault;
  const canManageMembers =
    !isDefault &&
    hasPermission("bb.projects.getIamPolicy") &&
    hasPermission("bb.projects.setIamPolicy");
  const canCreateDatabase =
    canListDatabases &&
    hasPermission("bb.instances.list") &&
    hasPermission("bb.issues.create") &&
    hasPermission("bb.plans.create") &&
    hasPermission("bb.sheets.create");

  if (availability === "empty") {
    if (instanceAvailability === "present" && canCreateDatabase) {
      actions.push("createDatabase");
    } else if (instanceAvailability === "empty" && canListInstances) {
      actions.push("instances");
    }
    if (canManageMembers) actions.push("members");
    if (actions.length === 0 && canListInstances) actions.push("instances");
    if (actions.length === 0 && canListDatabases) actions.push("databases");
    return actions;
  }
  if (canListPlans && availability === "present") {
    actions.push("plans");
  }
  if (
    availability === "present" &&
    canListDatabases &&
    hasPermission("bb.projects.getIamPolicy")
  ) {
    actions.push("query");
  }
  if (actions.length === 0 && canListDatabases) {
    actions.push("databases");
  }
  if (actions.length === 0 && canListInstances) {
    actions.push("instances");
  }
  return actions;
}

export function getProjectHomeAvailableActions(
  input: ProjectHomeActionInput
): ProjectHomeAction[] {
  const actions = getProjectHomeActions(input);
  const { hasPermission, isDefault } = input;
  if (
    input.availability === "empty" &&
    input.instanceAvailability === "present" &&
    hasPermission("bb.databases.list") &&
    hasPermission("bb.instances.list") &&
    hasPermission("bb.issues.create") &&
    hasPermission("bb.plans.create") &&
    hasPermission("bb.sheets.create") &&
    !actions.includes("createDatabase")
  ) {
    actions.push("createDatabase");
  }
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
    input.accessGrantsAvailable &&
    hasPermission("bb.accessGrants.list") &&
    !isDefault &&
    !actions.includes("access")
  ) {
    actions.push("access");
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
