import {
  getProjectSetupSteps,
  type ProjectSetupState,
  type ResourceAvailability,
} from "@/lib/projectSetup";
import { leadsWithQuery } from "@/modules/workspace-setup-guide/scenarios";
import type { GuideScenarioId } from "@/modules/workspace-setup-guide/types";

export type { ResourceAvailability } from "@/lib/projectSetup";

export type ProjectHomeNextStep =
  | "connect"
  | "transferDatabases"
  | "createDatabase"
  | "changeDatabase"
  | "queryDatabase"
  | "members";

export type ProjectHomeNextStepInput = ProjectSetupState & {
  database: ResourceAvailability;
  change: "loading" | "not-started" | "proposed" | "rolled-out" | "unknown";
  membership: "unknown" | "solo" | "has-collaborators";
  canCreateChange: boolean;
  canQueryDatabase: boolean;
  canManageMembers: boolean;
  // The setup wizard's answer, known only in the browser that ran it.
  goal?: GuideScenarioId;
};

export function getProjectHomeNextSteps(
  input: ProjectHomeNextStepInput
): ProjectHomeNextStep[] {
  if (
    input.isDefault ||
    input.change === "rolled-out" ||
    input.database === "loading" ||
    (input.database === "present" && input.change === "loading")
  ) {
    return [];
  }

  const steps: ProjectHomeNextStep[] = [];
  if (input.database === "empty") {
    steps.push(...getProjectSetupSteps(input));
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
