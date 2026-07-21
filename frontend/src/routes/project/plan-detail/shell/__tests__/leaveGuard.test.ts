import { describe, expect, test } from "vitest";
import { decideLeaveAction } from "../leaveGuard";

describe("decideLeaveAction", () => {
  test("allows navigation when no scopes are editing", () => {
    expect(
      decideLeaveAction({
        editingScopes: {},
        isBypassed: false,
        targetPath: "/x",
      })
    ).toEqual({ action: "allow" });
  });

  test("allows navigation when bypass flag is set", () => {
    expect(
      decideLeaveAction({
        editingScopes: { title: true },
        isBypassed: true,
        targetPath: "/x",
      })
    ).toEqual({ action: "allow" });
  });

  test("intercepts when an edit scope is open", () => {
    expect(
      decideLeaveAction({
        editingScopes: { title: true },
        isBypassed: false,
        targetPath: "/x",
      })
    ).toEqual({ action: "intercept", pendingTarget: "/x" });
  });

  test("intercepts when multiple scopes are open", () => {
    expect(
      decideLeaveAction({
        editingScopes: { title: true, description: true },
        isBypassed: false,
        targetPath: "/y",
      })
    ).toEqual({ action: "intercept", pendingTarget: "/y" });
  });
});
