import { describe, expect, test } from "vitest";
import { MCPSetting_Capability } from "@/types/proto-es/v1/setting_service_pb";
import { hydrateWhilePristine, normalizeCapability } from "./MCPSection";

const { CAPABILITY_UNSPECIFIED, DISABLED, READ_ONLY, READ_WRITE } =
  MCPSetting_Capability;

describe("normalizeCapability", () => {
  test("unset renders as Read-write (server resolves absent to READ_WRITE)", () => {
    expect(normalizeCapability(CAPABILITY_UNSPECIFIED)).toBe(READ_WRITE);
  });

  test("known values render as themselves", () => {
    expect(normalizeCapability(DISABLED)).toBe(DISABLED);
    expect(normalizeCapability(READ_ONLY)).toBe(READ_ONLY);
    expect(normalizeCapability(READ_WRITE)).toBe(READ_WRITE);
  });

  test("unknown and reserved stored values render as Disabled (matching the /mcp fail-closed gate)", () => {
    // 2 is the reserved number (was METADATA_ONLY); 99 is any unknown value.
    expect(normalizeCapability(2 as MCPSetting_Capability)).toBe(DISABLED);
    expect(normalizeCapability(99 as MCPSetting_Capability)).toBe(DISABLED);
  });
});

describe("hydrateWhilePristine", () => {
  const readWrite = { mcpCapability: MCPSetting_Capability.READ_WRITE };
  const readOnly = { mcpCapability: MCPSetting_Capability.READ_ONLY };
  const disabled = { mcpCapability: MCPSetting_Capability.DISABLED };

  test("takes the fetched value when the admin has not touched the form", () => {
    expect(hydrateWhilePristine(readWrite, readWrite, disabled)).toEqual(
      disabled
    );
  });

  test("keeps an edit the admin made before the fetch landed", () => {
    // Rendered with the placeholder, admin picked READ_ONLY, then DISABLED
    // arrived from the server. Overwriting here loses the edit and clears the
    // dirty footer with it.
    expect(hydrateWhilePristine(readOnly, readWrite, disabled)).toEqual(
      readOnly
    );
  });

  test("is a no-op when the edit already matches what arrived", () => {
    expect(hydrateWhilePristine(disabled, readWrite, disabled)).toEqual(
      disabled
    );
  });
});
