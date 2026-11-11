import { create } from "@bufbuild/protobuf";
import { SHA256 } from "crypto-js";
import { describe, expect, test, vi } from "vitest";
import type { AgentDiagnosticContext } from "@/modules/agent/logic/diagnosticHelp";
import {
  Plan_SpecSchema,
  PlanSchema,
} from "@/types/proto-es/v1/plan_service_pb";
import { ProjectSchema } from "@/types/proto-es/v1/project_service_pb";
import { SheetSchema } from "@/types/proto-es/v1/sheet_service_pb";
import { UserSchema } from "@/types/proto-es/v1/user_service_pb";
import type { PlanDetailPageState } from "../shell/hooks/types";
import { createPlanSqlFixTarget, type PlanSqlFixEditor } from "./sqlFixTarget";

const sql = "DELETE FROM books;";
const database = "instances/main/databases/books";
const project = "projects/example";
const sheet = `${project}/sheets/${SHA256(sql).toString()}`;
const identity = '["workspaces/example","users/member@example.com"]';

function fixture() {
  const spec = create(Plan_SpecSchema, {
    id: "spec-1",
    config: {
      case: "changeDatabaseConfig",
      value: { sheet, targets: [database] },
    },
  });
  let page = {
    pageKey: "example/42",
    ready: true,
    isCreating: false,
    readonly: false,
    plan: create(PlanSchema, {
      name: `${project}/plans/42`,
      creator: "users/member@example.com",
      specs: [spec],
    }),
    project: create(ProjectSchema, { name: project }),
    currentUser: create(UserSchema, { name: "users/member@example.com" }),
  } as PlanDetailPageState;
  let currentIdentity: string | undefined = identity;
  let currentSql = sql;
  let editorStatus: "ready" | "loading" = "ready";
  let mounted = true;
  const applied: string[] = [];
  const content = new TextEncoder().encode(sql);
  const fetchSheet = vi.fn(async (name: string) =>
    create(SheetSchema, { name, content, contentSize: BigInt(content.length) })
  );
  const reveal = vi.fn();
  const expandTargets = (candidate: typeof spec) =>
    candidate.config.case === "changeDatabaseConfig"
      ? candidate.config.value.targets
      : [];
  const readEditor = (): PlanSqlFixEditor | undefined =>
    mounted
      ? {
          getSnapshot: () => ({
            pageKey: page.pageKey,
            plan: page.plan.name,
            specId: spec.id,
            sheet,
            targets: expandTargets(spec),
            currentSql,
            status: editorStatus,
          }),
          apply: (expected, next) => {
            if (currentSql !== expected) return false;
            currentSql = next;
            applied.push(next);
            return true;
          },
        }
      : undefined;
  const adapter = createPlanSqlFixTarget({
    getPage: () => page,
    getIdentity: () => currentIdentity,
    getEditor: readEditor,
    getDraftSql: () => currentSql,
    expandTargets,
    fetchSheet,
    reveal,
  });
  const context: AgentDiagnosticContext = {
    resource: { kind: "plan", name: page.plan.name, project, label: "Plan" },
    kind: "sql-check",
    sourceId: `${page.plan.name}/planCheckRun`,
    summary: "Missing WHERE clause",
    sqlFixSource: { sha256: SHA256(sql).toString(), target: database },
  };
  return {
    adapter,
    applied,
    context,
    fetchSheet,
    reveal,
    spec,
    setSql: (value: string) => {
      currentSql = value;
    },
    setMounted: (value: boolean) => {
      mounted = value;
    },
    setStatus: (value: typeof editorStatus) => {
      editorStatus = value;
    },
    setPage: (patch: Partial<PlanDetailPageState>) => {
      page = { ...page, ...patch };
    },
    setIdentity: (value: string | undefined) => {
      currentIdentity = value;
    },
  };
}

describe("createPlanSqlFixTarget", () => {
  test("does not capture a page still owned by the previous signed-in user", async () => {
    const f = fixture();
    f.setIdentity('["workspaces/example","users/other@example.com"]');
    expect(await f.adapter.capture(f.context)).toBeUndefined();
  });
  test("captures an immutable exact source, reveals it, and only changes its unsaved editor", async () => {
    const f = fixture();
    const source = await f.adapter.capture(f.context);
    expect(source).toMatchObject({
      identity,
      pageKey: "example/42",
      plan: `${project}/plans/42`,
      specId: "spec-1",
      sheet,
      sql,
      targets: [database],
    });
    expect(f.fetchSheet).toHaveBeenCalledWith(sheet);
    f.adapter.reveal(source!);
    expect(f.reveal).toHaveBeenCalledWith("spec-1");
    expect(f.adapter.inspect(source!)).toEqual({
      status: "ready",
      currentSql: sql,
    });
    expect(f.adapter.apply(source!, "DELETE FROM books WHERE id = 1;")).toBe(
      true
    );
    expect(f.applied).toEqual(["DELETE FROM books WHERE id = 1;"]);
    expect(
      f.spec.config.case === "changeDatabaseConfig" && f.spec.config.value.sheet
    ).toBe(sheet);
  });

  test("waits for a revealed editor to mount without queuing application", async () => {
    const f = fixture();
    const source = await f.adapter.capture(f.context);
    f.setMounted(false);
    expect(f.adapter.inspect(source!)).toEqual({ status: "loading" });
    expect(f.adapter.apply(source!, "SELECT 1;")).toBe(false);
    f.setMounted(true);
    expect(f.applied).toEqual([]);
    expect(f.adapter.inspect(source!).status).toBe("ready");
  });

  test.each([
    "editor",
    "sheet",
    "targets",
    "page",
    "identity",
    "readonly",
    "rollout",
    "permission",
    "loading",
  ])("fails closed after the reviewed target changes: %s", async (change) => {
    const f = fixture();
    const source = await f.adapter.capture(f.context);
    if (change === "editor") f.setSql("DELETE FROM authors;");
    if (change === "sheet" && f.spec.config.case === "changeDatabaseConfig")
      f.spec.config.value.sheet = `${project}/sheets/${"b".repeat(64)}`;
    if (change === "targets" && f.spec.config.case === "changeDatabaseConfig")
      f.spec.config.value.targets = ["instances/main/databases/other"];
    if (change === "page") f.setPage({ pageKey: "example/43" });
    if (change === "identity") f.setIdentity(undefined);
    if (change === "readonly") f.setPage({ readonly: true });
    if (change === "rollout")
      f.setPage({
        plan: create(PlanSchema, {
          name: `${project}/plans/42`,
          hasRollout: true,
          specs: [f.spec],
        }),
      });
    if (change === "permission")
      f.setPage({
        currentUser: create(UserSchema, { name: "users/other@example.com" }),
      });
    if (change === "loading") f.setStatus("loading");
    expect(f.adapter.apply(source!, "SELECT 1;")).toBe(false);
    expect(f.adapter.inspect(source!).status).not.toBe("ready");
    expect(f.applied).toEqual([]);
  });

  test.each([
    "missing-hash",
    "ambiguous",
    "wrong-project",
    "wrong-resource",
    "wrong-target",
    "release",
    "incomplete",
    "invalid-utf8",
    "wrong-hash",
  ])("does not capture unverifiable source: %s", async (change) => {
    const f = fixture();
    if (change === "missing-hash") f.context.sqlFixSource = undefined;
    if (change === "ambiguous")
      f.setPage({
        plan: create(PlanSchema, {
          name: `${project}/plans/42`,
          specs: [f.spec, { ...f.spec, id: "spec-2" }],
        }),
      });
    if (change === "wrong-project")
      f.context.resource.project = "projects/other";
    if (change === "wrong-resource")
      f.context.resource.name = `${project}/plans/43`;
    if (change === "wrong-target")
      f.context.sqlFixSource!.target = "instances/main/databases/other";
    if (change === "release" && f.spec.config.case === "changeDatabaseConfig")
      f.spec.config.value.release = `${project}/releases/1`;
    if (change === "incomplete")
      f.fetchSheet.mockResolvedValue(
        create(SheetSchema, {
          name: sheet,
          content: new TextEncoder().encode(sql),
          contentSize: 999n,
        })
      );
    if (change === "invalid-utf8")
      f.fetchSheet.mockResolvedValue(
        create(SheetSchema, {
          name: sheet,
          content: new Uint8Array([0xff]),
          contentSize: 1n,
        })
      );
    if (change === "wrong-hash")
      f.fetchSheet.mockResolvedValue(
        create(SheetSchema, {
          name: sheet,
          content: new TextEncoder().encode("SELECT 1;"),
          contentSize: 9n,
        })
      );
    expect(await f.adapter.capture(f.context)).toBeUndefined();
  });

  test("retains an outdated finding's recorded SQL and disables Apply against another revision", async () => {
    const f = fixture();
    f.context.kind = "review-finding";
    f.context.sqlFixSource!.specId = "spec-1";
    f.context.sqlFixSource!.targets = [database];
    if (f.spec.config.case === "changeDatabaseConfig")
      f.spec.config.value.sheet = `${project}/sheets/${"d".repeat(64)}`;
    const source = await f.adapter.capture(f.context);
    expect(source?.sheet).toBe(sheet);
    expect(source?.sql).toBe(sql);
    expect(f.adapter.inspect(source!)).toEqual({ status: "stale" });
    expect(f.adapter.apply(source!, "SELECT 1;")).toBe(false);
  });

  test.each([
    { targets: ["instances/main/databases/other"] },
    { targets: [database, "instances/main/databases/other"] },
  ])(
    "does not rebind a finding to the current target set $targets",
    async ({ targets }) => {
      const f = fixture();
      f.context.kind = "review-finding";
      f.context.sqlFixSource = {
        specId: "spec-1",
        sha256: SHA256(sql).toString(),
        targets: [database],
      };
      if (f.spec.config.case === "changeDatabaseConfig")
        f.spec.config.value.targets = targets;

      const source = await f.adapter.capture(f.context);

      expect(source?.targets).toEqual([database]);
      expect(f.adapter.inspect(source!)).toEqual({ status: "stale" });
      expect(f.adapter.apply(source!, "SELECT 1;")).toBe(false);
      expect(f.applied).toEqual([]);
    }
  );

  test("allows a finding only when its recorded targets exactly match the current spec", async () => {
    const f = fixture();
    const targets = [database, "instances/main/databases/other"];
    f.context.kind = "review-finding";
    f.context.sqlFixSource = {
      specId: "spec-1",
      sha256: SHA256(sql).toString(),
      targets,
    };
    if (f.spec.config.case === "changeDatabaseConfig")
      f.spec.config.value.targets = [...targets].reverse();

    const source = await f.adapter.capture(f.context);
    targets.push("instances/main/databases/later");

    expect(source?.targets).toEqual([
      database,
      "instances/main/databases/other",
    ]);
    expect(f.adapter.inspect(source!).status).toBe("ready");
    expect(f.adapter.apply(source!, "SELECT 1;")).toBe(true);
    expect(f.applied).toEqual(["SELECT 1;"]);
  });

  test.each([
    { targets: undefined },
    { targets: [] },
    { targets: ["projects/example/databaseGroups/group"] },
    { targets: [database, database] },
  ])(
    "does not capture a finding without unambiguous database targets $targets",
    async ({ targets }) => {
      const f = fixture();
      f.context.kind = "review-finding";
      f.context.sqlFixSource = {
        specId: "spec-1",
        sha256: SHA256(sql).toString(),
        targets,
      };

      expect(await f.adapter.capture(f.context)).toBeUndefined();
      expect(f.fetchSheet).not.toHaveBeenCalled();
      expect(f.applied).toEqual([]);
    }
  );

  test("captures checked draft SQL rather than fetching saved SQL", async () => {
    const f = fixture();
    const draft = "DELETE FROM books WHERE id = 1;";
    f.setSql(draft);
    f.context.resource = {
      kind: "database",
      name: database,
      project,
      label: "Books",
    };
    f.context.sql = {
      source: "draft",
      text: draft,
      sheet,
      sha256: SHA256(draft).toString(),
    };
    f.context.sqlFixSource = {
      specId: "spec-1",
      sha256: SHA256(draft).toString(),
      target: database,
    };
    const source = await f.adapter.capture(f.context);
    expect(source?.sql).toBe(draft);
    expect(f.fetchSheet).not.toHaveBeenCalled();
    expect(f.adapter.inspect(source!).status).toBe("ready");
  });

  test("does not swap newly edited SQL into checked draft evidence", async () => {
    const f = fixture();
    f.context.sql = {
      source: "draft",
      text: sql,
      sheet,
      sha256: SHA256(sql).toString(),
    };
    f.context.sqlFixSource!.specId = "spec-1";
    f.setSql("SELECT 1;");
    expect(await f.adapter.capture(f.context)).toBeUndefined();
  });

  test("preserves a source UTF-8 BOM and rejects a stale asynchronous capture", async () => {
    const f = fixture();
    const original = "\uFEFFSELECT 1;";
    const hash =
      "a35432496aa3b58f1818bc547ae78c87d1a5e702dcc7f28e2e1fdb38ff3bbf52";
    const bomSheet = `${project}/sheets/${hash}`;
    if (f.spec.config.case === "changeDatabaseConfig")
      f.spec.config.value.sheet = bomSheet;
    f.context.sqlFixSource = { specId: "spec-1", sha256: hash };
    f.fetchSheet.mockResolvedValue(
      create(SheetSchema, {
        name: bomSheet,
        content: new TextEncoder().encode(original),
        contentSize: 12n,
      })
    );
    expect((await f.adapter.capture(f.context))?.sql).toBe(original);
    let resolve!: (
      value: ReturnType<typeof create<typeof SheetSchema>>
    ) => void;
    f.fetchSheet.mockImplementation(
      () =>
        new Promise((done) => {
          resolve = done;
        })
    );
    const capture = f.adapter.capture(f.context);
    f.setPage({ pageKey: "other" });
    resolve(
      create(SheetSchema, {
        name: bomSheet,
        content: new TextEncoder().encode(original),
        contentSize: 12n,
      })
    );
    expect(await capture).toBeUndefined();
  });
});
