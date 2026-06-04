import { beforeEach, describe, expect, test, vi } from "vitest";
import {
  navigateByName,
  navigateToPath,
  resolvePath,
  setAppRouter,
  setRouteNameIndex,
} from "./navigation";

beforeEach(() => {
  setRouteNameIndex(
    new Map<string, string>([
      ["auth.signin", "/auth"],
      ["project.detail", "/projects/:projectId"],
    ])
  );
});

describe("navigation resolvePath", () => {
  test("resolves a known route name to its path", () => {
    expect(resolvePath("auth.signin")).toBe("/auth");
  });

  test("fills :params", () => {
    expect(resolvePath("project.detail", { params: { projectId: "p1" } })).toBe(
      "/projects/p1"
    );
  });

  test("appends a query string", () => {
    expect(resolvePath("auth.signin", { query: { redirect: "/x" } })).toBe(
      "/auth?redirect=%2Fx"
    );
  });

  test("drops undefined query values", () => {
    expect(resolvePath("auth.signin", { query: { redirect: undefined } })).toBe(
      "/auth"
    );
  });

  test("unknown name falls back to / (never throws mid-guard)", () => {
    expect(resolvePath("does.not.exist")).toBe("/");
  });
});

describe("navigation dispatch", () => {
  test("navigateByName resolves then calls the registered router", () => {
    const navigate = vi.fn();
    setAppRouter({ navigate });
    navigateByName("auth.signin", {
      query: { redirect: "/x" },
      replace: true,
    });
    expect(navigate).toHaveBeenCalledWith("/auth?redirect=%2Fx", {
      replace: true,
    });
  });

  test("navigateToPath passes the raw path through", () => {
    const navigate = vi.fn();
    setAppRouter({ navigate });
    navigateToPath("/raw", { replace: false });
    expect(navigate).toHaveBeenCalledWith("/raw", { replace: false });
  });
});
