import type { MCPMode } from "@/components/mcp/mcpPolicy";
import { MCPSetting_Capability } from "@/types/proto-es/v1/setting_service_pb";

/**
 * The two tiers a capability row can belong to. They are the product's words
 * for the READ and WRITE method classes the gate serves (`mcpServingClasses`,
 * backend/api/v1/mcp_gate.go), and a mode serves whole tiers — never part of
 * one.
 */
export type MCPCapabilityTier = "read" | "write";

/**
 * The categories, in list order. A row's category comes from the two questions
 * in D16 of docs/design/mcp-capability-ladder.md; it orders the list and is
 * never drawn.
 */
export const MCP_CAPABILITY_CATEGORIES = [
  "databases",
  "changes",
  "workspace",
  "policies",
] as const;

export type MCPCapabilityCategory = (typeof MCP_CAPABILITY_CATEGORIES)[number];

/**
 * The rows, by the id their copy is keyed under. A union rather than `string`
 * so the table and its consumers cannot drift to an id nothing else knows.
 * Locale coverage is not checked: these keys are built from the id, so a row
 * with no copy renders the raw key.
 */
export type MCPCapabilityRowId =
  | "read-databases"
  | "read-changes"
  | "read-workspace"
  | "read-policies"
  | "run-statements"
  | "export"
  | "saved-queries"
  | "propose"
  | "approve"
  | "data-access"
  | "instances"
  | "workload-identities"
  | "policies";

export interface MCPCapabilityRow {
  /** Keys the row's title under `settings.mcp.ladder.row.<id>.title`. */
  readonly id: MCPCapabilityRowId;
  readonly tier: MCPCapabilityTier;
  readonly category: MCPCapabilityCategory;
}

/**
 * One ordered list in which every mode is a prefix: read rows first, then
 * write rows, each tier in category order.
 *
 * Between them the rows, titles with their docs sub-items, claim to cover every
 * READ and WRITE method, and nothing checks that: a method annotated into either
 * class is served the moment it is annotated, whether or not a row names it.
 * Which methods each row stands for is NOT here. Reclassifying an RPC means
 * rereading the row table in docs/design/mcp-capability-ladder.md and updating
 * it in the same change.
 *
 * `run-statements` is the one row backed by no method: it names the statement
 * clamp (`mcp_sql_clamp.go`) that Read-write lifts, which is a real difference
 * between the modes that no method name shows. What no mode serves is not a row;
 * `MCPCapabilityList` renders it last among the refused rows.
 */
export const MCP_CAPABILITY_ROWS: readonly MCPCapabilityRow[] = [
  { id: "read-databases", tier: "read", category: "databases" },
  { id: "read-changes", tier: "read", category: "changes" },
  { id: "read-workspace", tier: "read", category: "workspace" },
  { id: "read-policies", tier: "read", category: "policies" },
  { id: "run-statements", tier: "write", category: "databases" },
  { id: "export", tier: "write", category: "databases" },
  { id: "saved-queries", tier: "write", category: "databases" },
  { id: "propose", tier: "write", category: "changes" },
  { id: "approve", tier: "write", category: "changes" },
  { id: "data-access", tier: "write", category: "changes" },
  { id: "instances", tier: "write", category: "workspace" },
  { id: "workload-identities", tier: "write", category: "workspace" },
  { id: "policies", tier: "write", category: "policies" },
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
 * The locale keys the ladder's copy is stored under, built in one place beside
 * the ids they are built from. These families are template-keyed, so
 * `check-i18n.mjs` exempts them by prefix and cannot report a missing line.
 */
export const mcpTierKey = (tier: MCPCapabilityTier): string =>
  `settings.mcp.ladder.tier.${tier}`;

export const mcpRowKey = (row: MCPCapabilityRow): string =>
  `settings.mcp.ladder.row.${row.id}.title`;
