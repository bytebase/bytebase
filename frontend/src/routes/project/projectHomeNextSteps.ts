export type ResourceAvailability = "loading" | "present" | "empty" | "unknown";

export type ProjectHomeNextStep =
  | "connect"
  | "transferDatabases"
  | "createDatabase"
  | "changeDatabase"
  | "members";

export type ProjectHomeNextStepInput = {
  database: ResourceAvailability;
  instance: ResourceAvailability;
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
  const canOfferTransfer =
    input.database === "empty" &&
    input.canTransferDatabases &&
    input.transferableDatabase === "present";
  if (
    input.isDefault ||
    input.change === "submitted" ||
    input.database === "loading" ||
    (input.database === "present" && input.change === "loading")
  ) {
    return [];
  }

  const steps: ProjectHomeNextStep[] = [];
  const transferIsLoading =
    input.canTransferDatabases && input.transferableDatabase === "loading";
  if (input.database === "empty" && !transferIsLoading) {
    if (canOfferTransfer) {
      steps.push("transferDatabases");
    }
    if (
      input.instance === "empty" &&
      !canOfferTransfer &&
      input.canConnectInstance
    ) {
      steps.push("connect");
    } else if (input.instance === "present") {
      if (input.canCreateDatabase) {
        steps.push("createDatabase");
      }
    }
  } else if (
    input.database === "present" &&
    (input.change === "not-started" || input.change === "unknown") &&
    input.canCreateChange
  ) {
    steps.push("changeDatabase");
  }

  if (input.membership === "solo" && input.canManageMembers) {
    steps.push("members");
  }
  return steps;
}
