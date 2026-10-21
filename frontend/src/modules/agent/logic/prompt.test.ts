// @vitest-environment node
import { describe, expect, test } from "vitest";
import { buildSystemPrompt } from "./prompt";

describe("buildSystemPrompt", () => {
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
    expect(prompt).toContain("get_project_open_issues");
    expect(prompt).toContain("get_project_recent_activity");
    expect(prompt).toContain("get_workspace_recent_activity");
  });
});
