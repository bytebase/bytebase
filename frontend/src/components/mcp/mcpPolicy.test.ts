import { create } from "@bufbuild/protobuf";
import { describe, expect, test } from "vitest";
import { MCPMethodClass } from "@/types/proto-es/v1/annotation_pb";
import { Engine } from "@/types/proto-es/v1/common_pb";
import {
  MCPSetting_Capability,
  MCPSettingSchema,
} from "@/types/proto-es/v1/setting_service_pb";
import {
  MCPCapabilityModeSchema,
  MCPEngineEnforcement_Masking,
  MCPEngineEnforcement_ReadOnlyDepth,
  MCPEngineEnforcementSchema,
  MCPMethodSchema,
} from "@/types/proto-es/v1/workspace_service_pb";
import {
  ceilingIsBroken,
  groupEnginesByMasking,
  groupEnginesByReadOnlyDepth,
  groupMethodsByService,
  initialCapabilityPick,
  methodsServedBy,
  readStoredCeiling,
  serviceOfMethod,
} from "./mcpPolicy";

const setting = (fields: Partial<{ capability: number }>) =>
  create(MCPSettingSchema, {
    capability:
      fields.capability ?? MCPSetting_Capability.CAPABILITY_UNSPECIFIED,
  });

describe("readStoredCeiling", () => {
  test("a stored mode is the mode", () => {
    expect(readStoredCeiling(setting({ capability: 3 }))).toEqual({
      kind: "mode",
      capability: MCPSetting_Capability.READ_ONLY,
    });
  });

  test("an unspecified capability is unreadable, never read-write", () => {
    // Migration gives every workspace a concrete capability, so unspecified
    // means the stored token could not be resolved and enforcement refuses it.
    expect(readStoredCeiling(setting({}))).toEqual({ kind: "unreadable" });
  });

  test("a number no mode serves is unserved", () => {
    // The reserved 2 (was METADATA_ONLY), or a ceiling a newer release wrote.
    expect(readStoredCeiling(setting({ capability: 2 }))).toEqual({
      kind: "unserved",
      stored: "2",
    });
  });
});

describe("initialCapabilityPick", () => {
  test("a stored mode preselects itself", () => {
    expect(
      initialCapabilityPick({
        kind: "mode",
        capability: MCPSetting_Capability.DISABLED,
      })
    ).toBe(MCPSetting_Capability.DISABLED);
  });

  test("a broken row preselects nothing, so every pick is savable", () => {
    // Preselecting read-write here is what left an admin unable to repair the
    // row to read-write at all: the pick matched the preselection, so the form
    // was never dirty and the footer never enabled.
    for (const stored of [
      { kind: "unreadable" as const },
      { kind: "unserved" as const, stored: "2" },
    ]) {
      expect(ceilingIsBroken(stored)).toBe(true);
      expect(initialCapabilityPick(stored)).toBeUndefined();
    }
  });

  test("a valid mode is not broken", () => {
    // Without this the whole suite passes on `ceilingIsBroken = () => true`,
    // which would put every workspace into the repair banner.
    expect(
      ceilingIsBroken({
        kind: "mode",
        capability: MCPSetting_Capability.READ_ONLY,
      })
    ).toBe(false);
  });
});

describe("methodsServedBy", () => {
  const query = create(MCPMethodSchema, {
    method: "/bytebase.v1.SQLService/Query",
    operationId: "bytebase.v1.SQLService.Query",
    class: MCPMethodClass.READ,
  });
  const createSheet = create(MCPMethodSchema, {
    method: "/bytebase.v1.SheetService/CreateSheet",
    operationId: "bytebase.v1.SheetService.CreateSheet",
    class: MCPMethodClass.WRITE,
  });

  test("a mode serves a method when it serves the method's class", () => {
    const readOnly = create(MCPCapabilityModeSchema, {
      capability: MCPSetting_Capability.READ_ONLY,
      servedClasses: [MCPMethodClass.READ],
    });
    expect(methodsServedBy(readOnly, [query, createSheet])).toEqual([query]);
  });

  test("a mode that serves nothing serves nothing", () => {
    const disabled = create(MCPCapabilityModeSchema, {
      capability: MCPSetting_Capability.DISABLED,
      servedClasses: [],
    });
    expect(methodsServedBy(disabled, [query, createSheet])).toEqual([]);
    expect(methodsServedBy(undefined, [query, createSheet])).toEqual([]);
  });
});

describe("grouping", () => {
  test("methods group under their service's short name", () => {
    const rows = [
      create(MCPMethodSchema, {
        method: "/bytebase.v1.SQLService/Query",
        operationId: "bytebase.v1.SQLService.Query",
      }),
      create(MCPMethodSchema, {
        method: "/bytebase.v1.SQLService/Export",
        operationId: "bytebase.v1.SQLService.Export",
      }),
      create(MCPMethodSchema, {
        method: "/bytebase.v1.DatabaseService/GetDatabase",
        operationId: "bytebase.v1.DatabaseService.GetDatabase",
      }),
    ];
    expect(serviceOfMethod(rows[0])).toBe("SQLService");
    expect(groupMethodsByService(rows)).toEqual([
      { service: "DatabaseService", methods: [rows[2]] },
      { service: "SQLService", methods: [rows[1], rows[0]] },
    ]);
  });

  // Codex, #21236: a bundle in an open tab can outlive a backend upgrade, and
  // the server answers with the depth it computed, not the ones this build was
  // compiled against. Filtering the known order dropped the group entirely, so
  // the engine left the page rather than being described imprecisely — and both
  // renderers already have a default arm for exactly this.
  test("a depth this build does not know sorts last instead of vanishing", () => {
    const future = 99 as MCPEngineEnforcement_ReadOnlyDepth;
    const engines = [
      create(MCPEngineEnforcementSchema, {
        engine: Engine.SPANNER,
        readOnlyDepth: future,
      }),
      create(MCPEngineEnforcementSchema, {
        engine: Engine.POSTGRES,
        readOnlyDepth: MCPEngineEnforcement_ReadOnlyDepth.STATEMENT_AND_SESSION,
      }),
    ];
    expect(groupEnginesByReadOnlyDepth(engines)).toEqual([
      {
        value: MCPEngineEnforcement_ReadOnlyDepth.STATEMENT_AND_SESSION,
        engines: [Engine.POSTGRES],
      },
      { value: future, engines: [Engine.SPANNER] },
    ]);
  });

  test("a masking mode this build does not know sorts last too", () => {
    const future = 99 as MCPEngineEnforcement_Masking;
    const engines = [
      create(MCPEngineEnforcementSchema, {
        engine: Engine.SPANNER,
        masking: future,
      }),
      create(MCPEngineEnforcementSchema, {
        engine: Engine.POSTGRES,
        masking: MCPEngineEnforcement_Masking.COLUMN,
      }),
    ];
    expect(groupEnginesByMasking(engines)).toEqual([
      {
        value: MCPEngineEnforcement_Masking.COLUMN,
        engines: [Engine.POSTGRES],
      },
      { value: future, engines: [Engine.SPANNER] },
    ]);
  });

  test("engines group by read-only depth, deepest first", () => {
    const engines = [
      create(MCPEngineEnforcementSchema, {
        engine: Engine.MYSQL,
        readOnlyDepth: MCPEngineEnforcement_ReadOnlyDepth.STATEMENT,
      }),
      create(MCPEngineEnforcementSchema, {
        engine: Engine.POSTGRES,
        readOnlyDepth: MCPEngineEnforcement_ReadOnlyDepth.STATEMENT_AND_SESSION,
      }),
      create(MCPEngineEnforcementSchema, {
        engine: Engine.DATABRICKS,
        readOnlyDepth: MCPEngineEnforcement_ReadOnlyDepth.UNSUPPORTED,
      }),
    ];
    expect(groupEnginesByReadOnlyDepth(engines)).toEqual([
      {
        value: MCPEngineEnforcement_ReadOnlyDepth.STATEMENT_AND_SESSION,
        engines: [Engine.POSTGRES],
      },
      {
        value: MCPEngineEnforcement_ReadOnlyDepth.STATEMENT,
        engines: [Engine.MYSQL],
      },
      {
        value: MCPEngineEnforcement_ReadOnlyDepth.UNSUPPORTED,
        engines: [Engine.DATABRICKS],
      },
    ]);
  });

  test("masking keeps three states, never two", () => {
    // An engine Bytebase does not mask and an engine that masks without
    // consulting exemptions are different answers to "does ignoring exemptions
    // change anything here", and the copy must not flatten them.
    const engines = [
      create(MCPEngineEnforcementSchema, {
        engine: Engine.REDIS,
        masking: MCPEngineEnforcement_Masking.NONE,
      }),
      create(MCPEngineEnforcementSchema, {
        engine: Engine.MONGODB,
        masking: MCPEngineEnforcement_Masking.DOCUMENT,
      }),
      create(MCPEngineEnforcementSchema, {
        engine: Engine.POSTGRES,
        masking: MCPEngineEnforcement_Masking.COLUMN,
      }),
    ];
    expect(groupEnginesByMasking(engines).map((group) => group.value)).toEqual([
      MCPEngineEnforcement_Masking.COLUMN,
      MCPEngineEnforcement_Masking.DOCUMENT,
      MCPEngineEnforcement_Masking.NONE,
    ]);
  });
});
