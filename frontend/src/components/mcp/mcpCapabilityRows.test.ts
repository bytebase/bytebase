// @vitest-environment node
import { describe, expect, test } from "vitest";
import { MCPSetting_Capability } from "@/types/proto-es/v1/setting_service_pb";
import {
  isRowServed,
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
    ).toEqual(["read-schemas", "read-data", "read-workflow", "read-access"]);
    expect(
      servedRows(MCPSetting_Capability.READ_WRITE).map((row) => row.id)
    ).toEqual([
      "read-schemas",
      "read-data",
      "read-workflow",
      "read-access",
      "propose",
      "run-statements",
      "request-access",
      "approve",
      "masking",
      "add-instances",
      "manage",
    ]);
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
