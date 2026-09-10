import { describe, expect, test } from "vitest";
import { MCPSetting_Capability } from "@/types/proto-es/v1/setting_service_pb";
import {
  isRowServed,
  MCP_CAPABILITY_ROWS,
  servedRows,
  servedTiers,
  tierClosedBy,
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
    }
    expect(servedTiers(MCPSetting_Capability.READ_ONLY)).toEqual(["read"]);
    expect(servedTiers(MCPSetting_Capability.READ_WRITE)).toEqual([
      "read",
      "write",
    ]);
  });

  // A tier's rows must be contiguous: otherwise a "stops here" divider would
  // land above a row of the tier it closes.
  test("each tier occupies one contiguous run and is closed exactly once", () => {
    const dividers = MCP_CAPABILITY_ROWS.map((_, index) => tierClosedBy(index));
    expect(dividers.filter(Boolean)).toEqual(["read", "write"]);
    expect(dividers.at(-1)).toBe("write");
    expect(dividers[2]).toBe("read");
  });

  test("row ids are unique, so a row cannot key another row's copy", () => {
    const ids = MCP_CAPABILITY_ROWS.map((row) => row.id);
    expect(new Set(ids).size).toBe(ids.length);
  });
});
