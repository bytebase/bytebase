export type ResourceAvailability = "loading" | "present" | "empty" | "unknown";

export type ProjectHomeNextStep =
  | "connect"
  | "transferDatabases"
  | "createDatabase"
  | "changeDatabase"
  | "members";

export type ProjectHomeNextStepInput = {
  database: ResourceAvailability;
  instance: ResourceAvailability | "syncing";
  transferableDatabase: ResourceAvailability;
  change: "loading" | "not-started" | "submitted" | "unknown";
  membership: "unknown" | "solo" | "has-collaborators";
  isDefault: boolean;
  canConnectInstance: boolean;
  canTransferDatabases: boolean;
  canCreateDatabase: boolean;
  canCreateChange: boolean;
  canManageMembers: boolean;
};

export function getProjectHomeNextSteps(
  input: ProjectHomeNextStepInput
): ProjectHomeNextStep[] {
  if (
    input.isDefault ||
    input.change === "submitted" ||
    (input.database !== "empty" && input.database !== "present") ||
    (input.database === "present" && input.change !== "not-started") ||
    (input.database === "empty" &&
      input.instance === "present" &&
      input.canTransferDatabases &&
      input.transferableDatabase === "loading") ||
    (input.database === "empty" &&
      input.instance !== "empty" &&
      input.instance !== "present")
  ) {
    return [];
  }

  const steps: ProjectHomeNextStep[] = [];
  if (input.database === "empty") {
    if (input.instance === "empty" && input.canConnectInstance) {
      steps.push("connect");
    } else if (input.instance === "present") {
      if (
        input.canTransferDatabases &&
        input.transferableDatabase === "present"
      ) {
        steps.push("transferDatabases");
      }
      if (input.canCreateDatabase) {
        steps.push("createDatabase");
      }
    }
  } else if (input.change === "not-started" && input.canCreateChange) {
    steps.push("changeDatabase");
  }

  if (input.membership === "solo" && input.canManageMembers) {
    steps.push("members");
  }
  return steps;
}
