import { describe, expect, test } from "vitest";
import { MCPSetting_Capability } from "@/types/proto-es/v1/setting_service_pb";
import {
  isRowServed,
  MCP_CAPABILITY_ROWS,
  MCP_CAPABILITY_TIERS,
  rowsInTier,
  servedRows,
} from "./mcpCapabilityRows";

describe("mcpCapabilityRows", () => {
  // Each mode is a PREFIX of the one list. It is what lets a single ladder show
  // all three modes at once, and what a custom policy would later select over.
  test("every mode serves a prefix of the list", () => {
    for (const mode of [
      MCPSetting_Capability.DISABLED,
      MCPSetting_Capability.READ_ONLY,
      MCPSetting_Capability.READ_WRITE,
    ] as const) {
      const served = servedRows(mode);
      expect(served).toEqual(MCP_CAPABILITY_ROWS.slice(0, served.length));
    }
  });

  test("the served set grows with the mode and never shrinks a read row", () => {
    expect(servedRows(MCPSetting_Capability.DISABLED)).toHaveLength(0);
    expect(
      servedRows(MCPSetting_Capability.READ_ONLY).map((row) => row.id)
    ).toEqual(["read-schemas", "read-data", "read-workflow"]);
    // Read-write serves exactly the reads Read-only serves, which is why both
    // summaries may share the same read phrase.
    expect(servedRows(MCPSetting_Capability.READ_WRITE).slice(0, 3)).toEqual(
      servedRows(MCPSetting_Capability.READ_ONLY)
    );
    expect(servedRows(MCPSetting_Capability.READ_WRITE)).toHaveLength(
      MCP_CAPABILITY_ROWS.length
    );
  });

  test("read-only serves no write row, so no write tag can appear under it", () => {
    for (const row of MCP_CAPABILITY_ROWS) {
      expect(isRowServed(MCPSetting_Capability.READ_ONLY, row)).toBe(
        row.tier === "read"
      );
      expect(isRowServed(MCPSetting_Capability.DISABLED, row)).toBe(false);
      expect(isRowServed(MCPSetting_Capability.READ_WRITE, row)).toBe(true);
    }
  });

  // The ladder renders one list per tier, in MCP_CAPABILITY_TIERS order. This
  // pins that doing so reproduces the ordered list exactly — which is both the
  // contiguity of each tier and the position of its "stops here" divider.
  test("rendering tier by tier reproduces the one ordered list", () => {
    expect(
      MCP_CAPABILITY_TIERS.flatMap((tier) => [...rowsInTier(tier)])
    ).toEqual([...MCP_CAPABILITY_ROWS]);
    expect(MCP_CAPABILITY_TIERS).toEqual(["read", "write"]);
    // Every tier carries rows, so no divider is emitted for an empty list.
    for (const tier of MCP_CAPABILITY_TIERS) {
      expect(rowsInTier(tier).length).toBeGreaterThan(0);
    }
  });

  test("row ids are unique, so a row cannot key another row's copy", () => {
    const ids = MCP_CAPABILITY_ROWS.map((row) => row.id);
    expect(new Set(ids).size).toBe(ids.length);
  });
});
