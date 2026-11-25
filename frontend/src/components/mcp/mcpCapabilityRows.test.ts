// @vitest-environment node
import { describe, expect, test } from "vitest";
import { MCPSetting_Capability } from "@/types/proto-es/v1/setting_service_pb";
import {
  isRowServed,
  MCP_CAPABILITY_CATEGORIES,
  MCP_CAPABILITY_ROWS,
  servedRows,
} from "./mcpCapabilityRows";
import { isServingMode, MCP_CAPABILITY_CHOICES } from "./mcpPolicy";

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
    ).toEqual([
      "read-databases",
      "read-changes",
      "read-workspace",
      "read-policies",
    ]);
    expect(
      servedRows(MCPSetting_Capability.READ_WRITE).map((row) => row.id)
    ).toEqual([
      "read-databases",
      "read-changes",
      "read-workspace",
      "read-policies",
      "run-statements",
      "export",
      "saved-queries",
      "propose",
      "approve",
      "data-access",
      "instances",
      "workload-identities",
      "policies",
    ]);
  });

  test("rows run reads first, then by category in the design's order", () => {
    const rank = (row: (typeof MCP_CAPABILITY_ROWS)[number]) => [
      row.tier === "read" ? 0 : 1,
      MCP_CAPABILITY_CATEGORIES.indexOf(row.category),
    ];
    const ranks = MCP_CAPABILITY_ROWS.map(rank);
    const sorted = [...ranks].sort((a, b) => a[0] - b[0] || a[1] - b[1]);
    expect(ranks).toEqual(sorted);
    expect(MCP_CAPABILITY_CATEGORIES).toEqual([
      "databases",
      "changes",
      "workspace",
      "policies",
    ]);
  });

  test("each tier has one read row per category, so no read spans two", () => {
    expect(
      MCP_CAPABILITY_ROWS.filter((row) => row.tier === "read").map(
        (row) => row.category
      )
    ).toEqual([...MCP_CAPABILITY_CATEGORIES]);
  });

  // isServingMode gates the ladder and the consent routing, and is the one mode
  // predicate the compiler does not force: a fourth serving mode breaks SERVES
  // and MCP_MODE_PRESENTATION, but leaves this answering "false" and sending the
  // session to the Disabled consent screen.
  test("a mode serves a session exactly when it serves a row", () => {
    for (const mode of MCP_CAPABILITY_CHOICES) {
      expect(isServingMode(mode), String(mode)).toBe(
        MCP_CAPABILITY_ROWS.some((row) => isRowServed(mode, row))
      );
    }
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

  test("row ids are unique, so a row cannot key another row's copy", () => {
    const ids = MCP_CAPABILITY_ROWS.map((row) => row.id);
    expect(new Set(ids).size).toBe(ids.length);
  });
});
