import { describe, expect, test } from "vitest";
import { resolveReactPagePath } from "./mount";

describe("resolveReactPagePath", () => {
  test("resolves the InactiveRemindModal entry", () => {
    expect(resolveReactPagePath("InactiveRemindModal")).toBe(
      "./components/auth/InactiveRemindModal.tsx"
    );
  });

  test("does not resolve AgentWindow (now in the React app)", () => {
    expect(resolveReactPagePath("AgentWindow")).toBeUndefined();
  });

  test("does not resolve SessionExpiredSurface (now in the React app)", () => {
    expect(resolveReactPagePath("SessionExpiredSurface")).toBeUndefined();
  });

  test("does not resolve auth test modules as runtime pages", () => {
    expect(resolveReactPagePath("InactiveRemindModal.test")).toBeUndefined();
  });
});
