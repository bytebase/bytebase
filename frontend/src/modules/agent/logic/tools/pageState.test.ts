import { describe, expect, test, vi } from "vitest";
import type { AppRouterInstance } from "@/app/router";

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

import { lazyExtractDomTree } from "../../dom";
import type { AgentDomContext } from "../../dom/interactionRegistry";
import { extractRouteContext } from "../context";
import { createPageStateTool } from "./pageState";

const context: AgentDomContext = {
  actor: "users/1",
  workspace: "workspaces/demo",
  chatId: "chat",
  runId: "run",
  runMode: "default",
  pageKey: "/projects/demo",
  pageGeneration: 1,
  isCurrent: () => true,
};

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
    } as unknown as AppRouterInstance;

    document.title = "Project Demo";

    const tool = createPageStateTool(router, () => context);
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
    } as unknown as AppRouterInstance;

    document.title = "Issue 123";

    const tool = createPageStateTool(router, () => ({
      ...context,
      pageKey: router.currentRoute.value.fullPath,
    }));
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
  test("does not extract DOM on a page replaced during route enrichment", async () => {
    const router = {
      currentRoute: { value: { fullPath: context.pageKey } },
    } as AppRouterInstance;
    let current = true;
    vi.mocked(lazyExtractDomTree).mockClear();
    vi.mocked(extractRouteContext).mockImplementationOnce(async () => {
      current = false;
      return {};
    });
    const tool = createPageStateTool(router, () => ({
      ...context,
      isCurrent: () => current,
    }));
    expect(JSON.parse(await tool({ mode: "dom" }))).toHaveProperty("error");
    expect(lazyExtractDomTree).not.toHaveBeenCalled();
  });
  test("discards an extraction that loses the run before it returns", async () => {
    const router = {
      currentRoute: { value: { fullPath: context.pageKey } },
    } as AppRouterInstance;
    let current = true;
    const scope = { ...context, isCurrent: () => current };
    vi.mocked(lazyExtractDomTree).mockImplementationOnce(async (received) => {
      expect(received).toBe(scope);
      current = false;
      return { tree: "stale", count: 1 };
    });
    expect(
      JSON.parse(
        await createPageStateTool(router, () => scope)({ mode: "dom" })
      )
    ).toHaveProperty("error");
  });
});
