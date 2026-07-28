import { describe, expect, test } from "vitest";
import { WorkspaceProfileSetting_MCPCapability } from "@/types/proto-es/v1/setting_service_pb";
import { normalizeCapability } from "./MCPSection";

const {
  MCP_CAPABILITY_UNSPECIFIED,
  MCP_DISABLED,
  MCP_READ_ONLY,
  MCP_READ_WRITE,
} = WorkspaceProfileSetting_MCPCapability;

describe("normalizeCapability", () => {
  test("unset renders as Read-write (server resolves absent to READ_WRITE)", () => {
    expect(normalizeCapability(MCP_CAPABILITY_UNSPECIFIED)).toBe(
      MCP_READ_WRITE
    );
  });

  test("known values render as themselves", () => {
    expect(normalizeCapability(MCP_DISABLED)).toBe(MCP_DISABLED);
    expect(normalizeCapability(MCP_READ_ONLY)).toBe(MCP_READ_ONLY);
    expect(normalizeCapability(MCP_READ_WRITE)).toBe(MCP_READ_WRITE);
  });

  test("unknown and reserved stored values render as Disabled (matching the /mcp fail-closed gate)", () => {
    // 2 is the reserved number (was METADATA_ONLY); 99 is any unknown value.
    expect(
      normalizeCapability(2 as WorkspaceProfileSetting_MCPCapability)
    ).toBe(MCP_DISABLED);
    expect(
      normalizeCapability(99 as WorkspaceProfileSetting_MCPCapability)
    ).toBe(MCP_DISABLED);
  });
});
