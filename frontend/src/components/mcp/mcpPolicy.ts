import { MCPSetting_Capability } from "@/types/proto-es/v1/setting_service_pb";
import type { MCPInfo } from "@/types/proto-es/v1/workspace_service_pb";

export type MCPMode =
  | MCPSetting_Capability.DISABLED
  | MCPSetting_Capability.READ_ONLY
  | MCPSetting_Capability.READ_WRITE;

/**
 * The ceilings an admin picks between, least to most capable. The bundle's copy
 * of the serving table the gate evaluates (`mcpServingClasses`,
 * backend/api/v1/mcp_gate.go).
 */
export const MCP_CAPABILITY_CHOICES: readonly MCPMode[] = [
  MCPSetting_Capability.DISABLED,
  MCPSetting_Capability.READ_ONLY,
  MCPSetting_Capability.READ_WRITE,
];

export const isMCPMode = (
  capability: MCPSetting_Capability
): capability is MCPMode =>
  MCP_CAPABILITY_CHOICES.some((choice) => choice === capability);

/**
 * What the consent page can truthfully tell someone about to approve a client.
 *
 * Only `mode` is a policy it can disclose. The other two are the ways it can
 * fail to hold one, kept apart because the remedy differs: retry, or reload and
 * then find an admin (BOT-106).
 */
export type ConsentCeiling =
  /** Carries the response, so the disclosure cannot be rendered without it. */
  | { kind: "mode"; info: MCPInfo }
  /** GetMCPInfo failed or timed out. The policy is not known to be anything. */
  | { kind: "unknown" }
  /** A stored ceiling this build has no wording for, whatever wrote it. */
  | { kind: "undisclosable" };

/**
 * Reads a GetMCPInfo response — or its absence — into that state.
 *
 * The check is local and total: a capability this bundle can name is one it has
 * copy for. That covers a value nothing resolved (CAPABILITY_UNSPECIFIED), the
 * reserved 2, and a tier a newer release wrote, without asking the server which
 * of the three it is.
 */
export const readConsentCeiling = (
  info: MCPInfo | undefined
): ConsentCeiling => {
  if (!info) {
    return { kind: "unknown" };
  }
  if (!isMCPMode(info.capability)) {
    return { kind: "undisclosable" };
  }
  return { kind: "mode", info };
};
