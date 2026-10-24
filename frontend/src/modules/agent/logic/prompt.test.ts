// @vitest-environment node
import { describe, expect, test } from "vitest";
import { buildSystemPrompt } from "./prompt";

describe("buildSystemPrompt", () => {
  test("uses a selected project for one turn without narrowing normal API calls", () => {
    const prompt = buildSystemPrompt({
      path: "/projects/original",
      title: "Original project",
      context: "projects/original",
      referencedProject: "projects/referenced",
      runMode: "read-only",
    });

    expect(prompt).toContain("Chat context: projects/original");
    expect(prompt).toContain(
      "Referenced project for this turn: projects/referenced"
    );
    expect(prompt).toContain(
      "bounded project tools target the referenced project"
    );
    expect(prompt).toContain("This is not a chat context change");
    expect(prompt).toContain("matching the referenced project");
    expect(prompt).toContain("search_api");
    expect(prompt).toContain("call_api");
    expect(prompt).toContain("existing permissions and may change data");
    expect(prompt).toContain("only make changes when the user explicitly asks");
    expect(prompt).not.toContain("Do not navigate, call APIs");
    expect(prompt).not.toContain("Either works for mutations");
  });

  test("names the chat context separately from the live page", () => {
    const prompt = buildSystemPrompt({
      path: "/projects/other/issues",
      title: "Other project",
      context: "projects/orders",
    });

    expect(prompt).toContain("Chat context: projects/orders");
    expect(prompt).toContain("Current page: /projects/other/issues");
    expect(prompt).toContain("Navigation does not change the chat context");
  });

  test("accepts a workspace resource as the chat context", () => {
    expect(
      buildSystemPrompt({
        path: "/landing",
        title: "Workspace",
        context: "workspaces/demo",
      })
    ).toContain("Chat context: workspaces/demo");
  });

  test("does not carry a past project reference into an unqualified workspace follow-up", () => {
    const prompt = buildSystemPrompt({
      path: "/landing",
      title: "Workspace",
      context: "workspaces/demo",
    });

    expect(prompt).toContain(
      "A selected resource in an earlier user message does not scope this turn"
    );
    expect(prompt).toContain(
      "Do not infer a project from prior conversation when the current message has no project reference or explicit project name"
    );
    expect(prompt).toContain(
      "For an unqualified list or count in a workspace-bound chat, use the whole chat workspace"
    );
    expect(prompt).toContain("ListDatabases supports parent=workspaces/{id}");
    expect(prompt).toContain(
      "follow every page token before reporting an exact total"
    );
    expect(prompt).toContain(
      "ask which project only when the member requests a project-specific result without identifying one"
    );
    expect(prompt).not.toContain(
      "if it is a workspace and the request requires a specific project, ask which project"
    );
  });

  test("teaches the ref-based DOM workflow", () => {
    const prompt = buildSystemPrompt({
      path: "/issues/123",
      title: "Issue 123",
      role: "DBA",
    });

    expect(prompt).toContain(
      'DOM interaction workflow: get_page_state(mode="dom") → read element refs like [e1] in the DOM tree → dom_action(type, ref, value).'
    );
    expect(prompt).not.toContain("read element indices");
    expect(prompt).not.toContain("dom_action(type, index, value)");
  });

  test("keeps a read-only landing starter on its current page", () => {
    const prompt = buildSystemPrompt({
      path: "/projects/orders",
      title: "Order service",
      runMode: "read-only",
    });

    expect(prompt).toContain("Stay on the current page");
    expect(prompt).toContain("Do not navigate");
    expect(prompt).toContain("Do not navigate, call APIs");
    expect(prompt).toContain("get_project_open_issues");
    expect(prompt).toContain("get_project_overview");
    expect(prompt).toContain("get_project_governance_overview");
    expect(prompt).toContain("get_issue_overview");
    expect(prompt).toContain("get_database_overview");
    expect(prompt).toContain("get_project_recent_activity");
    expect(prompt).toContain("get_workspace_recent_activity");
  });

  test("asks the Agent to use a project overview for a current-state question", () => {
    const prompt = buildSystemPrompt({
      path: "/projects/orders",
      title: "Order service",
      context: "projects/orders",
      runMode: "read-only",
    });

    expect(prompt).toContain(
      "call get_project_overview before recommending a next step"
    );
    expect(prompt).toContain(
      "answer the member's actual question directly, not with a catalog of capabilities"
    );
    expect(prompt).toContain(
      "recommend at most one next step when the available evidence supports it"
    );
    expect(prompt).not.toContain("<project_readiness_evidence>");
  });

  test.each([
    ["project", "projects/orders"],
    ["workspace", "workspaces/default"],
  ])("requires complete %s facets before claiming absence", (_, context) => {
    const prompt = buildSystemPrompt({
      path: "/landing",
      title: "Home",
      context,
    });

    expect(prompt).toContain(
      "Do not claim a resource or issue is absent unless its facet is available, partial is false, and it is empty"
    );
    expect(prompt).toContain(
      "A partial or unavailable facet cannot establish absence"
    );
  });

  test("asks the Agent to inspect project governance before describing it", () => {
    const prompt = buildSystemPrompt({
      path: "/projects/orders",
      title: "Order service",
      context: "projects/orders",
      runMode: "read-only",
    });

    expect(prompt).toContain(
      "call get_project_governance_overview before describing how Database Changes are reviewed"
    );
    expect(prompt).toContain("conditions are evaluated per Database Change");
  });

  test("asks the Agent to inspect a selected Issue before describing its state", () => {
    const prompt = buildSystemPrompt({
      path: "/projects/orders",
      title: "Order service",
      context: "projects/orders",
      runMode: "read-only",
    });

    expect(prompt).toContain(
      "call get_issue_overview before explaining a specific Issue's approval"
    );
    expect(prompt).toContain(
      "not_started for a review run means no run is recorded; it does not prove that review is configured or applicable"
    );
    expect(prompt).toContain("matching the chat project");
  });

  test("asks the Agent to inspect a selected database before describing its state", () => {
    const prompt = buildSystemPrompt({
      path: "/projects/orders",
      title: "Order service",
      context: "projects/orders",
      runMode: "read-only",
    });

    expect(prompt).toContain(
      "call get_database_overview before explaining a specific database's current state"
    );
    expect(prompt).toContain("DML prior-backup readiness");
    expect(prompt).toContain(
      "does not establish that an existing recoverable backup exists"
    );
  });

  test("asks the Agent to use a workspace overview without recent activity", () => {
    const prompt = buildSystemPrompt({
      path: "/landing",
      title: "Workspace",
      context: "workspaces/default",
      runMode: "read-only",
    });

    expect(prompt).toContain(
      "call get_workspace_overview before recommending a next step"
    );
    expect(prompt).toContain(
      "Use get_workspace_recent_activity only when the member asks what changed recently"
    );
  });
});
