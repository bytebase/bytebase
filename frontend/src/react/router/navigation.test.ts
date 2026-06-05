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
      ["project.issue.detail", "/projects/:projectId/issues/:issueId"],
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

  // Regression: a substring-prefix collision in the params bag (e.g.
  // `:project` inherited from the SQL-editor route alongside an
  // explicit `:projectId` for an issue route) used to corrupt the
  // longer placeholder, turning `/projects/:projectId/issues/:issueId`
  // into `/projects/<v>Id/issues/123`. The substitution must match
  // `:key` as a whole token.
  test("does not substring-replace `:project` inside `:projectId`", () => {
    expect(
      resolvePath("project.issue.detail", {
        params: {
          project: "from-inherited-route",
          projectId: "project-sample",
          issueId: "123",
        },
      })
    ).toBe("/projects/project-sample/issues/123");
  });

  // Order-of-iteration safety: even if the shorter key appears first
  // in `Object.entries`, the negative-lookahead guard prevents the
  // collision. Both orderings must produce the same result.
  test("token match is independent of params iteration order", () => {
    expect(
      resolvePath("project.issue.detail", {
        params: {
          projectId: "p1",
          project: "p1",
          issueId: "i1",
        },
      })
    ).toBe("/projects/p1/issues/i1");
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
