import { describe, expect, test, vi } from "vitest";
import type { AppRouterInstance } from "@/app/router";
import { createNavigateTool } from "./navigate";

const routerWithRoutes = (paths: string[]) =>
  ({
    getRoutes: () => paths.map((path) => ({ path, name: path, children: [] })),
    push: vi.fn(),
    currentRoute: { value: { fullPath: "/" } },
  }) as unknown as AppRouterInstance;

describe("createNavigateTool", () => {
  test("bounds route discovery and reports omitted matches", async () => {
    const paths = Array.from(
      { length: 45 },
      (_, index) => `/route-${String(index).padStart(2, "0")}`
    );
    const navigate = createNavigateTool(routerWithRoutes(paths));

    const result = JSON.parse(await navigate({ list: true })) as {
      routes: string[];
      total: number;
      truncated: boolean;
    };

    expect(result.routes).toHaveLength(30);
    expect(result.total).toBe(45);
    expect(result.truncated).toBe(true);
  });

  test("filters route discovery by a user-provided query", async () => {
    const navigate = createNavigateTool(
      routerWithRoutes([
        "/projects/:projectId/databases",
        "/projects/:projectId/issues",
        "/instances/:instanceId",
      ])
    );

    const result = JSON.parse(
      await navigate({ list: true, query: "database" })
    ) as { routes: string[]; total: number; truncated: boolean };

    expect(result).toEqual({
      routes: ["/projects/:projectId/databases"],
      total: 1,
      truncated: false,
    });
  });
});
