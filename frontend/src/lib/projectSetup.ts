export type ResourceAvailability = "loading" | "present" | "empty" | "unknown";
export type ProjectSetupStep =
  | "transferDatabases"
  | "connect"
  | "createDatabase";

export type ProjectSetupState = {
  isDefault: boolean;
  instance: ResourceAvailability;
  projectInstance: ResourceAvailability;
  transferableDatabase: ResourceAvailability;
  canConnectInstance: boolean;
  canTransferDatabases: boolean;
  canCreateDatabase: boolean;
};

export function getProjectSetupSteps(
  input: ProjectSetupState
): ProjectSetupStep[] {
  if (input.isDefault) return [];
  if (input.canTransferDatabases && input.transferableDatabase === "loading") {
    return [];
  }

  const steps: ProjectSetupStep[] = [];
  const canOfferTransfer =
    input.canTransferDatabases && input.transferableDatabase === "present";
  if (canOfferTransfer) steps.push("transferDatabases");
  if (input.projectInstance === "empty" && input.canConnectInstance) {
    steps.push("connect");
  } else if (input.projectInstance === "present" && input.canCreateDatabase) {
    steps.push("createDatabase");
  }
  return steps;
}
