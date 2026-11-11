import { afterEach, describe, expect, test, vi } from "vitest";
import type { AgentDiagnosticContext } from "./diagnosticHelp";
import {
  type AgentSqlFixSource,
  applyAgentSqlFixToTarget,
  captureAgentSqlFixSource,
  extractAgentSqlFixSQL,
  getAgentSqlFixIdentity,
  getAgentSqlFixTargetVersion,
  inspectAgentSqlFixTarget,
  notifyAgentSqlFixTargetChanged,
  registerAgentSqlFixTarget,
  revealAgentSqlFixTarget,
  subscribeAgentSqlFixTarget,
} from "./sqlFix";

const app = vi.hoisted(() => ({
  workspace: "workspaces/example",
  user: "users/member@example.com",
}));
vi.mock("@/stores/app", () => ({
  useAppStore: {
    getState: () => ({
      workspaceResourceName: () => app.workspace,
      currentUser: { name: app.user },
    }),
  },
}));

const resource = {
  kind: "plan" as const,
  name: "projects/example/plans/7",
  project: "projects/example",
  label: "Plan",
};
const context: AgentDiagnosticContext = {
  resource,
  kind: "sql-check",
  sourceId: "projects/example/plans/7/planCheckRun",
  summary: "Missing WHERE clause",
  sqlFixSource: { specId: "spec-1", sha256: "a".repeat(64) },
};
const source = (): AgentSqlFixSource => ({
  identity: JSON.stringify([app.workspace, app.user]),
  pageKey: "/projects/example/plans/7",
  resource,
  sourceId: context.sourceId,
  plan: resource.name,
  specId: "spec-1",
  sheet: "projects/example/sheets/9",
  targets: ["instances/prod/databases/example"],
  sha256: "a".repeat(64),
  sql: "UPDATE books SET price = 1;",
});
const cleanups: (() => void)[] = [];
afterEach(() => {
  for (const cleanup of cleanups.splice(0)) cleanup();
  app.workspace = "workspaces/example";
  app.user = "users/member@example.com";
});

describe("SQL fix proposal", () => {
  test("preserves the exact content of one complete SQL fence", () => {
    expect(
      extractAgentSqlFixSQL(
        "Complete replacement:\n```sql\n  UPDATE books\r\nSET price = 2;  \n```"
      )
    ).toBe("  UPDATE books\r\nSET price = 2;  ");
    expect(extractAgentSqlFixSQL("~~~SQL\nSELECT 1;\n\n~~~")).toBe(
      "SELECT 1;\n"
    );
  });

  test("selects the explicit replacement instead of explanatory SQL snippets", () => {
    expect(
      extractAgentSqlFixSQL(
        "The trailing comma is invalid:\n```sql\nid INT,\n```\nComplete replacement:\n```sql replacement\nCREATE TABLE example (\n  id INT\n);\n```"
      )
    ).toBe("CREATE TABLE example (\n  id INT\n);");
    expect(
      extractAgentSqlFixSQL("~~~SQL replacement\n  SELECT 2;  \r\n~~~")
    ).toBe("  SELECT 2;  ");
  });

  test.each([
    "```sql\nUPDATE books SET price = 2;",
    "```sql\nSELECT 1;\n``",
    "```sql\nSELECT 1;\n```\n```sql\nSELECT 2;\n```",
    "```sql\n   \n```",
    "```sql\nUPDATE books SET token = '[redacted]';\n```",
    "```sql\nUPDATE books SET price = 2;\n-- rest of SQL unchanged\n```",
    "```sql\n...\n```",
    "```sql\nUPDATE <table> SET value = 1;\n```",
    "```sql replacement\nSELECT 1;\n```\n```sql replacement\nSELECT 2;\n```",
    "```sql replacement\nSELECT 1;",
    "```sql replacement\nUPDATE <table> SET value = 1;\n```",
    "```sql example\nSELECT 1;\n```",
    "1. ```sql replacement\n   INSERT INTO books(note) VALUES ('first\n   second');\n   ```",
    "```sql\nid INT,\n```\n1. ```sql replacement\n   CREATE TABLE example (id INT);\n   ```",
  ])("rejects incomplete or placeholder proposals: %s", (answer) => {
    expect(extractAgentSqlFixSQL(answer)).toBeUndefined();
  });
});

describe("SQL fix target bridge", () => {
  test("fails closed without a mounted adapter or a complete identity", () => {
    expect(captureAgentSqlFixSource(context)).toBeUndefined();
    expect(inspectAgentSqlFixTarget(source())).toEqual({
      status: "unavailable",
    });
    expect(applyAgentSqlFixToTarget(source(), "SELECT 2")).toBe(false);
    app.user = "";
    expect(getAgentSqlFixIdentity()).toBeUndefined();
  });

  test("captures only an explicitly hinted exact diagnostic source", () => {
    const captured = source();
    const capture = vi.fn(() => captured);
    cleanups.push(
      registerAgentSqlFixTarget({
        capture,
        reveal: vi.fn(),
        inspect: () => ({ status: "ready", currentSql: captured.sql }),
        apply: () => true,
      })
    );
    expect(
      captureAgentSqlFixSource({ ...context, sqlFixSource: undefined })
    ).toBeUndefined();
    expect(capture).not.toHaveBeenCalled();
    expect(captureAgentSqlFixSource(context)).toEqual(captured);
    expect(
      captureAgentSqlFixSource({ ...context, sourceId: "another-source" })
    ).toBeUndefined();
    expect(
      captureAgentSqlFixSource({
        ...context,
        sqlFixSource: { sha256: "b".repeat(64) },
      })
    ).toBeUndefined();
  });

  test("rejects malformed adapter sources without allowing a review", () => {
    cleanups.push(
      registerAgentSqlFixTarget({
        capture: () =>
          ({ ...source(), targets: undefined }) as unknown as AgentSqlFixSource,
        reveal: vi.fn(),
        inspect: () => ({ status: "ready" }),
        apply: () => true,
      })
    );
    expect(captureAgentSqlFixSource(context)).toBeUndefined();
  });

  test("accepts local negative Plan and sheet IDs only on their create page", () => {
    const captured = {
      ...source(),
      pageKey: "example/create",
      plan: "projects/example/plans/-101",
      sheet: "projects/example/sheets/-102",
    };
    cleanups.push(
      registerAgentSqlFixTarget({
        capture: () => captured,
        reveal: vi.fn(),
        inspect: () => ({ status: "ready" }),
        apply: () => true,
      })
    );
    expect(captureAgentSqlFixSource(context)).toEqual(captured);
    captured.pageKey = "example/7";
    expect(captureAgentSqlFixSource(context)).toBeUndefined();
  });

  test("ignores asynchronous capture after the mounted route or actor changes", async () => {
    let resolve: (value: AgentSqlFixSource) => void = () => {};
    const captured = source();
    const unregister = registerAgentSqlFixTarget({
      capture: () =>
        new Promise<AgentSqlFixSource>((done) => {
          resolve = done;
        }),
      reveal: vi.fn(),
      inspect: () => ({ status: "ready", currentSql: captured.sql }),
      apply: () => true,
    });
    const pending = captureAgentSqlFixSource(context);
    unregister();
    resolve(captured);
    await expect(pending).resolves.toBeUndefined();
  });

  test("rechecks current identity and original SQL before applying", () => {
    const captured = source();
    let currentSql = captured.sql;
    const apply = vi.fn(() => true);
    const reveal = vi.fn();
    cleanups.push(
      registerAgentSqlFixTarget({
        capture: () => captured,
        reveal,
        inspect: () => ({ status: "ready", currentSql }),
        apply,
      })
    );
    revealAgentSqlFixTarget(captured);
    expect(reveal).toHaveBeenCalledWith(captured);
    currentSql = "SELECT 'changed';";
    expect(applyAgentSqlFixToTarget(captured, "SELECT 2;")).toBe(false);
    currentSql = captured.sql;
    expect(applyAgentSqlFixToTarget(captured, "SELECT 2;")).toBe(true);
    app.user = "users/another@example.com";
    expect(inspectAgentSqlFixTarget(captured)).toEqual({
      status: "unavailable",
    });
    expect(applyAgentSqlFixToTarget(captured, "SELECT 3;")).toBe(false);
    expect(apply).toHaveBeenCalledTimes(1);
  });

  test("notifies subscribers while stale adapter cleanup cannot remove a newer registration", () => {
    const listener = vi.fn();
    cleanups.push(subscribeAgentSqlFixTarget(listener));
    const captured = source();
    const adapter = {
      capture: () => captured,
      reveal: vi.fn(),
      inspect: () => ({ status: "ready" as const, currentSql: captured.sql }),
      apply: () => true,
    };
    const before = getAgentSqlFixTargetVersion();
    const first = registerAgentSqlFixTarget(adapter);
    cleanups.push(registerAgentSqlFixTarget(adapter));
    first();
    notifyAgentSqlFixTargetChanged();
    expect(getAgentSqlFixTargetVersion()).toBeGreaterThan(before);
    expect(listener).toHaveBeenCalled();
    expect(captureAgentSqlFixSource(context)).toEqual(captured);
  });
});
