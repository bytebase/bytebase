// @vitest-environment node
import { create } from "@bufbuild/protobuf";
import { beforeEach, describe, expect, test, vi } from "vitest";
import type { AccessGrantDraft } from "@/modules/sql-editor/model/accessGrantDraft";
import { State } from "@/types/proto-es/v1/common_pb";
import { DatabaseSchema$ } from "@/types/proto-es/v1/database_service_pb";
import {
  createJustInTimeTools,
  parsePreparedAccessGrantDraft,
  withPreparedAccessGrantReason,
} from "./justInTime";

const api = vi.hoisted(() => ({
  database: vi.fn(),
  setup: vi.fn(),
  validate: vi.fn(),
}));
vi.mock("@/api", () => ({
  databaseServiceClientConnect: { getDatabase: api.database },
}));
vi.mock("./tools/maskingSetup", () => ({ getMaskingSetup: api.setup }));
vi.mock("./deniedQueryAccess", () => ({
  validateAccessGrantDraft: api.validate,
}));

const project = "projects/sample";
const database = "instances/main/databases/app";
const proposal = {
  project,
  database,
  statement: "SELECT email FROM public.people;\n",
  unmask: true,
};
const tools = () =>
  createJustInTimeTools({
    workspaceName: "workspaces/example",
    projectName: project,
  });

describe("withPreparedAccessGrantReason", () => {
  const fallback: AccessGrantDraft = {
    project,
    sourceDatabase: database,
    targets: [database],
    statement: proposal.statement,
    unmask: false,
  };
  const prepared = {
    ...fallback,
    reason: "  Investigate a discrepancy.\n",
  };

  test("copies only the exact reason without mutating or broadening the fallback", () => {
    expect(withPreparedAccessGrantReason(fallback, prepared)).toEqual({
      ...fallback,
      reason: prepared.reason,
    });
    expect(fallback).not.toHaveProperty("reason");
    expect(
      withPreparedAccessGrantReason(fallback, {
        ...prepared,
        targets: [database, database],
      })
    ).toEqual({ ...fallback, reason: prepared.reason });
  });
  test.each([
    { project: "projects/other" },
    { sourceDatabase: "instances/other/databases/app" },
    { statement: "SELECT 1" },
    { targets: ["instances/main/databases/other"] },
    { schema: "public" },
    { container: "other" },
    { unmask: true },
  ])("does not merge a mismatched proposal %j", (change) => {
    expect(
      withPreparedAccessGrantReason(fallback, { ...prepared, ...change })
    ).toBe(fallback);
  });
  test("keeps missing inputs, reason and role drafts unchanged", () => {
    expect(withPreparedAccessGrantReason(undefined, prepared)).toBeUndefined();
    expect(withPreparedAccessGrantReason(fallback, undefined)).toBe(fallback);
    expect(withPreparedAccessGrantReason(fallback, fallback)).toBe(fallback);
    const role = {
      ...fallback,
      kind: "requestRole" as const,
      resources: [database],
      requiredPermissions: ["bb.sql.select" as const],
    };
    expect(withPreparedAccessGrantReason(role, prepared)).toBe(role);
  });
});

beforeEach(() => {
  vi.clearAllMocks();
  api.database.mockResolvedValue(
    create(DatabaseSchema$, { name: database, project, state: State.ACTIVE })
  );
  api.setup.mockResolvedValue({
    justInTime: { eligible: true, exemptionFallback: true },
    subscription: { status: "available", plan: "ENTERPRISE" },
    scopeAccess: { databasePicker: { status: "allowed" } },
  });
  api.validate.mockResolvedValue(true);
});

describe("JIT overview", () => {
  test("uses fresh database ownership and independent JIT evidence", async () => {
    const result = JSON.parse(await tools().overview({ database }));
    expect(result).toMatchObject({
      project,
      database,
      justInTime: { eligible: true },
    });
    expect(api.database).toHaveBeenCalledWith(
      expect.objectContaining({ name: database }),
      expect.anything()
    );
    expect(api.setup).toHaveBeenCalledWith(
      expect.objectContaining({ projectName: project, databaseName: database })
    );
    expect(result).not.toHaveProperty("features");
  });

  test("derives project from the selected database in a workspace chat", async () => {
    const result = JSON.parse(
      await createJustInTimeTools({
        workspaceName: "workspaces/example",
      }).overview({ database })
    );
    expect(result.project).toBe(project);
  });

  test.each([project, undefined])(
    "reuses and freshly verifies the previous query database with bound project %j",
    async (projectName) => {
      const result = JSON.parse(
        await createJustInTimeTools({
          workspaceName: "workspaces/example",
          projectName,
          historicalDatabase: { database, project },
        }).overview({})
      );
      expect(result).toMatchObject({ project, database });
      expect(api.database).toHaveBeenCalledWith(
        expect.objectContaining({ name: database }),
        expect.anything()
      );
    }
  );

  test("does not carry a historical database into a different project", async () => {
    const result = JSON.parse(
      await createJustInTimeTools({
        workspaceName: "workspaces/example",
        projectName: "projects/other",
        historicalDatabase: { database, project },
      }).overview({})
    );
    expect(result).not.toHaveProperty("database");
    expect(api.database).not.toHaveBeenCalled();
  });

  test("rejects historical database ownership that changed after the query", async () => {
    api.database.mockResolvedValue(
      create(DatabaseSchema$, {
        name: database,
        project: "projects/other",
        state: State.ACTIVE,
      })
    );
    const result = JSON.parse(
      await createJustInTimeTools({
        workspaceName: "workspaces/example",
        historicalDatabase: { database, project },
      }).overview({})
    );
    expect(result).toHaveProperty("error");
    expect(api.setup).not.toHaveBeenCalled();
  });

  test("allows an explicit database to replace historical context without rebinding the turn", async () => {
    const result = JSON.parse(
      await createJustInTimeTools({
        workspaceName: "workspaces/example",
        projectName: project,
        historicalDatabase: {
          database: "instances/main/databases/previous",
          project,
        },
      }).overview({ database })
    );
    expect(result).toMatchObject({ project, database });
  });

  test.each([
    { project: "projects/other" },
    { database: "instances/-/databases/app" },
  ])("rejects invalid or foreign scope %j", async (args) => {
    expect(JSON.parse(await tools().overview(args))).toHaveProperty("error");
    expect(api.setup).not.toHaveBeenCalled();
  });

  test.each([State.DELETED, State.STATE_UNSPECIFIED])(
    "rejects inactive database %s",
    async (state) => {
      api.database.mockResolvedValue(
        create(DatabaseSchema$, { name: database, project, state })
      );
      expect(JSON.parse(await tools().overview({ database }))).toHaveProperty(
        "error"
      );
    }
  );

  test("does not use a cached database from a different project", async () => {
    api.database.mockResolvedValue(
      create(DatabaseSchema$, {
        name: database,
        project: "projects/other",
        state: State.ACTIVE,
      })
    );
    expect(JSON.parse(await tools().overview({ database }))).toHaveProperty(
      "error"
    );
  });
});

describe("JIT draft preparation", () => {
  test("preserves exact SQL, targets and Unmask without navigation or API writes", async () => {
    const result = await tools().prepare({
      ...proposal,
      schema: "public",
      reason: "  Investigate the member-reported discrepancy.\n",
      container: "tenant",
      targets: [database, "instances/main/databases/other"],
    });
    const draft = parsePreparedAccessGrantDraft(result);
    expect(draft).toEqual({
      kind: "accessGrant",
      project,
      sourceDatabase: database,
      targets: [database, "instances/main/databases/other"],
      statement: proposal.statement,
      unmask: true,
      schema: "public",
      reason: "  Investigate the member-reported discrepancy.\n",
      container: "tenant",
    });
    expect(api.validate).toHaveBeenCalledWith(draft);
    expect(api.database).not.toHaveBeenCalled();
  });

  test("defaults to only the source target and supports ordinary JIT", async () => {
    const draft = parsePreparedAccessGrantDraft(
      await tools().prepare({ ...proposal, unmask: false })
    );
    expect(draft).toMatchObject({ targets: [database], unmask: false });
    expect(draft?.reason).toBe("Request temporary query access to app.");
  });

  test.each([false, true])(
    "persists a default reason for every blank reason with Unmask=%s",
    async (unmask) => {
      for (const reason of [undefined, "", " \n "]) {
        const draft = parsePreparedAccessGrantDraft(
          await tools().prepare({ ...proposal, unmask, reason })
        );
        expect(draft).toEqual({
          kind: "accessGrant",
          project,
          sourceDatabase: database,
          targets: [database],
          statement: proposal.statement,
          unmask,
          reason: unmask
            ? "Request temporary query access to app with unmasked results."
            : "Request temporary query access to app.",
        });
      }
    }
  );

  test("includes each exact target in the generated reason without changing query scope", async () => {
    const draft = parsePreparedAccessGrantDraft(
      await tools().prepare({
        ...proposal,
        targets: [database, "instances/main/databases/archive"],
      })
    );
    expect(draft).toMatchObject({
      targets: [database, "instances/main/databases/archive"],
      statement: proposal.statement,
      unmask: true,
      reason:
        "Request temporary query access to app, archive with unmasked results.",
    });
  });

  test("rejects a non-string reason in a prepared tool result", () => {
    expect(
      parsePreparedAccessGrantDraft(
        JSON.stringify({
          status: "unsaved",
          draft: {
            ...proposal,
            kind: "accessGrant",
            sourceDatabase: database,
            targets: [database],
            reason: 42,
          },
        })
      )
    ).toBeUndefined();
  });

  test.each([
    { statement: "" },
    { statement: " ", unmask: true },
    { unmask: undefined },
    { unmask: "true" },
    { database: "" },
    { project: "projects/other" },
    { statement: "x".repeat(100001) },
    { targets: [] },
    { targets: ["instances/-/databases/app"] },
    { schema: 42 },
    { container: [] },
    { reason: 42 },
    { reason: null },
  ])("rejects malformed draft input %j", async (change) => {
    const result = await tools().prepare({ ...proposal, ...change });
    expect(JSON.parse(result)).toHaveProperty("error");
    expect(parsePreparedAccessGrantDraft(result)).toBeUndefined();
    expect(api.validate).not.toHaveBeenCalled();
  });

  test("rejects revoked eligibility or target ownership on preparation", async () => {
    api.validate.mockResolvedValue(false);
    expect(JSON.parse(await tools().prepare(proposal))).toHaveProperty("error");
  });

  test("cannot substitute another database for this turn's selected resource", async () => {
    const selected = createJustInTimeTools({
      workspaceName: "workspaces/example",
      projectName: project,
      databaseName: database,
    });
    expect(
      JSON.parse(
        await selected.prepare({
          ...proposal,
          database: "instances/main/databases/other",
        })
      )
    ).toHaveProperty("error");
    expect(api.validate).not.toHaveBeenCalled();
  });

  test.each([
    "{}",
    "not json",
    '{"status":"unsaved","draft":{"kind":"requestRole"}}',
  ])("does not accept a model action or malformed tool result %s", (result) => {
    expect(parsePreparedAccessGrantDraft(result)).toBeUndefined();
  });
});
