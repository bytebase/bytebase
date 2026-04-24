import { describe, expect, test } from "vitest";

const sources = import.meta.glob(["./*.{ts,tsx}", "../BytebaseLogo.tsx"], {
  query: "?raw",
  import: "default",
  eager: true,
}) as Record<string, string>;

const appStateSources = import.meta.glob(
  ["../../hooks/useAppState.ts", "../../stores/app/**/*.ts"],
  {
    query: "?raw",
    import: "default",
    eager: true,
  }
) as Record<string, string>;

const forbidden = [
  "useVueState",
  "@/store",
  "@/components/Project/useRecentProjects",
  "hasWorkspacePermissionV2",
  "hasProjectPermissionV2",
  "@/router",
];

describe("React dashboard header dependencies", () => {
  test("does not directly depend on Vue, Pinia, or Vue Router", () => {
    const violations = [];
    for (const [file, source] of Object.entries(sources)) {
      if (file.endsWith(".test.ts") || file.endsWith(".test.tsx")) {
        continue;
      }
      for (const token of forbidden) {
        if (source.includes(token)) {
          violations.push(`${file}: ${token}`);
        }
      }
    }
    expect(violations).toEqual([]);
  });

  test("does not keep header-specific app state names", () => {
    const violations = [];
    for (const [file, source] of Object.entries(appStateSources)) {
      for (const token of [
        "HeaderAppState",
        "useHeaderAppStore",
        "useHeaderApp",
      ]) {
        if (source.includes(token)) {
          violations.push(`${file}: ${token}`);
        }
      }
    }
    expect(violations).toEqual([]);
  });
});
