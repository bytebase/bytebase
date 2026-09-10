import type { MCPMode } from "@/components/mcp/mcpPolicy";
import { MCPSetting_Capability } from "@/types/proto-es/v1/setting_service_pb";

/**
 * The two tiers a capability row can belong to. They are the product's words
 * for the READ and WRITE method classes the gate serves (`mcpServingClasses`,
 * backend/api/v1/mcp_gate.go), and a mode serves whole tiers — never part of
 * one.
 */
export type MCPCapabilityTier = "read" | "write";

export interface MCPCapabilityRow {
  /** Keys the row's copy under `settings.mcp.ladder.row.<id>.*`. */
  readonly id: string;
  readonly tier: MCPCapabilityTier;
}

/**
 * One ordered list in which every mode is a prefix. Read rows first, then write
 * rows.
 *
 * Between them these titles claim to cover every READ and WRITE method, and
 * nothing checks that: a method annotated into either class is served the moment
 * it is annotated, whether or not a row names it. Reclassifying an RPC therefore
 * means rereading these rows, which is the rule recorded under Metadata and API
 * conventions in the root AGENTS.md.
 *
 * Which methods each row stands for is NOT here — this file carries only the
 * order and the tier. That mapping is the row table in
 * docs/design/mcp-capability-ladder.md, which is the artifact to read and to
 * update when a classification changes.
 *
 * `run-statements` is the one row backed by no method: it names the statement
 * clamp (`mcp_sql_clamp.go`) that Read-write lifts, which is a real difference
 * between the modes that no method name shows.
 */
export const MCP_CAPABILITY_ROWS: readonly MCPCapabilityRow[] = [
  { id: "read-schemas", tier: "read" },
  { id: "read-data", tier: "read" },
  { id: "read-workflow", tier: "read" },
  { id: "propose", tier: "write" },
  { id: "run-rollouts", tier: "write" },
  { id: "run-statements", tier: "write" },
  { id: "export", tier: "write" },
  { id: "manage", tier: "write" },
];

/**
 * Tier order, and with it the order the "stops here" dividers appear in. Each
 * tier renders as its own list, so a row cannot land on the wrong side of its
 * tier's divider.
 */
export const MCP_CAPABILITY_TIERS: readonly MCPCapabilityTier[] = [
  "read",
  "write",
];

/**
 * The bundle's copy of the ceiling the gate evaluates (`mcpServingClasses`,
 * backend/api/v1/mcp_gate.go).
 */
export const isRowServed = (mode: MCPMode, row: MCPCapabilityRow): boolean => {
  switch (mode) {
    case MCPSetting_Capability.READ_WRITE:
      return true;
    case MCPSetting_Capability.READ_ONLY:
      return row.tier === "read";
    default:
      return false;
  }
};

export const servedRows = (mode: MCPMode): readonly MCPCapabilityRow[] =>
  MCP_CAPABILITY_ROWS.filter((row) => isRowServed(mode, row));

export const rowsInTier = (
  tier: MCPCapabilityTier
): readonly MCPCapabilityRow[] =>
  MCP_CAPABILITY_ROWS.filter((row) => row.tier === tier);
