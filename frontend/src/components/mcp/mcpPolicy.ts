import { Eye, type LucideIcon, PencilLine, Unplug } from "lucide-react";
import type { BadgeProps } from "@/components/ui/badge";
import {
  type MCPSetting,
  MCPSetting_Capability,
} from "@/types/proto-es/v1/setting_service_pb";

export type MCPMode =
  | MCPSetting_Capability.DISABLED
  | MCPSetting_Capability.READ_ONLY
  | MCPSetting_Capability.READ_WRITE;

/**
 * A mode that admits MCP sessions. Anything describing what a session may do —
 * the capability ladder, the consent disclosure, the masking toggle — takes
 * this rather than MCPMode, so "there is a session to describe" is checked by
 * the compiler instead of by a comparison at each call site.
 */
export type MCPServingMode =
  | MCPSetting_Capability.READ_ONLY
  | MCPSetting_Capability.READ_WRITE;

/**
 * How a mode identifies itself wherever it appears: the locale-key stem, the
 * glyph, and the chip variant.
 *
 * One row per mode rather than parallel tables, because the identity has to be
 * the same on the settings page and on the consent page — an admin who picked
 * the eye is shown the eye when the session asks to connect. The variant is the
 * only place a mode's color appears; the selector cards stay neutral so the
 * accent keeps meaning "selected".
 */
export const MCP_MODE_PRESENTATION: Record<
  MCPMode,
  { key: string; icon: LucideIcon; badge: BadgeProps["variant"] }
> = {
  [MCPSetting_Capability.DISABLED]: {
    key: "disabled",
    icon: Unplug,
    badge: "destructive",
  },
  [MCPSetting_Capability.READ_ONLY]: {
    key: "read-only",
    icon: Eye,
    badge: "success",
  },
  [MCPSetting_Capability.READ_WRITE]: {
    key: "read-write",
    icon: PencilLine,
    badge: "warning",
  },
};

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
 * Whether a mode admits MCP sessions at all.
 *
 * Stated by the modes it admits rather than as "not Disabled", because the
 * caller's mode is often not yet chosen: an unreadable stored ceiling leaves
 * the editor with no pick, and a negation would count that absence as serving.
 */
export const isServingMode = (
  capability: MCPSetting_Capability | undefined
): capability is MCPServingMode =>
  capability === MCPSetting_Capability.READ_ONLY ||
  capability === MCPSetting_Capability.READ_WRITE;

/**
 * The locale key for one of a mode's strings, assembled in one place so the
 * product and the copy test cannot disagree about its shape.
 */
/** The collapsed disclosure line for a mode that serves a session. */
export const mcpSummaryKey = (mode: MCPServingMode): string =>
  `settings.mcp.ladder.summary.${MCP_MODE_PRESENTATION[mode].key}`;

export const mcpModeKey = (
  mode: MCPMode,
  part: "title" | "caption" | "best-for"
): string =>
  `settings.mcp.policy.mode.${MCP_MODE_PRESENTATION[mode].key}.${part}`;

/**
 * What the consent page can truthfully tell someone about to approve a client.
 *
 * Only `mode` is a policy it can disclose. The other two are the ways it can
 * fail to hold one, kept apart because the remedy differs: retry, or reload and
 * then find an admin (BOT-106).
 */
export type ConsentCeiling =
  /** Carries the response, so the disclosure cannot be rendered without it. */
  | { kind: "mode"; setting: MCPSetting }
  /** Actuator info did not provide a policy. The policy is not known to be anything. */
  | { kind: "unknown" }
  /** A stored ceiling this build has no wording for, whatever wrote it. */
  | { kind: "undisclosable" };

/**
 * Reads the MCP setting in actuator info — or its absence — into that state.
 *
 * The check is local and total: a capability this bundle can name is one it has
 * copy for. That covers a value nothing resolved (CAPABILITY_UNSPECIFIED), the
 * reserved 2, and a tier a newer release wrote, without asking the server which
 * of the three it is.
 */
export const readConsentCeiling = (
  setting: MCPSetting | undefined
): ConsentCeiling => {
  if (!setting) {
    return { kind: "unknown" };
  }
  if (!isMCPMode(setting.capability)) {
    return { kind: "undisclosable" };
  }
  return { kind: "mode", setting };
};
