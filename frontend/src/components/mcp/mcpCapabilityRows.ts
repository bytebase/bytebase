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
 * Tier order, and with it the order the "stops here" dividers appear in. The
 * ladder walks the tiers and emits each one's rows followed by its divider, so
 * a row cannot land on the wrong side of the line that closes its tier.
 */
export const MCP_CAPABILITY_TIERS: readonly MCPCapabilityTier[] = [
  "read",
  "write",
];

/**
 * The bundle's copy of the ceiling the gate evaluates (`mcpServingClasses`,
 * backend/api/v1/mcp_gate.go).
 *
 * A table rather than a switch, because a switch is exhaustive over the modes
 * and not over the tiers: a comparison against the tiers that exist today
 * compiles unchanged against a widened union and answers "refused" for rows
 * nobody classified. Every cell here has to be filled in, so adding either a
 * mode or a tier fails to compile until someone decides what it serves.
 */
const SERVES: Record<MCPMode, Record<MCPCapabilityTier, boolean>> = {
  [MCPSetting_Capability.DISABLED]: { read: false, write: false },
  [MCPSetting_Capability.READ_ONLY]: { read: true, write: false },
  [MCPSetting_Capability.READ_WRITE]: { read: true, write: true },
};

export const isRowServed = (mode: MCPMode, row: MCPCapabilityRow): boolean =>
  SERVES[mode][row.tier];

export const servedRows = (mode: MCPMode): readonly MCPCapabilityRow[] =>
  MCP_CAPABILITY_ROWS.filter((row) => isRowServed(mode, row));

/**
 * The locale keys the ladder's copy is stored under, stated beside the ids they
 * are built from so the product and the copy test cannot disagree about their
 * shape. Every template-keyed family has one; a family without one is a family
 * where a rename leaves the test passing against the old shape.
 */
export const mcpTierKey = (
  tier: MCPCapabilityTier,
  part: "tier" | "stops"
): string => `settings.mcp.ladder.${part}.${tier}`;

export const mcpRowKey = (
  row: MCPCapabilityRow,
  part: "title" | "details"
): string => `settings.mcp.ladder.row.${row.id}.${part}`;

export const rowsInTier = (
  tier: MCPCapabilityTier
): readonly MCPCapabilityRow[] =>
  MCP_CAPABILITY_ROWS.filter((row) => row.tier === tier);
