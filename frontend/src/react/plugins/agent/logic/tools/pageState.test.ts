import { describe, expect, test, vi } from "vitest";
import type { Router } from "vue-router";

vi.mock("../../dom", () => ({
  lazyExtractDomTree: vi.fn(async () => ({
    count: 2,
    tree: ['[e1] button "Create"', '[e2] textbox "Name"'],
  })),
}));

vi.mock("../context", () => ({
  extractRouteContext: vi.fn(async () => ({
    project: {
      name: "projects/demo",
    },
  })),
}));

import { createPageStateTool } from "./pageState";

describe("createPageStateTool", () => {
  test("returns semantic state by default", async () => {
    const router = {
      currentRoute: {
        value: {
          fullPath: "/projects/demo",
          name: "project",
          params: { projectId: "demo" },
          query: { view: "overview" },
        },
      },
    } as unknown as Router;

    document.title = "Project Demo";

    const tool = createPageStateTool(router);
    const result = JSON.parse(await tool());

    expect(result).toEqual({
      path: "/projects/demo",
      name: "project",
      params: { projectId: "demo" },
      query: { view: "overview" },
      title: "Project Demo",
      context: {
        project: {
          name: "projects/demo",
        },
      },
    });
    expect(result).not.toHaveProperty("domTree");
    expect(result).not.toHaveProperty("interactiveElements");
  });

  test("returns DOM state with ref-labeled tree when requested", async () => {
    const router = {
      currentRoute: {
        value: {
          fullPath: "/projects/demo/issues/123",
          name: "issue",
          params: { projectId: "demo", issueId: "123" },
          query: {},
        },
      },
    } as unknown as Router;

    document.title = "Issue 123";

    const tool = createPageStateTool(router);
    const result = JSON.parse(await tool({ mode: "dom" }));

    expect(result).toEqual({
      path: "/projects/demo/issues/123",
      name: "issue",
      params: { projectId: "demo", issueId: "123" },
      query: {},
      title: "Issue 123",
      context: {
        project: {
          name: "projects/demo",
        },
      },
      interactiveElements: 2,
      domTree: ['[e1] button "Create"', '[e2] textbox "Name"'],
    });
  });
});
