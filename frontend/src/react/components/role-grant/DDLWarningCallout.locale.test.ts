import { describe, expect, test } from "vitest";
import enReact from "@/react/locales/en-US.json";

// Locale-format guard for the new role-grant warning strings. The component
// tests assert that interpolation reaches the rendered text via a mock t(),
// but they don't see the real locale strings — so a typo like "kind" instead
// of "{{kind}}" would slip through. This test loads en-US directly and
// asserts the placeholders survive translation, independent of any component.
//
// Other locales are kept in sync by frontend/scripts/check-react-i18n.mjs
// (cross-locale key parity) and locales without these placeholders would be
// caught by a localized smoke test, but this guard pins the EN source of
// truth so every consumer of `kind` / `environments` interpolation is safe.

describe("DDLWarningCallout locale strings", () => {
  test.each([
    ["project.members.ddl-warning", enReact.project.members["ddl-warning"]],
    [
      "project.members.ddl-current-all",
      enReact.project.members["ddl-current-all"],
    ],
    [
      "project.members.ddl-current-some",
      enReact.project.members["ddl-current-some"],
    ],
    [
      "project.members.ddl-current-none",
      enReact.project.members["ddl-current-none"],
    ],
  ])("%s contains {{kind}} placeholder", (_key, value) => {
    expect(value).toBeDefined();
    expect(value).toContain("{{kind}}");
  });

  test("issue.role-grant.ddl-warning contains both {{kind}} and {{environments}}", () => {
    const value = enReact.issue["role-grant"]["ddl-warning"];
    expect(value).toBeDefined();
    expect(value).toContain("{{kind}}");
    expect(value).toContain("{{environments}}");
  });
});
