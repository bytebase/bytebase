import { create } from "@bufbuild/protobuf";
import { describe, expect, test } from "vitest";
import {
  MCPSetting_Capability,
  MCPSettingSchema,
} from "@/types/proto-es/v1/setting_service_pb";
import {
  isMCPMode,
  MCP_CAPABILITY_CHOICES,
  readConsentCeiling,
} from "./mcpPolicy";

describe("MCP_CAPABILITY_CHOICES", () => {
  // This list is the bundle's copy of the gate's serving table
  // (mcpServingClasses, backend/api/v1/mcp_gate.go), and it decides two things:
  // which ceilings the settings radio offers, and which ones the consent page
  // will disclose. A capability a release adds to the enum and forgets here
  // ships a settings page that cannot select it and a consent page that
  // withholds Allow on every workspace using it. Nothing on the Go side notices
  // a TypeScript omission, so the generated enum is the counterpart to check.
  test("covers every capability the enum offers", () => {
    const declared = Object.values(MCPSetting_Capability).filter(
      (value): value is MCPSetting_Capability => typeof value === "number"
    );
    // 0 is the absence of a ceiling, not a ceiling. The reserved 2 needs no
    // exclusion: a reserved number generates no enum member.
    const selectable = declared.filter(
      (value) => value !== MCPSetting_Capability.CAPABILITY_UNSPECIFIED
    );
    expect([...MCP_CAPABILITY_CHOICES].sort()).toEqual(selectable.sort());
  });
});

describe("isMCPMode", () => {
  test("accepts only selectable capabilities", () => {
    expect(isMCPMode(MCPSetting_Capability.DISABLED)).toBe(true);
    expect(isMCPMode(MCPSetting_Capability.READ_ONLY)).toBe(true);
    expect(isMCPMode(MCPSetting_Capability.READ_WRITE)).toBe(true);
    expect(isMCPMode(MCPSetting_Capability.CAPABILITY_UNSPECIFIED)).toBe(false);
    expect(isMCPMode(2 as MCPSetting_Capability)).toBe(false);
    expect(isMCPMode(99 as MCPSetting_Capability)).toBe(false);
  });
});

const settingWith = (capability: number) =>
  create(MCPSettingSchema, { capability });

describe("readConsentCeiling", () => {
  test("no response is the policy being unknown, not absent", () => {
    expect(readConsentCeiling(undefined)).toEqual({ kind: "unknown" });
  });

  test("a served ceiling carries the response the disclosure needs", () => {
    const setting = settingWith(MCPSetting_Capability.READ_WRITE);
    expect(readConsentCeiling(setting)).toEqual({ kind: "mode", setting });
  });

  test("disabled is a policy, so it reaches its own screen", () => {
    // Not undisclosable: an admin turned MCP off, which is a decision this page
    // can name and the one refusing ceiling with a screen of its own.
    const setting = settingWith(MCPSetting_Capability.DISABLED);
    expect(readConsentCeiling(setting)).toEqual({ kind: "mode", setting });
  });

  // The three ways a stored ceiling reaches this page without a name for it.
  // They differed once, by whether the server's serving table carried a row;
  // the remedy the page prints now names both repairs, so they are one state.
  test("a value nothing could resolve is undisclosable", () => {
    expect(
      readConsentCeiling(
        settingWith(MCPSetting_Capability.CAPABILITY_UNSPECIFIED)
      )
    ).toEqual({ kind: "undisclosable" });
  });

  test("the reserved tier is undisclosable", () => {
    expect(readConsentCeiling(settingWith(2))).toEqual({
      kind: "undisclosable",
    });
  });

  test("a tier a newer release wrote is undisclosable", () => {
    expect(readConsentCeiling(settingWith(5))).toEqual({
      kind: "undisclosable",
    });
  });
});
