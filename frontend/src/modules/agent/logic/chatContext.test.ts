import { describe, expect, test } from "vitest";
import type { ReactRoute } from "@/app/router";
import { resolveChatContext } from "./chatContext";

const routeWithParams = (params: ReactRoute["params"]) =>
  ({ params }) as ReactRoute;

describe("resolveChatContext", () => {
  test("uses the project resource on project routes", () => {
    expect(
      resolveChatContext(
        routeWithParams({ projectId: "orders" }),
        "workspaces/demo"
      )
    ).toBe("projects/orders");
    expect(
      resolveChatContext(
        routeWithParams({ project: "orders" }),
        "workspaces/demo"
      )
    ).toBe("projects/orders");
  });

  test("uses the active workspace on other routes", () => {
    expect(resolveChatContext(routeWithParams({}), "workspaces/demo")).toBe(
      "workspaces/demo"
    );
  });

  test("does not invent a context before the workspace is known", () => {
    expect(resolveChatContext(routeWithParams({}), "")).toBeUndefined();
  });
});
