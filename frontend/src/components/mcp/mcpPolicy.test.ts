import { create } from "@bufbuild/protobuf";
import { describe, expect, test } from "vitest";
import { MCPMethodClass } from "@/types/proto-es/v1/annotation_pb";
import { Engine } from "@/types/proto-es/v1/common_pb";
import { MCPSetting_Capability } from "@/types/proto-es/v1/setting_service_pb";
import {
  MCPCapabilityModeSchema,
  MCPEngineEnforcement_Masking,
  MCPEngineEnforcement_ReadOnlyDepth,
  MCPEngineEnforcementSchema,
  MCPInfoSchema,
  MCPMethodSchema,
} from "@/types/proto-es/v1/workspace_service_pb";
import {
  groupEnginesByMasking,
  groupEnginesByReadOnlyDepth,
  groupMethodsByService,
  isMCPMode,
  methodsServedBy,
  readConsentCeiling,
  serviceOfMethod,
} from "./mcpPolicy";

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

// The serving table every real response carries, one row per ceiling the gate
// evaluates. DISABLED is a row rather than an omission: it is a mode that
// decided to serve nothing, which is not the same as a mode nobody decided
// about (backend/api/v1/mcp_gate.go, mcpServingClasses).
const SERVED_MODES = [1, 3, 4];

const infoWith = (fields: Partial<{ capability: number; modes: number[] }>) =>
  create(MCPInfoSchema, {
    capability: fields.capability ?? MCPSetting_Capability.READ_ONLY,
    modes: (fields.modes ?? SERVED_MODES).map((capability) =>
      create(MCPCapabilityModeSchema, { capability })
    ),
  });

describe("readConsentCeiling", () => {
  test("no response is the policy being unknown, not absent", () => {
    expect(readConsentCeiling(undefined)).toEqual({ kind: "unknown" });
  });

  test("a served ceiling carries the response the disclosure needs", () => {
    const info = infoWith({ capability: 4 });
    expect(readConsentCeiling(info)).toEqual({ kind: "mode", info });
  });

  test("disabled is a policy, so it reaches its own screen", () => {
    // Not undisclosed: an admin turned MCP off, which is a decision this page
    // can name and the one refusing ceiling with a screen of its own.
    const info = infoWith({ capability: 1 });
    expect(readConsentCeiling(info)).toEqual({ kind: "mode", info });
  });

  test("an unresolvable stored value is unserved, like any value no mode serves", () => {
    // The capability arrives unspecified when nothing could be resolved from
    // the row. It has no row in modes either, and the remedy is the same as for
    // a value that parsed but nothing serves, so it is one state.
    expect(readConsentCeiling(infoWith({ capability: 0 }))).toEqual({
      kind: "unserved",
    });
  });

  test("a value with no row in the serving table is unserved", () => {
    expect(readConsentCeiling(infoWith({ capability: 2 }))).toEqual({
      kind: "unserved",
    });
  });

  test("a served ceiling this bundle cannot name is outdated, not unserved", () => {
    // The difference decides the instruction on screen: reload, or find an
    // admin. The server serves this one, so there is nothing for an admin to
    // repair.
    expect(
      readConsentCeiling(
        infoWith({ capability: 5, modes: [...SERVED_MODES, 5] })
      )
    ).toEqual({ kind: "outdated" });
  });
});
