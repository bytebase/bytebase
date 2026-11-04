import { afterEach, describe, expect, test, vi } from "vitest";
import {
  AGENT_PLAN_DRAFT_EVENT,
  appendAgentPlanDraftSQL,
  dispatchAgentPlanDraftToCurrentPlan,
} from "./agentDraftHandoff";

afterEach(() => {
  vi.restoreAllMocks();
});

describe("Agent Plan draft handoff", () => {
  test("delivers the exact project, database, and SQL to the open Plan", () => {
    const listener = vi.fn((event: Event) => {
      (event as CustomEvent<{ handled: boolean }>).detail.handled = true;
    });
    window.addEventListener(AGENT_PLAN_DRAFT_EVENT, listener);

    expect(
      dispatchAgentPlanDraftToCurrentPlan({
        project: "projects/example",
        database: "instances/dev/databases/app",
        sql: "CREATE TABLE app_log (id INT);",
      })
    ).toBe(true);
    expect(listener).toHaveBeenCalledOnce();
    expect((listener.mock.calls[0][0] as CustomEvent).detail).toMatchObject({
      project: "projects/example",
      database: "instances/dev/databases/app",
      sql: "CREATE TABLE app_log (id INT);",
      handled: true,
    });

    window.removeEventListener(AGENT_PLAN_DRAFT_EVENT, listener);
  });

  test("reports when no Plan-create page can receive the draft", () => {
    expect(
      dispatchAgentPlanDraftToCurrentPlan({
        project: "projects/example",
        database: "instances/dev/databases/app",
        sql: "CREATE TABLE app_log (id INT);",
      })
    ).toBe(false);
  });

  test("appends SQL without losing existing content or adding blank noise", () => {
    expect(
      appendAgentPlanDraftSQL("SELECT 1;\n", "ALTER TABLE app ADD x INT;")
    ).toBe("SELECT 1;\n\nALTER TABLE app ADD x INT;");
    expect(
      appendAgentPlanDraftSQL("  \n", "  ALTER TABLE app ADD x INT;  ")
    ).toBe("ALTER TABLE app ADD x INT;");
  });
});
