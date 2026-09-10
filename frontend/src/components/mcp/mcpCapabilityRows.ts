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
 * Between them these eight titles claim to cover every READ and WRITE method,
 * and nothing checks that: a method annotated into either class is served the
 * moment it is annotated, whether or not a row names it. Reclassifying an RPC
 * therefore means rereading these rows, which is the rule recorded under
 * Metadata and API conventions in the root AGENTS.md.
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
 * The tiers a mode serves. The bundle's copy of the ceiling the gate evaluates:
 * Read-only serves the reads, Read-write serves both, Disabled serves nothing
 * and renders no list at all.
 */
export const servedTiers = (mode: MCPMode): readonly MCPCapabilityTier[] => {
  switch (mode) {
    case MCPSetting_Capability.READ_ONLY:
      return ["read"];
    case MCPSetting_Capability.READ_WRITE:
      return ["read", "write"];
    default:
      return [];
  }
};

export const isRowServed = (mode: MCPMode, row: MCPCapabilityRow): boolean =>
  servedTiers(mode).includes(row.tier);

export const servedRows = (mode: MCPMode): readonly MCPCapabilityRow[] =>
  MCP_CAPABILITY_ROWS.filter((row) => isRowServed(mode, row));

/**
 * The tier a row closes, or undefined when the row is not the last of its
 * tier. Deriving the divider from the list means a row added to a tier cannot
 * land below that tier's "stops here" line.
 */
export const tierClosedBy = (index: number): MCPCapabilityTier | undefined => {
  const row = MCP_CAPABILITY_ROWS[index];
  if (!row) {
    return undefined;
  }
  return MCP_CAPABILITY_ROWS[index + 1]?.tier === row.tier
    ? undefined
    : row.tier;
};
