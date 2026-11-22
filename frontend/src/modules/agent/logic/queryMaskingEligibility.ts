import { getMaskingSetup } from "./tools/maskingSetup";
import type { AgentQueryMaskingEligibility } from "./types";

export async function resolveQueryMaskingEligibility(
  workspaceName: string,
  database: string
): Promise<AgentQueryMaskingEligibility> {
  if (!/^workspaces\/(?!-$)[A-Za-z0-9_-]+$/.test(workspaceName))
    return { database, allowed: false };
  try {
    const setup = await getMaskingSetup({ workspaceName });
    const global = setup.steps.find((step) => step.id === "global-masking");
    const types = setup.steps.find((step) => step.id === "semantic-types");
    return {
      database,
      allowed:
        setup.features.masking === "available" &&
        global?.configure.status === "allowed" &&
        global.readAccess.status === "allowed" &&
        types?.readAccess.status === "allowed",
    };
  } catch {
    return { database, allowed: false };
  }
}
