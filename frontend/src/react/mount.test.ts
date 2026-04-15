import { describe, expect, test } from "vitest";
import { resolveReactPagePath } from "./mount";

describe("resolveReactPagePath", () => {
  test("resolves the session expired surface entry", () => {
    expect(resolveReactPagePath("SessionExpiredSurface")).toBe(
      "./components/auth/SessionExpiredSurface.tsx"
    );
  });

  test("does not resolve auth test modules as runtime pages", () => {
    expect(resolveReactPagePath("SessionExpiredSurface.test")).toBeUndefined();
  });
});
