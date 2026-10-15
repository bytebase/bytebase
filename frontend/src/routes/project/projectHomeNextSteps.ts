import { leadsWithQuery } from "@/modules/workspace-setup-guide/scenarios";
import type { GuideScenarioId } from "@/modules/workspace-setup-guide/types";

export type ResourceAvailability = "loading" | "present" | "empty" | "unknown";

export type ProjectHomeNextStep =
  | "connect"
  | "transferDatabases"
  | "createDatabase"
  | "changeDatabase"
  | "queryDatabase"
  | "members";

export type ProjectHomeNextStepInput = {
  database: ResourceAvailability;
  instance: ResourceAvailability;
  transferableDatabase: ResourceAvailability;
  change: "loading" | "not-started" | "rolled-out" | "unknown";
  membership: "unknown" | "solo" | "has-collaborators";
  isDefault: boolean;
  canConnectInstance: boolean;
  canTransferDatabases: boolean;
  canCreateDatabase: boolean;
  canCreateChange: boolean;
  canQueryDatabase: boolean;
  canManageMembers: boolean;
  // The setup wizard's answer, known only in the browser that ran it.
  goal?: GuideScenarioId;
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
    input.change === "rolled-out" ||
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
    if (!input.canQueryDatabase) {
      steps.push("changeDatabase");
    } else if (leadsWithQuery(input.goal)) {
      steps.push("queryDatabase", "changeDatabase");
    } else {
      steps.push("changeDatabase", "queryDatabase");
    }
  }

  if (input.membership === "solo" && input.canManageMembers) {
    steps.push("members");
  }
  return steps;
}
